package ocr

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed rapidocr_runner.py
var rapidOCRScript string

type rapidOCRRecognizer struct {
	python string
}

func newRapidOCRRecognizer() (Recognizer, error) {
	python, err := resolvePython()
	if err != nil {
		return nil, err
	}
	return &rapidOCRRecognizer{python: python}, nil
}

func (r *rapidOCRRecognizer) Name() string {
	return "rapidocr"
}

func (r *rapidOCRRecognizer) Recognize(ctx context.Context, img image.Image) ([]Line, error) {
	tempDir, err := os.MkdirTemp("", "egg-analyze-rapidocr-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "input.png")
	scriptPath := filepath.Join(tempDir, "rapidocr_runner.py")

	if err := writePNG(imagePath, img); err != nil {
		return nil, err
	}
	if err := os.WriteFile(scriptPath, []byte(rapidOCRScript), 0o755); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, r.python, scriptPath, imagePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("rapidocr failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	var lines []Line
	if err := json.Unmarshal(output, &lines); err != nil {
		return nil, fmt.Errorf("decode rapidocr output: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return lines, nil
}

func writePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func resolvePython() (string, error) {
	candidates := []string{
		strings.TrimSpace(os.Getenv("EGG_ANALYZE_PYTHON")),
		filepath.Join(".venv", "bin", "python3"),
		filepath.Join(".venv", "bin", "python"),
		"python3",
		"python",
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if strings.Contains(candidate, string(filepath.Separator)) {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
			continue
		}
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, nil
		}
	}

	return "", errors.New("RapidOCR requires Python. Create .venv and install rapidocr-onnxruntime, or set EGG_ANALYZE_PYTHON")
}
