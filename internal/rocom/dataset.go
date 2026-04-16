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
	Pets []Pet
}

type Pet struct {
	ID            int           `json:"id"`
	Name          string        `json:"name"`
	Localized     LocalizedPet  `json:"localized"`
	Implemented   bool          `json:"implemented"`
	EvolvesFromID *int          `json:"evolves_from_id"`
	Breeding      *BreedingInfo `json:"breeding"`
}

type LocalizedPet struct {
	ZH LocalizedPetName `json:"zh"`
}

type LocalizedPetName struct {
	Name string `json:"name"`
}

type BreedingInfo struct {
	BreedingVariant
	Variants []BreedingVariant `json:"variants"`
}

type BreedingVariant struct {
	ID         *int     `json:"id"`
	PetID      *int     `json:"pet_id"`
	Name       string   `json:"name"`
	ModelID    *int     `json:"model_id"`
	HatchData  *int     `json:"hatch_data"`
	WeightLow  *float64 `json:"weight_low"`
	WeightHigh *float64 `json:"weight_high"`
	HeightLow  *float64 `json:"height_low"`
	HeightHigh *float64 `json:"height_high"`
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

	dataset, err := decodeDataset(data)
	if err != nil {
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

	return dataset, SourceInfo{
		URL:         url,
		FromCache:   false,
		UpdatedAt:   info.ModTime(),
		GroupCount:  len(dataset.Pets),
		RecordCount: len(dataset.Pets),
	}, nil
}

func readCachedDataset(cachePath, url string) (*Dataset, SourceInfo, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, SourceInfo{}, err
	}

	dataset, err := decodeDataset(data)
	if err != nil {
		return nil, SourceInfo{}, err
	}

	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, SourceInfo{}, err
	}

	return dataset, SourceInfo{
		URL:         url,
		FromCache:   true,
		UpdatedAt:   info.ModTime(),
		GroupCount:  len(dataset.Pets),
		RecordCount: len(dataset.Pets),
	}, nil
}

func decodeDataset(data []byte) (*Dataset, error) {
	var pets []Pet
	if err := json.Unmarshal(data, &pets); err != nil {
		return nil, err
	}
	return &Dataset{Pets: pets}, nil
}
