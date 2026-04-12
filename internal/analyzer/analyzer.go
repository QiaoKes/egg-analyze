package analyzer

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"egg-analyze/internal/ocr"
	"egg-analyze/internal/rocom"

	xdraw "golang.org/x/image/draw"
)

var numberRegex = regexp.MustCompile(`\d+(?:\.\d+)?`)

type Measurement struct {
	Size       float64
	Weight     float64
	AnchorText string
	AnchorX    int
	AnchorY    int
	RawLines   []ocr.Line
}

type Result struct {
	Measurement Measurement
	Candidates  []rocom.Candidate
}

type extractionCandidate struct {
	value      float64
	line       ocr.Line
	normalized string
	order      int
}

type Service struct {
	ocr    ocr.Recognizer
	engine *rocom.Engine
	priors rocom.MeasurementPriors
	topN   int
}

func NewService(recognizer ocr.Recognizer, engine *rocom.Engine, topN int) *Service {
	return &Service{
		ocr:    recognizer,
		engine: engine,
		priors: engine.MeasurementPriors(),
		topN:   topN,
	}
}

func (s *Service) AnalyzeImage(ctx context.Context, img image.Image) ([]Result, []ocr.Line, error) {
	measurements, lines, err := ExtractBestMeasurements(ctx, s.ocr, img, s.priors)
	if err != nil {
		return nil, nil, err
	}
	results := buildResults(s.engine, measurements, s.topN)
	return dedupe(results), lines, nil
}

func ExtractBestMeasurements(ctx context.Context, recognizer ocr.Recognizer, img image.Image, priors rocom.MeasurementPriors) ([]Measurement, []ocr.Line, error) {
	prepared := prepareOCRImage(img)
	lines, err := recognizer.Recognize(ctx, prepared)
	if err != nil {
		return nil, nil, err
	}
	measurements := ExtractMeasurementsWithPriors(lines, priors)
	if len(measurements) == 0 {
		return nil, lines, fmt.Errorf("没有提取出有效的尺寸/重量")
	}
	return measurements, lines, nil
}

func buildResults(engine *rocom.Engine, measurements []Measurement, topN int) []Result {
	results := make([]Result, 0, len(measurements))
	for _, measurement := range measurements {
		results = append(results, Result{
			Measurement: measurement,
			Candidates:  engine.Search(measurement.Size, measurement.Weight, topN),
		})
	}
	sort.Slice(results, func(i, j int) bool {
		left := results[i].Measurement
		right := results[j].Measurement
		if absInt(left.AnchorY-right.AnchorY) <= 12 {
			return left.AnchorX < right.AnchorX
		}
		return left.AnchorY < right.AnchorY
	})
	return results
}

func ExtractMeasurements(lines []ocr.Line) []Measurement {
	return ExtractMeasurementsWithPriors(lines, defaultMeasurementPriors())
}

