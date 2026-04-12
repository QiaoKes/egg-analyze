//go:build !windows

package ocr

import (
	"context"
	"fmt"
	"image"
)

type Recognizer struct{}

func New() *Recognizer {
	return &Recognizer{}
}

func (r *Recognizer) Recognize(_ context.Context, _ image.Image) ([]Line, error) {
	return nil, fmt.Errorf("windows OCR is only supported on Windows")
}
