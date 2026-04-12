//go:build windows

package analyzer

import (
	"context"
	"image"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"egg-analyze/internal/config"
	"egg-analyze/internal/ocr"
	"egg-analyze/internal/rocom"
)

func TestAnalyzeDetailSample(t *testing.T) {
	service := buildService(t)
	img := loadImage(t, filepath.Join("..", "..", "picture", "蛋详细页面.png"))

	results, _, err := service.AnalyzeImage(context.Background(), img)
	if err != nil {
		t.Fatalf("AnalyzeImage failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected at least one result")
	}

	got := results[0].Measurement
	if math.Abs(got.Size-0.22) > 0.02 {
		t.Fatalf("unexpected size: got %.3f", got.Size)
	}
	if math.Abs(got.Weight-6.181) > 0.15 {
		t.Fatalf("unexpected weight: got %.3f", got.Weight)
	}
}

func TestAnalyzeIncubationCardSample(t *testing.T) {
	service := buildService(t)
	img := loadImage(t, filepath.Join("..", "..", "picture", "孵蛋页面3.png"))

	results, _, err := service.AnalyzeImage(context.Background(), img)
	if err != nil {
		t.Fatalf("AnalyzeImage failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected at least one result")
	}

	got := results[0].Measurement
	if math.Abs(got.Size-0.17) > 0.02 {
		t.Fatalf("unexpected size: got %.3f", got.Size)
	}
	if math.Abs(got.Weight-2.775) > 0.15 {
		t.Fatalf("unexpected weight: got %.3f", got.Weight)
	}
}

func buildService(t *testing.T) *Service {
	t.Helper()
	ctx := context.Background()
	dataset, _, err := rocom.LoadDataset(ctx, config.Default().DatasetURL)
	if err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	engine, err := rocom.NewEngine(dataset)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	return NewService(ocr.New(), engine, 5)
}

func loadImage(t *testing.T, path string) image.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	return img
}