func ExtractMeasurementsWithPriors(lines []ocr.Line, priors rocom.MeasurementPriors) []Measurement {
	type pair struct {
		left  extractionCandidate
		right extractionCandidate
		score float64
	}

	sortedLines := append([]ocr.Line(nil), lines...)
	sort.Slice(sortedLines, func(i, j int) bool {
		if absInt(sortedLines[i].Y-sortedLines[j].Y) <= rowTolerance(sortedLines[i], sortedLines[j]) {
			return sortedLines[i].X < sortedLines[j].X
		}
		return sortedLines[i].Y < sortedLines[j].Y
	})

	values := make([]extractionCandidate, 0, len(sortedLines))
	for idx, line := range sortedLines {
		number, normalized, ok := parseNumber(line.Text)
		if !ok || isIgnoredNumberLine(line.Text, normalized) {
			continue
		}
		if !inNumericRange(number, expandRange(unionRange(priors.Diameter, priors.Weight), 0.35, 1)) {
			continue
		}
		values = append(values, extractionCandidate{
			value:      number,
			line:       line,
			normalized: normalized,
			order:      idx,
		})
	}

	pairs := make([]pair, 0)
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			score, ok := scorePair(values[i], values[j], priors)
			if !ok {
				continue
			}
			pairs = append(pairs, pair{left: values[i], right: values[j], score: score})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		if math.Abs(pairs[i].score-pairs[j].score) < 0.001 {
			if pairs[i].left.order == pairs[j].left.order {
				return pairs[i].right.order < pairs[j].right.order
			}
			return pairs[i].left.order < pairs[j].left.order
		}
		return pairs[i].score > pairs[j].score
	})

	used := make(map[int]struct{})
	results := make([]Measurement, 0)
	for _, item := range pairs {
		if _, exists := used[item.left.order]; exists {
			continue
		}
		if _, exists := used[item.right.order]; exists {
			continue
		}
		used[item.left.order] = struct{}{}
		used[item.right.order] = struct{}{}
		results = append(results, Measurement{
			Size:       item.left.value,
			Weight:     item.right.value,
			AnchorText: item.left.line.Text,
			AnchorX:    item.left.line.X,
			AnchorY:    item.left.line.Y,
			RawLines:   lines,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if absInt(results[i].AnchorY-results[j].AnchorY) <= 12 {
			return results[i].AnchorX < results[j].AnchorX
		}
		return results[i].AnchorY < results[j].AnchorY
	})
	return dedupeMeasurements(results)
}

func parseNumber(input string) (float64, string, bool) {
	normalized := normalizeText(input)
	match := numberRegex.FindString(normalized)
	if match == "" {
		return 0, "", false
	}
	value, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, "", false
	}
	return value, match, true
}

func normalizeText(input string) string {
	replacer := strings.NewReplacer(
		" ", "",
		"　", "",
		"．", ".",
		"。", ".",
		"·", ".",
		"O", "0",
		"o", "0",
		"Q", "0",
		"D", "0",
		"□", "0",
		"口", "0",
		"〇", "0",
		"I", "1",
		"l", "1",
		"|", "1",
		"S", "5",
		"s", "5",
		"B", "8",
		",", ".",
	)
	return replacer.Replace(input)
}

func scale(src image.Image, factor int) image.Image {
	if factor <= 1 {
		return src
	}
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx()*factor, bounds.Dy()*factor))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, xdraw.Over, nil)
	return dst
}

func prepareOCRImage(img image.Image) image.Image {
	return img
}

func grayscale(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(x, y, color.GrayModel.Convert(src.At(x, y)))
		}
	}
	return dst
}

func autocontrast(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewGray(bounds)
	minValue := uint8(255)
	maxValue := uint8(0)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			v := color.GrayModel.Convert(src.At(x, y)).(color.Gray).Y
			if v < minValue {
				minValue = v
			}
			if v > maxValue {
				maxValue = v
			}
		}
	}
	if maxValue <= minValue {
		return src
	}
	scaleFactor := 255.0 / float64(maxValue-minValue)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			v := color.GrayModel.Convert(src.At(x, y)).(color.Gray).Y
			adjusted := uint8(math.Round(float64(v-minValue) * scaleFactor))
			dst.SetGray(x, y, color.Gray{Y: adjusted})
		}
	}
	return dst
}

func dedupe(results []Result) []Result {
	seen := make(map[string]struct{})
	filtered := make([]Result, 0, len(results))
	for _, result := range results {
		key := fmt.Sprintf("%.3f-%.3f", result.Measurement.Size, result.Measurement.Weight)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, result)
	}
	return filtered
}

func dedupeMeasurements(values []Measurement) []Measurement {
	seen := make(map[string]struct{})
	filtered := make([]Measurement, 0, len(values))
	for _, value := range values {
		key := fmt.Sprintf("%.3f-%.3f", value.Size, value.Weight)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, value)
	}
	return filtered
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func defaultMeasurementPriors() rocom.MeasurementPriors {
	return rocom.MeasurementPriors{
		Diameter: rocom.Range{Min: 0.03, Max: 1.2},
		Weight:   rocom.Range{Min: 0.03, Max: 300},
	}
}

