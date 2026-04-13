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
	configureCommand(cmd)
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
		executable = ""
	}

	baseDirs := collectSearchRoots(executable)
	if runtime.GOOS == "windows" {
		var candidates []string
		for _, baseDir := range baseDirs {
			candidates = append(candidates,
				filepath.Join(baseDir, "flameshot.exe"),
				filepath.Join(baseDir, "flameshot-cli.exe"),
				filepath.Join(baseDir, "flameshot", "flameshot.exe"),
				filepath.Join(baseDir, "flameshot", "flameshot-cli.exe"),
				filepath.Join(baseDir, "bin", "flameshot.exe"),
				filepath.Join(baseDir, "bin", "flameshot-cli.exe"),
				filepath.Join(baseDir, "flameshot", "bin", "flameshot.exe"),
				filepath.Join(baseDir, "flameshot", "bin", "flameshot-cli.exe"),
				filepath.Join(baseDir, "flameshot", "flameshot-13.3.0-win64", "bin", "flameshot.exe"),
				filepath.Join(baseDir, "flameshot", "flameshot-13.3.0-win64", "bin", "flameshot-cli.exe"),
			)

			if matches, err := filepath.Glob(filepath.Join(baseDir, "flameshot", "*", "bin", "flameshot*.exe")); err == nil {
				candidates = append(candidates, matches...)
			}
		}
		return candidates
	}
	if runtime.GOOS == "darwin" {
		candidates := []string{
			filepath.Join("/Applications", "Flameshot.app", "Contents", "MacOS", "flameshot"),
		}
		for _, baseDir := range baseDirs {
			candidates = append(candidates,
				filepath.Join(baseDir, "Flameshot.app", "Contents", "MacOS", "flameshot"),
				filepath.Join(baseDir, "flameshot"),
				filepath.Join(baseDir, "bin", "Flameshot.app", "Contents", "MacOS", "flameshot"),
				filepath.Join(baseDir, "..", "Resources", "Flameshot.app", "Contents", "MacOS", "flameshot"),
			)
		}
		return candidates
	}
	var candidates []string
	for _, baseDir := range baseDirs {
		candidates = append(candidates,
			filepath.Join(baseDir, "flameshot"),
			filepath.Join(baseDir, "bin", "flameshot"),
		)
	}
	return candidates
}

func collectSearchRoots(executable string) []string {
	seen := map[string]struct{}{}
	roots := make([]string, 0, 6)
	add := func(path string) {
		if strings.TrimSpace(path) == "" {
			return
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		roots = append(roots, clean)
	}

	if executable != "" {
		baseDir := filepath.Dir(executable)
		add(baseDir)
		add(filepath.Dir(baseDir))
	}
	if cwd, err := os.Getwd(); err == nil {
		add(cwd)
		add(filepath.Dir(cwd))
	}

	return roots
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
