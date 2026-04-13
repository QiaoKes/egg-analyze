package atlas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"egg-analyze/internal/config"

	"golang.org/x/image/webp"
)

const (
	atlasBaseURL  = "https://rocom.mfsky.qzz.io/creature-atlas"
	masterListURL = "https://rocom.mfsky.qzz.io/data/creatures-master-list.json"
)

type masterList struct {
	Creatures []creature `json:"creatures"`
}

type creature struct {
	ID     string        `json:"id"`
	Images creatureImage `json:"images"`
}

type creatureImage struct {
	Default string `json:"default"`
}

type indexFetch struct {
	done  chan struct{}
	index map[string]string
	err   error
}

type imageLoad struct {
	done chan struct{}
	img  image.Image
	err  error
}

type Service struct {
	client *http.Client

	mu         sync.RWMutex
	index      map[string]string
	indexFetch *indexFetch
	decoded    map[string]image.Image
	loading    map[string]*imageLoad
}

func New() *Service {
	return &Service{
		client:  &http.Client{Timeout: 8 * time.Second},
		index:   make(map[string]string),
		decoded: make(map[string]image.Image),
		loading: make(map[string]*imageLoad),
	}
}

func (s *Service) Load(ctx context.Context, petID string) (image.Image, error) {
	normalizedID := normalizePetID(petID)
	if normalizedID == "" {
		return nil, fmt.Errorf("empty pet id")
	}

	s.mu.RLock()
	if cached, ok := s.decoded[normalizedID]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	if call, ok := s.loading[normalizedID]; ok {
		s.mu.Unlock()
		return waitImageLoad(ctx, call)
	}
	call := &imageLoad{done: make(chan struct{})}
	s.loading[normalizedID] = call
	s.mu.Unlock()

	img, err := s.loadUncached(ctx, normalizedID)

	s.mu.Lock()
	if err == nil {
		s.decoded[normalizedID] = img
	}
	call.img = img
	call.err = err
	close(call.done)
	delete(s.loading, normalizedID)
	s.mu.Unlock()

	return img, err
}

func (s *Service) loadUncached(ctx context.Context, normalizedID string) (image.Image, error) {
	fileName, ok, err := s.resolveFileName(ctx, normalizedID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, os.ErrNotExist
	}

	data, err := s.readOrDownload(ctx, fileName)
	if err != nil {
		return nil, err
	}

	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", fileName, err)
	}
	return img, nil
}

func waitImageLoad(ctx context.Context, call *imageLoad) (image.Image, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.done:
		return call.img, call.err
	}
}

func (s *Service) resolveFileName(ctx context.Context, petID string) (string, bool, error) {
	s.mu.RLock()
	fileName, ok := s.index[petID]
	s.mu.RUnlock()
	if ok {
		return fileName, fileName != "", nil
	}

	index, err := s.getIndex(ctx)
	if err != nil {
		return "", false, err
	}
	fileName, ok = index[petID]
	return fileName, ok && fileName != "", nil
}

func (s *Service) getIndex(ctx context.Context) (map[string]string, error) {
	s.mu.Lock()
	if len(s.index) > 0 {
		index := s.index
		s.mu.Unlock()
		return index, nil
	}
	if call := s.indexFetch; call != nil {
		s.mu.Unlock()
		return waitIndexFetch(ctx, call)
	}

	call := &indexFetch{done: make(chan struct{})}
	s.indexFetch = call
	s.mu.Unlock()

	index, err := s.fetchIndex(ctx)

	s.mu.Lock()
	if err == nil {
		s.index = index
	}
	call.index = index
	call.err = err
	close(call.done)
	s.indexFetch = nil
	s.mu.Unlock()

	return index, err
}

func waitIndexFetch(ctx context.Context, call *indexFetch) (map[string]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.done:
		return call.index, call.err
	}
}

func (s *Service) fetchIndex(ctx context.Context) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, masterListURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "egg-analyze")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("atlas index HTTP %d", resp.StatusCode)
	}

	var payload masterList
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	index := make(map[string]string, len(payload.Creatures))
	for _, item := range payload.Creatures {
		id := normalizePetID(item.ID)
		if id == "" {
			continue
		}
		index[id] = strings.TrimSpace(item.Images.Default)
	}
	return index, nil
}

func (s *Service) readOrDownload(ctx context.Context, name string) ([]byte, error) {
	cachePath, err := cachePath(name)
	if err != nil {
		return nil, err
	}

	if data, err := os.ReadFile(cachePath); err == nil {
		return data, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL(name), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "egg-analyze")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, os.ErrNotExist
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("atlas asset HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return nil, err
	}
	return data, nil
}

func cachePath(name string) (string, error) {
	dir, err := config.AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "creature-atlas", name), nil
}

func assetURL(name string) string {
	return atlasBaseURL + "/" + url.PathEscape(name)
}

func (s *Service) WarmIndex(ctx context.Context) error {
	_, err := s.getIndex(ctx)
	return err
}

func normalizePetID(petID string) string {
	petID = strings.TrimSpace(petID)
	if petID == "" {
		return ""
	}
	if value, err := strconv.Atoi(petID); err == nil {
		return fmt.Sprintf("%03d", value)
	}
	return petID
}
