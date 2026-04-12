package ocr

import (
	"context"
	"image"
)

type Line struct {
	Text   string `json:"Text"`
	X      int    `json:"X"`
	Y      int    `json:"Y"`
	Width  int    `json:"Width"`
	Height int    `json:"Height"`
}

type Recognizer interface {
	Name() string
	Recognize(context.Context, image.Image) ([]Line, error)
}

type BatchRecognizer interface {
	Recognizer
	RecognizeBatch(context.Context, []image.Image) ([][]Line, error)
}
