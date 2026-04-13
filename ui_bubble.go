package main

import (
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type bubbleWidget struct {
	widget.BaseWidget
	onTap            func()
	onDragStart      func()
	onDragEnd        func()
	dragged          bool
	suppressTapUntil time.Time
	nextDragAt       time.Time
}

func newBubbleWidget(onTap func(), onDragStart func(), onDragEnd func()) *bubbleWidget {
	w := &bubbleWidget{onTap: onTap, onDragStart: onDragStart, onDragEnd: onDragEnd}
	w.ExtendBaseWidget(w)
	return w
}

func (w *bubbleWidget) MinSize() fyne.Size {
	return fyne.NewSize(52, 52)
}

func (w *bubbleWidget) Tapped(*fyne.PointEvent) {
	if time.Now().Before(w.suppressTapUntil) {
		return
	}
	if w.onTap != nil {
		w.onTap()
	}
}

func (w *bubbleWidget) TappedSecondary(*fyne.PointEvent) {}

func (w *bubbleWidget) Dragged(event *fyne.DragEvent) {
	if math.Abs(float64(event.Dragged.DX)) >= 2 || math.Abs(float64(event.Dragged.DY)) >= 2 {
		w.dragged = true
		if time.Now().After(w.nextDragAt) && w.onDragStart != nil {
			w.nextDragAt = time.Now().Add(220 * time.Millisecond)
			w.onDragStart()
		}
	}
}

func (w *bubbleWidget) DragEnd() {
	if w.dragged {
		w.suppressTapUntil = time.Now().Add(120 * time.Millisecond)
		if w.onDragEnd != nil {
			w.onDragEnd()
		}
	}
	w.dragged = false
}

func (w *bubbleWidget) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromResource(bubbleImageResource)
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleSmooth
	return &bubbleRenderer{
		widget:  w,
		image:   img,
		objects: []fyne.CanvasObject{img},
	}
}

type bubbleRenderer struct {
	widget  *bubbleWidget
	image   *canvas.Image
	objects []fyne.CanvasObject
}

func (r *bubbleRenderer) Layout(size fyne.Size) {
	r.image.Move(fyne.NewPos(0, 0))
	r.image.Resize(size)
}

func (r *bubbleRenderer) MinSize() fyne.Size {
	return r.widget.MinSize()
}

func (r *bubbleRenderer) Refresh() {
	canvas.Refresh(r.image)
}

func (r *bubbleRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *bubbleRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *bubbleRenderer) Destroy() {}
