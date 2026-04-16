package main

import (
	"bytes"
	"image"
	stddraw "image/draw"
	"image/png"
	"math"
	"sync"

	projectassets "egg-analyze/assets"

	xdraw "golang.org/x/image/draw"
)

var (
	bubbleWidgetImageOnce sync.Once
	bubbleWidgetImage     image.Image
	bubbleWidgetImageErr  error
)

func loadBubbleWidgetImage() (image.Image, error) {
	bubbleWidgetImageOnce.Do(func() {
		src, err := png.Decode(bytes.NewReader(projectassets.BubblePNG))
		if err != nil {
			bubbleWidgetImageErr = err
			return
		}

		const canvasSize = 128
		srcSquare := bubbleWidgetCenterSquare(src.Bounds())
		dst := image.NewNRGBA(image.Rect(0, 0, canvasSize, canvasSize))
		cardInset := bubbleMaxInt(2, canvasSize/14)
		cardRect := image.Rect(cardInset, cardInset, canvasSize-cardInset, canvasSize-cardInset)
		cardRect = bubbleWidgetFitSquareRect(cardRect)
		if cardRect.Dx() <= 0 || cardRect.Dy() <= 0 {
			cardRect = dst.Bounds()
		}

		xdraw.CatmullRom.Scale(dst, cardRect, src, srcSquare, stddraw.Over, nil)
		applyBubbleRoundedRectAlpha(dst, cardRect, bubbleMaxInt(8, cardRect.Dx()/5))
		bubbleWidgetImage = dst
	})
	return bubbleWidgetImage, bubbleWidgetImageErr
}

func bubbleWidgetCenterSquare(r image.Rectangle) image.Rectangle {
	size := r.Dx()
	if r.Dy() < size {
		size = r.Dy()
	}
	if size <= 0 {
		return r
	}

	x0 := r.Min.X + (r.Dx()-size)/2
	y0 := r.Min.Y + (r.Dy()-size)/2
	return image.Rect(x0, y0, x0+size, y0+size)
}

func bubbleWidgetFitSquareRect(r image.Rectangle) image.Rectangle {
	size := bubbleMinInt(r.Dx(), r.Dy())
	if size <= 0 {
		return r
	}

	x0 := r.Min.X + (r.Dx()-size)/2
	y0 := r.Min.Y + (r.Dy()-size)/2
	return image.Rect(x0, y0, x0+size, y0+size)
}

func applyBubbleRoundedRectAlpha(img *image.NRGBA, rect image.Rectangle, radius int) {
	radius = bubbleMinInt(radius, bubbleMinInt(rect.Dx(), rect.Dy())/2)
	if radius <= 0 {
		return
	}

	innerLeft := rect.Min.X + radius
	innerRight := rect.Max.X - radius
	innerTop := rect.Min.Y + radius
	innerBottom := rect.Max.Y - radius
	radiusSquared := float64(radius * radius)

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			inside := x >= innerLeft && x < innerRight
			inside = inside || y >= innerTop && y < innerBottom
			if !inside {
				cx := bubbleClampInt(x, innerLeft, innerRight-1)
				cy := bubbleClampInt(y, innerTop, innerBottom-1)
				dx := float64(x - cx)
				dy := float64(y - cy)
				if dx*dx+dy*dy > radiusSquared {
					offset := img.PixOffset(x, y)
					img.Pix[offset+3] = 0
				}
			}
		}
	}
}

func bubbleClampInt(value, minValue, maxValue int) int {
	return int(math.Max(float64(minValue), math.Min(float64(maxValue), float64(value))))
}

func bubbleMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func bubbleMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
