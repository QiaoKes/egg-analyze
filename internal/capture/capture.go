package capture

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/kbinani/screenshot"
)

func CapturePrimary() (image.Image, error) {
	count := screenshot.NumActiveDisplays()
	if count == 0 {
		return nil, fmt.Errorf("no active display found")
	}
	bounds := screenshot.GetDisplayBounds(0)
	return screenshot.CaptureRect(bounds)
}

func SavePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}
