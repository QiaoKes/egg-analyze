package main

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"egg-analyze/internal/analyzer"
	"egg-analyze/internal/ocr"
	"egg-analyze/internal/rocom"
)

func main() {
	root := filepath.Join(".", "picture")
	entries, err := os.ReadDir(root)
	if err != nil {
		fatal(err)
	}

	recognizer := ocr.New()
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "synthetic_debug_") {
			continue
		}
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".png") && !strings.HasSuffix(lower, ".jpg") && !strings.HasSuffix(lower, ".jpeg") {
			continue
		}
		files = append(files, filepath.Join(root, name))
	}
	sort.Strings(files)

	fmt.Printf("OCR: %s\n", recognizer.Name())
	for _, path := range files {
		start := time.Now()
		measurements, lines, err := inspectFile(path, recognizer)
		elapsed := time.Since(start).Round(time.Millisecond)
		fmt.Printf("\n[%s] %s\n", elapsed, filepath.Base(path))
		if err != nil {
			fmt.Printf("  ERROR: %v\n", err)
			continue
		}
		if len(measurements) == 0 {
			fmt.Println("  no measurements")
			printLines(lines)
			continue
		}
		for idx, item := range measurements {
			fmt.Printf("  #%d size=%.3f weight=%.3f anchor=(%d,%d) text=%q\n",
				idx+1, item.Size, item.Weight, item.AnchorX, item.AnchorY, item.AnchorText)
		}
	}
}

func inspectFile(path string, recognizer ocr.Recognizer) ([]analyzer.Measurement, []ocr.Line, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	measurements, lines, err := analyzer.ExtractBestMeasurements(ctx, recognizer, img, rocom.MeasurementPriors{
		Diameter: rocom.Range{Min: 0.03, Max: 1.2},
		Weight:   rocom.Range{Min: 0.03, Max: 300},
	})
	if err != nil {
		return nil, nil, err
	}
	return measurements, lines, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func printLines(lines []ocr.Line) {
	if len(lines) == 0 {
		fmt.Println("  raw ocr: none")
		return
	}
	fmt.Println("  raw ocr:")
	for _, line := range lines {
		fmt.Printf("    (%d,%d %dx%d) %q\n", line.X, line.Y, line.Width, line.Height, line.Text)
	}
}
