package ocr

import (
	"context"
	"errors"
	"image"
)

type unavailableRecognizer struct {
	message string
}

func (r *unavailableRecognizer) Name() string {
	return "unavailable"
}

func (r *unavailableRecognizer) Recognize(_ context.Context, _ image.Image) ([]Line, error) {
	return nil, errors.New(r.reason())
}

func (r *unavailableRecognizer) reason() string {
	if r == nil || r.message == "" {
		return "no OCR provider is configured"
	}
	return r.message
}
