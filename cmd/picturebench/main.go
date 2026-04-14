package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	"egg-analyze/internal/analyzer"
	"egg-analyze/internal/ocr"
	"egg-analyze/internal/rocom"

	xdraw "golang.org/x/image/draw"
)

type manifestEntry struct {
	Image        string                `json:"image"`
	Case         string                `json:"case"`
	Scales       []float64             `json:"scales"`
	Measurements []manifestMeasurement `json:"measurements"`
}

type manifestMeasurement struct {
	Size   float64 `json:"size"`
	Weight float64 `json:"weight"`
}

func main() {
	data, err := os.ReadFile(filepath.Join(".", "picture", "manifest.json"))
	if err != nil {
		fatal(err)
	}

	var entries []manifestEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		fatal(err)
	}

	recognizer := ocr.New()
	priors := rocom.MeasurementPriors{
		Diameter: rocom.Range{Min: 0.03, Max: 1.2},
		Weight:   rocom.Range{Min: 0.03, Max: 300},
	}

	passed := 0
	total := 0
	for _, entry := range entries {
		scales := entry.Scales
		if len(scales) == 0 {
			scales = []float64{1}
		}
		for _, scale := range scales {
			total++
			ok, details := inspect(entry, scale, recognizer, priors)
			if ok {
				passed++
				fmt.Printf("PASS %s scale=%.2f %s\n", entry.Case, scale, filepath.Base(entry.Image))
				continue
			}
			fmt.Printf("FAIL %s scale=%.2f %s %s\n", entry.Case, scale, filepath.Base(entry.Image), details)
		}
	}

	fmt.Printf("\nOverall: %d/%d passed\n", passed, total)
}

func inspect(entry manifestEntry, scale float64, recognizer ocr.Recognizer, priors rocom.MeasurementPriors) (bool, string) {
	file, err := os.Open(entry.Image)
	if err != nil {
		return false, fmt.Sprintf("open: %v", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return false, fmt.Sprintf("decode: %v", err)
	}

	scaled := resizeImage(img, scale)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	measurements, _, err := analyzer.ExtractBestMeasurements(ctx, recognizer, scaled, priors)
	if err != nil {
		return false, fmt.Sprintf("extract: %v", err)
	}
	if len(measurements) != len(entry.Measurements) {
		return false, fmt.Sprintf("expected %d measurements got %d", len(entry.Measurements), len(measurements))
	}

	for idx, expected := range entry.Measurements {
		got := measurements[idx]
		if !closeEnough(got.Size, expected.Size) || !closeEnough(got.Weight, expected.Weight) {
			return false, fmt.Sprintf(
				"item %d expect=(%.3f,%.3f) got=(%.3f,%.3f)",
				idx+1, expected.Size, expected.Weight, got.Size, got.Weight,
			)
		}
	}
	return true, ""
}

func resizeImage(src image.Image, factor float64) image.Image {
	if math.Abs(factor-1) < 0.001 {
		return src
	}
	bounds := src.Bounds()
	width := maxInt(1, int(math.Round(float64(bounds.Dx())*factor)))
	height := maxInt(1, int(math.Round(float64(bounds.Dy())*factor)))
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, xdraw.Over, nil)
	return dst
}

func closeEnough(left, right float64) bool {
	return math.Abs(left-right) <= 0.002
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