func unionRange(left, right rocom.Range) rocom.Range {
	return rocom.Range{
		Min: math.Min(left.Min, right.Min),
		Max: math.Max(left.Max, right.Max),
	}
}

func isIgnoredNumberLine(raw, normalized string) bool {
	trimmed := strings.TrimSpace(normalized)
	if trimmed == "" {
		return true
	}
	if strings.Contains(raw, "%") {
		return true
	}
	if matched, _ := regexp.MatchString(`^\d{4}[-/.]\d{1,2}[-/.]\d{1,2}$`, trimmed); matched {
		return true
	}
	return !strings.Contains(trimmed, ".")
}

func rowTolerance(left, right ocr.Line) int {
	height := maxInt(left.Height, right.Height)
	return maxInt(12, height/2)
}

func scorePair(left, right extractionCandidate, priors rocom.MeasurementPriors) (float64, bool) {
	dy := absInt(right.line.Y - left.line.Y)
	dx := right.line.X - left.line.X
	sameRow := dy <= rowTolerance(left.line, right.line)
	sameColumn := absInt(right.line.X-left.line.X) <= maxInt(left.line.Width, right.line.Width)*2

	switch {
	case sameRow:
		if dx <= 0 || dx > 520 {
			return 0, false
		}
	case sameColumn:
		if dy <= 0 || dy > 220 {
			return 0, false
		}
	default:
		return 0, false
	}

	score := 0.0
	score += plausibilityScore(left.value, priors.Diameter, 0.18)
	score += plausibilityScore(right.value, priors.Weight, 0.18)
	if score <= 40 {
		return 0, false
	}

	if sameRow {
		score += 140
		score += 60 * closenessScore(float64(dx), 520)
		score += 30 * closenessScore(float64(dy), float64(maxInt(16, rowTolerance(left.line, right.line))))
	} else {
		score += 90
		score += 50 * closenessScore(float64(dy), 220)
		score += 20 * closenessScore(float64(absInt(dx)), float64(maxInt(80, maxInt(left.line.Width, right.line.Width)*2)))
	}

	gap := right.order - left.order - 1
	if gap > 0 {
		score -= float64(gap) * 18
	}

	if strings.Count(left.normalized, ".") == 1 {
		score += 8
	}
	if strings.Count(right.normalized, ".") == 1 {
		score += 8
	}

	return score, true
}

func plausibilityScore(value float64, expected rocom.Range, margin float64) float64 {
	expanded := expandRange(expected, margin, 0.02)
	if !inNumericRange(value, expanded) {
		return -120
	}
	if inNumericRange(value, expected) {
		return 80 + 20*closenessScore(distanceToCenter(value, expected), rangeHalfWidth(expected)+0.01)
	}
	return 45 + 15*closenessScore(distanceToRangeValue(value, expected), expanded.Max-expanded.Min)
}

func expandRange(value rocom.Range, ratio, floor float64) rocom.Range {
	span := value.Max - value.Min
	margin := math.Max(span*ratio, floor)
	return rocom.Range{
		Min: math.Max(0, value.Min-margin),
		Max: value.Max + margin,
	}
}

func inNumericRange(value float64, r rocom.Range) bool {
	return value >= r.Min && value <= r.Max
}

func distanceToCenter(value float64, r rocom.Range) float64 {
	center := (r.Min + r.Max) / 2
	return math.Abs(value - center)
}

func rangeHalfWidth(r rocom.Range) float64 {
	return math.Max((r.Max-r.Min)/2, 0.001)
}

func distanceToRangeValue(value float64, r rocom.Range) float64 {
	if value < r.Min {
		return r.Min - value
	}
	if value > r.Max {
		return value - r.Max
	}
	return 0
}

func closenessScore(distance, scale float64) float64 {
	if scale <= 0 {
		return 0
	}
	score := 1 - distance/scale
	if score < 0 {
		return 0
	}
	return score
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
