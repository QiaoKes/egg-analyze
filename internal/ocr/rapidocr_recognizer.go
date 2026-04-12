package ocr

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed rapidocr_runner.py
var rapidOCRScript string

type rapidOCRRecognizer struct {
	python     string
	mu         sync.Mutex
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *bufio.Reader
	stderr     strings.Builder
	scriptPath string
	tempDir    string
}

func newRapidOCRRecognizer() (Recognizer, error) {
	python, err := resolvePython()
	if err != nil {
		return nil, err
	}
	tempDir, err := os.MkdirTemp("", "egg-analyze-rapidocr-worker-*")
	if err != nil {
		return nil, err
	}
	scriptPath := filepath.Join(tempDir, "rapidocr_runner.py")
	if err := os.WriteFile(scriptPath, []byte(rapidOCRScript), 0o755); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, err
	}
	return &rapidOCRRecognizer{
		python:     python,
		tempDir:    tempDir,
		scriptPath: scriptPath,
	}, nil
}

func (r *rapidOCRRecognizer) Name() string {
	return "rapidocr"
}

func (r *rapidOCRRecognizer) Recognize(ctx context.Context, img image.Image) ([]Line, error) {
	batches, err := r.RecognizeBatch(ctx, []image.Image{img})
	if err != nil {
		return nil, err
	}
	if len(batches) == 0 {
		return nil, nil
	}
	return batches[0], nil
}

func (r *rapidOCRRecognizer) RecognizeBatch(ctx context.Context, images []image.Image) ([][]Line, error) {
	if len(images) == 0 {
		return nil, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensureProcess(); err != nil {
		return nil, err
	}

	batches := make([][]Line, 0, len(images))
	for idx, img := range images {
		imagePath := filepath.Join(r.tempDir, fmt.Sprintf("input-%02d.png", idx))
		if err := writePNG(imagePath, img); err != nil {
			return nil, err
		}
		lines, err := r.recognizePathLocked(imagePath)
		_ = os.Remove(imagePath)
		if err != nil {
			r.closeProcessLocked()
			return nil, err
		}
		batches = append(batches, lines)
	}
	return batches, nil
}

func (r *rapidOCRRecognizer) ensureProcess() error {
	if r.cmd != nil && r.cmd.Process != nil {
		return nil
	}

	cmd := exec.Command(r.python, r.scriptPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}
	cmd.Stderr = &r.stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}

	r.cmd = cmd
	r.stdin = stdin
	r.stdout = bufio.NewReader(stdout)
	return nil
}

func (r *rapidOCRRecognizer) recognizePathLocked(path string) ([]Line, error) {
	r.stderr.Reset()
	payload, err := json.Marshal(map[string]string{"path": path})
	if err != nil {
		return nil, err
	}
	if _, err := r.stdin.Write(append(payload, '\n')); err != nil {
		return nil, err
	}

	line, err := r.stdout.ReadBytes('\n')
	if err != nil {
		if stderr := strings.TrimSpace(r.stderr.String()); stderr != "" {
			return nil, fmt.Errorf("%w: %s", err, stderr)
		}
		return nil, err
	}

	var response struct {
		OK        bool   `json:"ok"`
		Error     string `json:"error"`
		Traceback string `json:"traceback"`
		Lines     []Line `json:"lines"`
	}
	if err := json.Unmarshal(bytesTrimSpace(line), &response); err != nil {
		return nil, fmt.Errorf("decode rapidocr output: %w: %s", err, strings.TrimSpace(string(line)))
	}
	if !response.OK {
		message := strings.TrimSpace(response.Error)
		if response.Traceback != "" {
			message = strings.TrimSpace(message + " " + response.Traceback)
		}
		if stderr := strings.TrimSpace(r.stderr.String()); stderr != "" {
			message = strings.TrimSpace(message + " " + stderr)
		}
		return nil, fmt.Errorf("rapidocr failed: %s", strings.TrimSpace(message))
	}
	return response.Lines, nil
}

func (r *rapidOCRRecognizer) closeProcessLocked() {
	if r.stdin != nil {
		_ = r.stdin.Close()
	}
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
		_, _ = r.cmd.Process.Wait()
	}
	r.cmd = nil
	r.stdin = nil
	r.stdout = nil
}

func bytesTrimSpace(input []byte) []byte {
	return []byte(strings.TrimSpace(string(input)))
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
		bundledPythonPath(),
		filepath.Join(".venv", "bin", "python3"),
		filepath.Join(".venv", "bin", "python"),
		filepath.Join(".venv", "Scripts", "python.exe"),
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

	return "", errors.New("RapidOCR requires Python. Use a bundled package, create .venv and install rapidocr-onnxruntime, or set EGG_ANALYZE_PYTHON")
}

func bundledPythonPath() string {
	executable, err := os.Executable()
	if err != nil {
		return ""
	}

	baseDir := filepath.Dir(executable)
	candidates := []string{
		filepath.Join(baseDir, "python", "python.exe"),
		filepath.Join(baseDir, "python", "bin", "python3"),
		filepath.Join(baseDir, "python", "bin", "python"),
		filepath.Join(baseDir, "..", "Resources", "python", "bin", "python3"),
		filepath.Join(baseDir, "..", "Resources", "python", "bin", "python"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
