package rocom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"egg-analyze/internal/config"
)

type Dataset struct {
	Total     int        `json:"total"`
	TotalPets int        `json:"totalPets"`
	Groups    []EggGroup `json:"groups"`
}

type EggGroup struct {
	PetID      string        `json:"petId"`
	Pet        string        `json:"pet"`
	RangeItems []MeasureItem `json:"rangeItems"`
	ExactItems []MeasureItem `json:"exactItems"`
}

type MeasureItem struct {
	ID          int    `json:"id"`
	EggDiameter string `json:"eggDiameter"`
	EggWeight   string `json:"eggWeight"`
}

type SourceInfo struct {
	URL         string
	FromCache   bool
	UpdatedAt   time.Time
	GroupCount  int
	RecordCount int
}

func LoadDataset(ctx context.Context, url string) (*Dataset, SourceInfo, error) {
	cachePath, err := config.DatasetCachePath()
	if err != nil {
		return nil, SourceInfo{}, err
	}

	dataset, info, err := fetchDataset(ctx, url, cachePath)
	if err == nil {
		return dataset, info, nil
	}

	cached, cacheInfo, cacheErr := readCachedDataset(cachePath, url)
	if cacheErr == nil {
		return cached, cacheInfo, nil
	}

	return nil, SourceInfo{}, fmt.Errorf("load remote: %w; load cache: %v", err, cacheErr)
}

func fetchDataset(ctx context.Context, url, cachePath string) (*Dataset, SourceInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, SourceInfo{}, err
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, SourceInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, SourceInfo{}, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, SourceInfo{}, err
	}

	var dataset Dataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, SourceInfo{}, err
	}

	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return nil, SourceInfo{}, err
	}
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return nil, SourceInfo{}, err
	}

	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, SourceInfo{}, err
	}

	return &dataset, SourceInfo{
		URL:         url,
		FromCache:   false,
		UpdatedAt:   info.ModTime(),
		GroupCount:  len(dataset.Groups),
		RecordCount: dataset.Total,
	}, nil
}

func readCachedDataset(cachePath, url string) (*Dataset, SourceInfo, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, SourceInfo{}, err
	}

	var dataset Dataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, SourceInfo{}, err
	}

	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, SourceInfo{}, err
	}

	return &dataset, SourceInfo{
		URL:         url,
		FromCache:   true,
		UpdatedAt:   info.ModTime(),
		GroupCount:  len(dataset.Groups),
		RecordCount: dataset.Total,
	}, nil
}
