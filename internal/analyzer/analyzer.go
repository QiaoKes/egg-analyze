package analyzer

import (
	"context"
	"fmt"
	"image"
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

type Service struct {
	ocr    ocr.Recognizer
	engine *rocom.Engine
	topN   int
}

func NewService(recognizer ocr.Recognizer, engine *rocom.Engine, topN int) *Service {
	return &Service{
		ocr:    recognizer,
		engine: engine,
		topN:   topN,
	}
}

func (s *Service) AnalyzeImage(ctx context.Context, img image.Image) ([]Result, []ocr.Line, error) {
	lines, err := s.ocr.Recognize(ctx, img)
	if err != nil {
		return nil, nil, err
	}

	results := buildResults(s.engine, ExtractMeasurements(lines), s.topN)
	if len(results) == 0 {
		scaledLines, scaledErr := s.ocr.Recognize(ctx, scale(img, 2))
		if scaledErr == nil {
			lines = scaledLines
			results = buildResults(s.engine, ExtractMeasurements(lines), s.topN)
		}
	}

	if len(results) == 0 {
		return nil, lines, fmt.Errorf("没有提取出有效的尺寸/重量")
	}

	return dedupe(results), lines, nil
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
	type candidate struct {
		value      float64
		line       ocr.Line
		normalized string
		kind       string
	}

	values := make([]candidate, 0)
	for _, line := range lines {
		number, normalized, ok := parseNumber(line.Text)
		if !ok {
			continue
		}
		switch {
		case isLikelySize(number, normalized):
			values = append(values, candidate{value: number, line: line, normalized: normalized, kind: "size"})
		case isLikelyWeight(number, normalized):
			values = append(values, candidate{value: number, line: line, normalized: normalized, kind: "weight"})
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].line.Y < values[j].line.Y
	})

	results := make([]Measurement, 0)
	for idx, item := range values {
		if item.kind != "size" {
			continue
		}
		bestScore := math.MaxFloat64
		var best Measurement
		for next := idx + 1; next < len(values); next++ {
			other := values[next]
			if other.kind != "weight" {
				continue
			}
			dy := math.Abs(float64(other.line.Y - item.line.Y))
			if dy > 120 {
				continue
			}
			dx := math.Abs(float64(other.line.X - item.line.X))
			if dx > 280 {
				continue
			}
			sameRow := dy <= 18
			if sameRow && other.line.X <= item.line.X {
				continue
			}
			score := dy*2 + dx
			if other.line.Y < item.line.Y {
				score += 80
			}
			if score >= bestScore {
				continue
			}
			bestScore = score
			best = Measurement{
				Size:       item.value,
				Weight:     other.value,
				AnchorText: item.line.Text,
				AnchorX:    item.line.X,
				AnchorY:    item.line.Y,
				RawLines:   lines,
			}
		}
		if bestScore == math.MaxFloat64 {
			continue
		}
		results = append(results, best)
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

func isLikelySize(number float64, raw string) bool {
	return number > 0 && number < 1 && strings.Contains(raw, ".")
}

func isLikelyWeight(number float64, raw string) bool {
	if number < 0.5 || number > 100 {
		return false
	}
	return strings.Contains(raw, ".")
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
