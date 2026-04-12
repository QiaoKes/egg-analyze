package analyzer

import (
	"math"
	"testing"

	"egg-analyze/internal/ocr"
	"egg-analyze/internal/rocom"
)

func TestExtractMeasurementsWithoutTitleAnchor(t *testing.T) {
	lines := []ocr.Line{
		{Text: "完成", X: 220, Y: 100, Width: 60, Height: 20},
		{Text: "0.17", X: 220, Y: 140, Width: 60, Height: 20},
		{Text: "2.775", X: 220, Y: 190, Width: 80, Height: 20},
		{Text: "56%", X: 220, Y: 280, Width: 60, Height: 20},
		{Text: "0.23", X: 220, Y: 320, Width: 60, Height: 20},
		{Text: "2.75", X: 220, Y: 370, Width: 70, Height: 20},
	}

	results := ExtractMeasurements(lines)
	if len(results) != 2 {
		t.Fatalf("expected 2 measurements, got %d", len(results))
	}

	assertMeasurement(t, results[0], 0.17, 2.775)
	assertMeasurement(t, results[1], 0.23, 2.75)
}

func TestExtractMeasurementsIgnoresDateAndPercent(t *testing.T) {
	lines := []ocr.Line{
		{Text: "2026-04-12", X: 300, Y: 50, Width: 140, Height: 20},
		{Text: "55%", X: 220, Y: 100, Width: 60, Height: 20},
		{Text: "0.22", X: 60, Y: 120, Width: 60, Height: 20},
		{Text: "6.181", X: 180, Y: 120, Width: 80, Height: 20},
	}

	results := ExtractMeasurements(lines)
	if len(results) != 1 {
		t.Fatalf("expected 1 measurement, got %d", len(results))
	}
	assertMeasurement(t, results[0], 0.22, 6.181)
}

func TestExtractMeasurementsKeepsScreenOrder(t *testing.T) {
	lines := []ocr.Line{
		{Text: "0.20", X: 120, Y: 80, Width: 60, Height: 20},
		{Text: "11.052", X: 120, Y: 120, Width: 90, Height: 20},
		{Text: "0.17", X: 120, Y: 260, Width: 60, Height: 20},
		{Text: "2.775", X: 120, Y: 300, Width: 80, Height: 20},
		{Text: "0.23", X: 120, Y: 440, Width: 60, Height: 20},
		{Text: "2.75", X: 120, Y: 480, Width: 70, Height: 20},
	}

	results := ExtractMeasurements(lines)
	if len(results) != 3 {
		t.Fatalf("expected 3 measurements, got %d", len(results))
	}

	assertMeasurement(t, results[0], 0.20, 11.052)
	assertMeasurement(t, results[1], 0.17, 2.775)
	assertMeasurement(t, results[2], 0.23, 2.75)
}

func TestExtractMeasurementsAcceptsLowWeightFromDatasetRange(t *testing.T) {
	lines := []ocr.Line{
		{Text: "神奇的蛋", X: 80, Y: 61, Width: 153, Height: 36},
		{Text: "0.21", X: 81, Y: 253, Width: 84, Height: 25},
		{Text: "0.309<", X: 273, Y: 253, Width: 109, Height: 25},
		{Text: "2026-04-12", X: 427, Y: 254, Width: 147, Height: 23},
	}

	results := ExtractMeasurementsWithPriors(lines, rocom.MeasurementPriors{
		Diameter: rocom.Range{Min: 0.04, Max: 1.1},
		Weight:   rocom.Range{Min: 0.03, Max: 280},
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 measurement, got %d", len(results))
	}
	assertMeasurement(t, results[0], 0.21, 0.309)
}

func TestExtractMeasurementsUsesReadingOrderWithinRow(t *testing.T) {
	lines := []ocr.Line{
		{Text: "0.21", X: 81, Y: 266, Width: 84, Height: 25},
		{Text: "5.04<×", X: 249, Y: 265, Width: 96, Height: 26},
		{Text: "2026-04-12", X: 407, Y: 269, Width: 142, Height: 18},
	}

	results := ExtractMeasurementsWithPriors(lines, rocom.MeasurementPriors{
		Diameter: rocom.Range{Min: 0.04, Max: 1.1},
		Weight:   rocom.Range{Min: 0.03, Max: 280},
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 measurement, got %d", len(results))
	}
	assertMeasurement(t, results[0], 0.21, 5.04)
}

func assertMeasurement(t *testing.T, got Measurement, wantSize, wantWeight float64) {
	t.Helper()
	if math.Abs(got.Size-wantSize) > 0.001 {
		t.Fatalf("unexpected size: got %.3f want %.3f", got.Size, wantSize)
	}
	if math.Abs(got.Weight-wantWeight) > 0.001 {
		t.Fatalf("unexpected weight: got %.3f want %.3f", got.Weight, wantWeight)
	}
}
