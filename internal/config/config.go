package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

const (
	appDirName          = "EggAnalyze"
	defaultHotkey       = "Ctrl+Shift+Q"
	defaultDatasetURL   = "https://rocom.mfsky.qzz.io/data/egg-measurements-final.json"
	configFileName      = "config.json"
	datasetCacheName    = "egg-measurements-final.json"
	lastCaptureFileName = "last-capture.png"
)

type Config struct {
	Hotkey       string `json:"hotkey"`
	DatasetURL   string `json:"dataset_url"`
	CaptureMode  string `json:"capture_mode"`
	AutoRefresh  bool   `json:"auto_refresh"`
	TopCandidate int    `json:"top_candidate"`
}

type Manager struct {
	mu   sync.RWMutex
	path string
	cfg  Config
}

func Default() Config {
	return Config{
		Hotkey:       defaultHotkey,
		DatasetURL:   defaultDatasetURL,
		CaptureMode:  "primary",
		AutoRefresh:  true,
		TopCandidate: 5,
	}
}

func Load() (*Manager, error) {
	dir, err := AppDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	manager := &Manager{
		path: filepath.Join(dir, configFileName),
		cfg:  Default(),
	}

	data, err := os.ReadFile(manager.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if saveErr := manager.Save(manager.cfg); saveErr != nil {
				return nil, saveErr
			}
			return manager, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	manager.cfg = normalize(cfg)
	return manager, nil
}

func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) Save(cfg Config) error {
	cfg = normalize(cfg)

	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(m.path, data, 0o644); err != nil {
		return err
	}
	m.cfg = cfg
	return nil
}

func AppDir() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, appDirName), nil
}

func DatasetCachePath() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, datasetCacheName), nil
}

func LastCapturePath() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, lastCaptureFileName), nil
}

func normalize(cfg Config) Config {
	def := Default()
	if cfg.Hotkey == "" {
		cfg.Hotkey = def.Hotkey
	}
	if cfg.DatasetURL == "" {
		cfg.DatasetURL = def.DatasetURL
	}
	if cfg.CaptureMode == "" {
		cfg.CaptureMode = def.CaptureMode
	}
	if cfg.TopCandidate <= 0 {
		cfg.TopCandidate = def.TopCandidate
	}
	return cfg
}
