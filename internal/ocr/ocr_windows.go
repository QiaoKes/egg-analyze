//go:build windows

package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

type Recognizer struct{}

func New() *Recognizer {
	return &Recognizer{}
}

func (r *Recognizer) Recognize(ctx context.Context, img image.Image) ([]Line, error) {
	tempDir, err := os.MkdirTemp("", "egg-analyze-ocr-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	imagePath := filepath.Join(tempDir, "input.png")
	file, err := os.Create(imagePath)
	if err != nil {
		return nil, err
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}

	script := buildScript(imagePath)
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ocr command failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	var lines []Line
	if err := json.Unmarshal(output, &lines); err != nil {
		return nil, fmt.Errorf("decode ocr output: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return lines, nil
}

func buildScript(imagePath string) string {
	escapedPath := strings.ReplaceAll(imagePath, "'", "''")
	return strings.Join([]string{
		"Add-Type -AssemblyName System.Runtime.WindowsRuntime",
		"$null = [Windows.Storage.StorageFile, Windows.Storage, ContentType = WindowsRuntime]",
		"$null = [Windows.Media.Ocr.OcrEngine, Windows.Foundation, ContentType = WindowsRuntime]",
		"$null = [Windows.Foundation.IAsyncOperation`1, Windows.Foundation, ContentType = WindowsRuntime]",
		"$null = [Windows.Graphics.Imaging.SoftwareBitmap, Windows.Foundation, ContentType = WindowsRuntime]",
		"$null = [Windows.Storage.Streams.RandomAccessStream, Windows.Storage.Streams, ContentType = WindowsRuntime]",
		"$null = [WindowsRuntimeSystemExtensions]",
		"$null = [Windows.Media.Ocr.OcrEngine]::AvailableRecognizerLanguages",
		"$awaiter = [WindowsRuntimeSystemExtensions].GetMember('GetAwaiter', 'Method', 'Public,Static') |",
		"  Where-Object { $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1' } |",
		"  Select-Object -First 1",
		"function Invoke-Async([object]$AsyncTask, [Type]$As) {",
		"  return $awaiter.MakeGenericMethod($As).Invoke($null, @($AsyncTask)).GetResult()",
		"}",
		fmt.Sprintf("$path = '%s'", escapedPath),
		"$storageFile = Invoke-Async ([Windows.Storage.StorageFile]::GetFileFromPathAsync($path)) -As ([Windows.Storage.StorageFile])",
		"$fileStream = Invoke-Async ($storageFile.OpenAsync([Windows.Storage.FileAccessMode]::Read)) -As ([Windows.Storage.Streams.IRandomAccessStream])",
		"$bitmapDecoder = Invoke-Async ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($fileStream)) -As ([Windows.Graphics.Imaging.BitmapDecoder])",
		"$softwareBitmap = Invoke-Async ($bitmapDecoder.GetSoftwareBitmapAsync()) -As ([Windows.Graphics.Imaging.SoftwareBitmap])",
		"$ocrEngine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()",
		"$result = Invoke-Async ($ocrEngine.RecognizeAsync($softwareBitmap)) -As ([Windows.Media.Ocr.OcrResult])",
		"$lines = foreach ($line in $result.Lines) {",
		"  $words = @($line.Words)",
		"  if ($words.Count -eq 0) { continue }",
		"  $first = $words[0]",
		"  $last = $words[$words.Count - 1]",
		"  [PSCustomObject]@{",
		"    Text = $line.Text",
		"    X = [int]$first.BoundingRect.X",
		"    Y = [int]$first.BoundingRect.Y",
		"    Width = [int](($last.BoundingRect.X + $last.BoundingRect.Width) - $first.BoundingRect.X)",
		"    Height = [int]$first.BoundingRect.Height",
		"  }",
		"}",
		"$lines | ConvertTo-Json -Depth 4 -Compress",
	}, "\n")
}
