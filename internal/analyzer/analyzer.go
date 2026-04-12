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
	RawLines   []ocr.Line
}

type Result struct {
	Measurement Measurement
	Candidates  []rocom.Candidate
}

type Service struct {
	ocr    *ocr.Recognizer
	engine *rocom.Engine
	topN   int
}

func NewService(recognizer *ocr.Recognizer, engine *rocom.Engine, topN int) *Service {
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

	anchors := findEggAnchors(lines)
	if len(anchors) == 0 {
		return nil, lines, fmt.Errorf("未识别到“神奇的蛋”标题")
	}

	results := make([]Result, 0, len(anchors))
	for idx, anchor := range anchors {
		windowLines := filterLinesForAnchor(lines, anchor, nextAnchorY(anchors, idx), len(anchors) == 1)
		measurement, ok := extractMeasurement(anchor.Text, windowLines)
		if !ok {
			crop := cropAroundAnchor(img, anchor, nextAnchorY(anchors, idx), len(anchors) == 1)
			cropLines, err := s.ocr.Recognize(ctx, scale(crop, 2))
			if err != nil {
				continue
			}
			measurement, ok = extractMeasurement(anchor.Text, cropLines)
			if !ok {
				continue
			}
		}
		results = append(results, Result{
			Measurement: measurement,
			Candidates:  s.engine.Search(measurement.Size, measurement.Weight, s.topN),
		})
	}

	if len(results) == 0 {
		return nil, lines, fmt.Errorf("识别到了蛋标题，但没有提取出有效的尺寸/重量")
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Measurement.Size < results[j].Measurement.Size
	})
	return dedupe(results), lines, nil
}

func findEggAnchors(lines []ocr.Line) []ocr.Line {
	anchors := make([]ocr.Line, 0)
	for _, line := range lines {
		text := normalizeText(line.Text)
		if strings.Contains(text, "神奇的蛋") || strings.Contains(text, "奇的蛋") {
			anchors = append(anchors, line)
		}
	}
	sort.Slice(anchors, func(i, j int) bool {
		return anchors[i].Y < anchors[j].Y
	})
	return anchors
}

func nextAnchorY(anchors []ocr.Line, current int) int {
	if current+1 >= len(anchors) {
		return math.MaxInt32
	}
	return anchors[current+1].Y
}

func filterLinesForAnchor(lines []ocr.Line, anchor ocr.Line, nextY int, single bool) []ocr.Line {
	xMin := max(0, anchor.X-120)
	xMax := anchor.X + 420
	yMin := anchor.Y + 35
	yMax := anchor.Y + 320
	if nextY != math.MaxInt32 {
		yMax = min(yMax, nextY-10)
	}
	if single {
		xMin = 0
		xMax = anchor.X + 520
		yMax = anchor.Y + 360
	}

	filtered := make([]ocr.Line, 0)
	for _, line := range lines {
		if line.X < xMin || line.X > xMax {
			continue
		}
		if line.Y < yMin || line.Y > yMax {
			continue
		}
		filtered = append(filtered, line)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Y < filtered[j].Y
	})
	return filtered
}

func cropAroundAnchor(img image.Image, anchor ocr.Line, nextY int, single bool) image.Image {
	bounds := img.Bounds()
	left := max(bounds.Min.X, anchor.X-80)
	top := max(bounds.Min.Y, anchor.Y+40)
	right := min(bounds.Max.X, anchor.X+420)
	bottom := min(bounds.Max.Y, anchor.Y+260)
	if nextY != math.MaxInt32 {
		bottom = min(bottom, nextY-10)
	}
	if single {
		left = bounds.Min.X
		right = bounds.Max.X
		bottom = min(bounds.Max.Y, anchor.Y+340)
	}

	rect := image.Rect(left, top, right, bottom)
	cropped := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			cropped.Set(x-rect.Min.X, y-rect.Min.Y, img.At(x, y))
		}
	}
	return cropped
}

func extractMeasurement(anchorText string, lines []ocr.Line) (Measurement, bool) {
	type candidate struct {
		value float64
		line  ocr.Line
		kind  string
	}

	values := make([]candidate, 0)
	for _, line := range lines {
		number, ok := parseNumber(line.Text)
		if !ok {
			continue
		}
		switch {
		case number > 0 && number <= 1.2:
			values = append(values, candidate{value: number, line: line, kind: "size"})
		case number >= 0.3 && number <= 100:
			values = append(values, candidate{value: number, line: line, kind: "weight"})
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].line.Y < values[j].line.Y
	})

	bestScore := math.MaxFloat64
	var best Measurement
	for idx, item := range values {
		if item.kind != "size" {
			continue
		}
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
				AnchorText: anchorText,
				RawLines:   lines,
			}
		}
	}
	if bestScore == math.MaxFloat64 {
		return Measurement{}, false
	}
	return best, true
}

func parseNumber(input string) (float64, bool) {
	normalized := normalizeText(input)
	match := numberRegex.FindString(normalized)
	if match == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, false
	}
	return value, true
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

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
