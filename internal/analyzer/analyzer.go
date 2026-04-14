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

var numberRegex = regexp.MustCompile(`(?:\d+\.\d+|\.\d+|\d+)`)

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
	if textRecognizer, ok := recognizer.(ocr.TextRecognizer); ok {
		measurements = append(measurements, rescueMeasurements(ctx, textRecognizer, img, lines, measurements, priors)...)
		measurements = dedupeMeasurements(measurements)
		sort.Slice(measurements, func(i, j int) bool {
			if absInt(measurements[i].AnchorY-measurements[j].AnchorY) <= 12 {
				return measurements[i].AnchorX < measurements[j].AnchorX
			}
			return measurements[i].AnchorY < measurements[j].AnchorY
		})
	}
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
	normalized := normalizeNumericText(input)
	match := numberRegex.FindString(normalized)
	if match == "" {
		return 0, "", false
	}
	if strings.HasPrefix(match, ".") {
		match = "0" + match
	}
	value, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, "", false
	}
	return value, match, true
}

func rescueMeasurements(ctx context.Context, recognizer ocr.TextRecognizer, img image.Image, lines []ocr.Line, existing []Measurement, priors rocom.MeasurementPriors) []Measurement {
	bounds := img.Bounds()
	matched := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		matched[measurementAnchorKey(item.AnchorX, item.AnchorY)] = struct{}{}
	}

	sortedLines := append([]ocr.Line(nil), lines...)
	sort.Slice(sortedLines, func(i, j int) bool {
		if absInt(sortedLines[i].Y-sortedLines[j].Y) <= rowTolerance(sortedLines[i], sortedLines[j]) {
			return sortedLines[i].X < sortedLines[j].X
		}
		return sortedLines[i].Y < sortedLines[j].Y
	})

	rescued := make([]Measurement, 0)
	for _, line := range sortedLines {
		if _, exists := matched[measurementAnchorKey(line.X, line.Y)]; exists {
			continue
		}

		size, normalized, ok := parseNumber(line.Text)
		if !ok || isIgnoredNumberLine(line.Text, normalized) {
			continue
		}
		if !inNumericRange(size, expandRange(priors.Diameter, 0.18, 0.02)) {
			continue
		}

		weight, ok := rescueWeightBelowAnchor(ctx, recognizer, img, line, sortedLines, priors.Weight)
		if !ok {
			continue
		}

		measurement := Measurement{
			Size:       size,
			Weight:     weight,
			AnchorText: line.Text,
			AnchorX:    clampInt(line.X, bounds.Min.X, bounds.Max.X),
			AnchorY:    clampInt(line.Y, bounds.Min.Y, bounds.Max.Y),
			RawLines:   lines,
		}
		rescued = append(rescued, measurement)
		matched[measurementAnchorKey(line.X, line.Y)] = struct{}{}
	}

	return rescued
}

func rescueWeightBelowAnchor(ctx context.Context, recognizer ocr.TextRecognizer, img image.Image, anchor ocr.Line, lines []ocr.Line, weightRange rocom.Range) (float64, bool) {
	bestScore := math.Inf(-1)
	bestValue := 0.0
	for _, crop := range buildRescueCrops(img, anchor, lines) {
		for _, prepared := range prepareRescueImages(crop) {
			textLines, err := recognizer.RecognizeText(ctx, prepared)
			if err != nil {
				continue
			}
			value, score, ok := parseBestRescuedWeight(textLines, weightRange)
			if ok && score > bestScore {
				bestScore = score
				bestValue = value
			}
		}
	}
	return bestValue, !math.IsInf(bestScore, -1)
}

func buildRescueCrops(img image.Image, anchor ocr.Line, lines []ocr.Line) []image.Image {
	crops := make([]image.Image, 0, 3)
	if target, ok := findRescueTarget(anchor, lines); ok {
		if crop, ok := cropLineWithPadding(img, target, 18, 12); ok {
			crops = append(crops, crop)
		}
	}
	if crop, ok := cropWeightBelowAnchor(img, anchor, 0.10, 1.20, 1.30, 1.55); ok {
		crops = append(crops, crop)
	}
	if crop, ok := cropWeightBelowAnchor(img, anchor, 0.00, 1.12, 1.18, 1.70); ok {
		crops = append(crops, crop)
	}
	return crops
}

func cropWeightBelowAnchor(src image.Image, anchor ocr.Line, leftPadRatio, topOffsetRatio, widthRatio, heightRatio float64) (image.Image, bool) {
	left := anchor.X - int(math.Round(float64(anchor.Width)*leftPadRatio))
	top := anchor.Y + int(math.Round(float64(anchor.Height)*topOffsetRatio))
	right := anchor.X + int(math.Round(float64(anchor.Width)*widthRatio))
	bottom := top + int(math.Round(float64(anchor.Height)*heightRatio))
	return cropRect(src, image.Rect(left, top, right, bottom))
}

func prepareRescueImages(src image.Image) []image.Image {
	images := []image.Image{src}
	height := src.Bounds().Dy()
	switch {
	case height > 0 && height < 28:
		images = append(images, scale(src, 3))
	case height >= 28 && height < 64:
		images = append(images, scale(src, 2))
	}
	return images
}

