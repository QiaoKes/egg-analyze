package capture

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var ErrCancelled = errors.New("capture cancelled")

func CaptureWithFlameshot(ctx context.Context) (image.Image, error) {
	binary, err := findFlameshotBinary()
	if err != nil {
		return nil, err
	}

	targetPath, cleanup, err := newTempCapturePath()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	cmd := exec.CommandContext(ctx, binary, "gui", "-p", targetPath, "-s")
	output, runErr := cmd.CombinedOutput()
	if runErr != nil {
		if _, statErr := os.Stat(targetPath); errors.Is(statErr, os.ErrNotExist) {
			return nil, ErrCancelled
		}
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = runErr.Error()
		}
		return nil, fmt.Errorf("flameshot failed: %s", message)
	}

	file, err := os.Open(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrCancelled
		}
		return nil, fmt.Errorf("open flameshot capture: %w", err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode flameshot capture: %w", err)
	}
	return img, nil
}

func SavePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func findFlameshotBinary() (string, error) {
	candidates := []string{}
	if configured := strings.TrimSpace(os.Getenv("EGG_ANALYZE_FLAMESHOT_BIN")); configured != "" {
		candidates = append(candidates, configured)
	}
	candidates = append(candidates, bundledFlameshotCandidates()...)
	if runtime.GOOS == "windows" {
		candidates = append(candidates, "flameshot.exe", "flameshot-cli.exe", "flameshot", "flameshot-cli")
	} else {
		candidates = append(candidates, "flameshot", "flameshot-cli")
	}

	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("flameshot not found; install it, set EGG_ANALYZE_FLAMESHOT_BIN, or use a bundle that includes flameshot")
}

func bundledFlameshotCandidates() []string {
	executable, err := os.Executable()
	if err != nil {
		return nil
	}

	baseDir := filepath.Dir(executable)
	if runtime.GOOS == "windows" {
		return []string{
			filepath.Join(baseDir, "flameshot.exe"),
			filepath.Join(baseDir, "flameshot-cli.exe"),
			filepath.Join(baseDir, "flameshot", "flameshot.exe"),
			filepath.Join(baseDir, "flameshot", "flameshot-cli.exe"),
			filepath.Join(baseDir, "bin", "flameshot.exe"),
			filepath.Join(baseDir, "bin", "flameshot-cli.exe"),
		}
	}
	if runtime.GOOS == "darwin" {
		return []string{
			filepath.Join(baseDir, "Flameshot.app", "Contents", "MacOS", "flameshot"),
			filepath.Join(baseDir, "flameshot"),
			filepath.Join(baseDir, "bin", "Flameshot.app", "Contents", "MacOS", "flameshot"),
			filepath.Join(baseDir, "..", "Resources", "Flameshot.app", "Contents", "MacOS", "flameshot"),
		}
	}
	return []string{
		filepath.Join(baseDir, "flameshot"),
		filepath.Join(baseDir, "bin", "flameshot"),
	}
}

func newTempCapturePath() (string, func(), error) {
	dir, err := os.MkdirTemp("", "egg-analyze-flameshot-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp capture dir: %w", err)
	}

	path := filepath.Join(dir, "capture.png")
	cleanup := func() {
		_ = os.Remove(path)
		_ = os.RemoveAll(dir)
	}
	return path, cleanup, nil
}
