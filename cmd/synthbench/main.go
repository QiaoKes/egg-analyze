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
	"sort"
	"time"

	"egg-analyze/internal/analyzer"
	"egg-analyze/internal/config"
	"egg-analyze/internal/ocr"
	"egg-analyze/internal/rocom"
)

type manifestEntry struct {
	Image    string  `json:"image"`
	Template string  `json:"template"`
	Case     string  `json:"case"`
	Size     float64 `json:"size"`
	Weight   float64 `json:"weight"`
}

type templateStats struct {
	Total int
	Pass  int
}

func main() {
	manifestPath := filepath.Join(".", "picture", "synthetic", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		fatal(err)
	}

	var entries []manifestEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := config.Default()
	dataset, _, err := rocom.LoadDataset(ctx, cfg.DatasetURL)
	if err != nil {
		fatal(err)
	}
	engine, err := rocom.NewEngine(dataset)
	if err != nil {
		fatal(err)
	}

	recognizer := ocr.New()
	stats := make(map[string]*templateStats)
	passed := 0

	for _, entry := range entries {
		ok, got, err := inspect(entry, recognizer, engine.MeasurementPriors())
		if _, exists := stats[entry.Template]; !exists {
			stats[entry.Template] = &templateStats{}
		}
		stats[entry.Template].Total++
		if ok {
			passed++
			stats[entry.Template].Pass++
			continue
		}

		if err != nil {
			fmt.Printf("FAIL %s/%s error=%v\n", entry.Template, entry.Case, err)
			continue
		}
		fmt.Printf(
			"FAIL %s/%s expect=(%.3f, %.3f) got=(%.3f, %.3f)\n",
			entry.Template, entry.Case, entry.Size, entry.Weight, got.Size, got.Weight,
		)
	}

	keys := make([]string, 0, len(stats))
	for key := range stats {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fmt.Printf("\nOverall: %d/%d passed\n", passed, len(entries))
	for _, key := range keys {
		item := stats[key]
		fmt.Printf("%s: %d/%d passed\n", key, item.Pass, item.Total)
	}
}

func inspect(entry manifestEntry, recognizer ocr.Recognizer, priors rocom.MeasurementPriors) (bool, analyzer.Measurement, error) {
	file, err := os.Open(entry.Image)
	if err != nil {
		return false, analyzer.Measurement{}, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return false, analyzer.Measurement{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	measurements, _, err := analyzer.ExtractBestMeasurements(ctx, recognizer, img, priors)
	if err != nil {
		return false, analyzer.Measurement{}, err
	}
	if len(measurements) != 1 {
		return false, analyzer.Measurement{}, fmt.Errorf("expected 1 measurement, got %d", len(measurements))
	}
	got := measurements[0]
	return closeEnough(got.Size, entry.Size) && closeEnough(got.Weight, entry.Weight), got, nil
}

func closeEnough(left, right float64) bool {
	return math.Abs(left-right) <= 0.001
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