func parseBestRescuedWeight(lines []ocr.Line, weightRange rocom.Range) (float64, float64, bool) {
	bestScore := -1.0
	bestValue := 0.0
	expanded := expandRange(weightRange, 0.25, 0.03)
	for _, line := range lines {
		value, normalized, ok := parseNumber(line.Text)
		if !ok || isIgnoredNumberLine(line.Text, normalized) {
			continue
		}
		if !inNumericRange(value, expanded) {
			continue
		}
		score := plausibilityScore(value, weightRange, 0.12) + rescuedPrecisionScore(normalized)
		if score > bestScore {
			bestScore = score
			bestValue = value
		}
	}
	return bestValue, bestScore, bestScore > 0
}

func rescuedPrecisionScore(normalized string) float64 {
	decimalDigits := digitsAfterDecimal(normalized)
	score := float64(minInt(decimalDigits, 3)) * 3
	if decimalDigits >= 3 {
		score += 1
	}
	return score
}

func digitsAfterDecimal(value string) int {
	index := strings.IndexByte(value, '.')
	if index == -1 || index == len(value)-1 {
		return 0
	}
	count := 0
	for _, r := range value[index+1:] {
		if r < '0' || r > '9' {
			break
		}
		count++
	}
	return count
}

func findRescueTarget(anchor ocr.Line, lines []ocr.Line) (ocr.Line, bool) {
	bestScore := math.Inf(-1)
	var best ocr.Line
	for _, candidate := range lines {
		if candidate.X == anchor.X && candidate.Y == anchor.Y && candidate.Width == anchor.Width && candidate.Height == anchor.Height {
			continue
		}

		dy := candidate.Y - anchor.Y
		dx := candidate.X - anchor.X
		sameColumn := absInt(dx) <= maxInt(anchor.Width, candidate.Width)
		sameRow := absInt(dy) <= rowTolerance(anchor, candidate)

		score := math.Inf(-1)
		switch {
		case sameColumn && dy > 0 && dy <= maxInt(180, anchor.Height*4):
			score = 220 - float64(dy) - float64(absInt(dx))*0.8
		case sameRow && dx > 0 && dx <= maxInt(260, anchor.Width*5):
			score = 180 - float64(dx) - float64(absInt(dy))*1.2
		default:
			continue
		}

		if score > bestScore {
			bestScore = score
			best = candidate
		}
	}
	return best, !math.IsInf(bestScore, -1)
}

func cropLineWithPadding(src image.Image, line ocr.Line, padX, padY int) (image.Image, bool) {
	return cropRect(src, image.Rect(
		line.X-padX,
		line.Y-padY,
		line.X+maxInt(line.Width, 1)+padX,
		line.Y+maxInt(line.Height, 1)+padY,
	))
}

func cropRect(src image.Image, rect image.Rectangle) (image.Image, bool) {
	bounds := src.Bounds()
	left := clampInt(rect.Min.X, bounds.Min.X, bounds.Max.X)
	top := clampInt(rect.Min.Y, bounds.Min.Y, bounds.Max.Y)
	right := clampInt(rect.Max.X, bounds.Min.X, bounds.Max.X)
	bottom := clampInt(rect.Max.Y, bounds.Min.Y, bounds.Max.Y)
	if right-left < 8 || bottom-top < 8 {
		return nil, false
	}
	dst := image.NewRGBA(image.Rect(0, 0, right-left, bottom-top))
	for y := top; y < bottom; y++ {
		for x := left; x < right; x++ {
			dst.Set(x-left, y-top, src.At(x, y))
		}
	}
	return dst, true
}

func measurementAnchorKey(x, y int) string {
	return fmt.Sprintf("%d:%d", x, y)
}

func normalizeNumericText(input string) string {
	normalized := strings.NewReplacer(
		" ", "",
		"　", "",
		"．", ".",
		"。", ".",
		"·", ".",
		",", ".",
	).Replace(input)

	runes := []rune(normalized)
	for idx, current := range runes {
		mapped, ok := mapDigitLikeRune(current)
		if !ok {
			continue
		}
		if !isNumericContextRune(runes, idx) {
			continue
		}
		runes[idx] = mapped
	}
	return string(runes)
}

func mapDigitLikeRune(value rune) (rune, bool) {
	switch value {
	case 'O', 'o', 'Q', 'D', '□', '口', '〇':
		return '0', true
	case 'I', 'i', 'l', '|', '!':
		return '1', true
	case 'Z', 'z':
		return '2', true
	case 'S', 's':
		return '5', true
	default:
		return 0, false
	}
}

func isNumericContextRune(values []rune, index int) bool {
	leftNumeric := index > 0 && isDigitOrDot(values[index-1])
	rightNumeric := index+1 < len(values) && isDigitOrDot(values[index+1])
	return leftNumeric || rightNumeric
}

func isDigitOrDot(value rune) bool {
	return (value >= '0' && value <= '9') || value == '.'
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

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
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

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
