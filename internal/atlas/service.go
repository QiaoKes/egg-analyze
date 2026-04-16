package atlas

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"egg-analyze/internal/config"

	"golang.org/x/image/webp"
)

const atlasBaseURL = "https://rocom.aoe.top/assets/webp/friends"

type imageLoad struct {
	done chan struct{}
	img  image.Image
	err  error
}

type Service struct {
	client *http.Client

	mu      sync.RWMutex
	decoded map[string]image.Image
	loading map[string]*imageLoad
}

func New() *Service {
	return &Service{
		client:  &http.Client{Timeout: 8 * time.Second},
		decoded: make(map[string]image.Image),
		loading: make(map[string]*imageLoad),
	}
}

func (s *Service) Load(ctx context.Context, spriteKey string) (image.Image, error) {
	normalizedKey := normalizeSpriteKey(spriteKey)
	if normalizedKey == "" {
		return nil, fmt.Errorf("empty sprite key")
	}

	s.mu.RLock()
	if cached, ok := s.decoded[normalizedKey]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	if call, ok := s.loading[normalizedKey]; ok {
		s.mu.Unlock()
		return waitImageLoad(ctx, call)
	}
	call := &imageLoad{done: make(chan struct{})}
	s.loading[normalizedKey] = call
	s.mu.Unlock()

	img, err := s.loadUncached(ctx, normalizedKey)

	s.mu.Lock()
	if err == nil {
		s.decoded[normalizedKey] = img
	}
	call.img = img
	call.err = err
	close(call.done)
	delete(s.loading, normalizedKey)
	s.mu.Unlock()

	return img, err
}

func (s *Service) loadUncached(ctx context.Context, normalizedKey string) (image.Image, error) {
	data, err := s.readOrDownload(ctx, normalizedKey)
	if err != nil {
		return nil, err
	}

	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", normalizedKey, err)
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

func (s *Service) readOrDownload(ctx context.Context, key string) ([]byte, error) {
	cachePath, err := cachePath(key)
	if err != nil {
		return nil, err
	}

	if data, err := os.ReadFile(cachePath); err == nil {
		return data, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL(key), nil)
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

func cachePath(key string) (string, error) {
	dir, err := config.AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "friend-atlas", "JL_"+normalizeSpriteKey(key)+".webp"), nil
}

func assetURL(key string) string {
	return atlasBaseURL + "/" + url.PathEscape("JL_"+normalizeSpriteKey(key)+".webp")
}

func (s *Service) WarmIndex(context.Context) error {
	return nil
}

func normalizeSpriteKey(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "JL_")
	value = strings.TrimSuffix(value, ".webp")
	return value
}
