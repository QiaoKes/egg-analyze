package main

import (
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
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
	return fyne.NewSize(78, 78)
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
	shadow := canvas.NewCircle(color.NRGBA{R: 0x18, G: 0x1B, B: 0x22, A: 0x12})
	halo := canvas.NewCircle(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x16})
	face := canvas.NewCircle(color.NRGBA{R: 0xF9, G: 0xFB, B: 0xFF, A: 0xD8})
	face.StrokeColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x90}
	face.StrokeWidth = 1
	inner := canvas.NewCircle(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x36})
	icon := canvas.NewImageFromResource(theme.MediaPhotoIcon())
	icon.FillMode = canvas.ImageFillContain
	indicator := canvas.NewCircle(color.NRGBA{R: 0x57, G: 0x93, B: 0xFF, A: 0xFF})
	indicator.StrokeColor = color.White
	indicator.StrokeWidth = 1.4

	objects := []fyne.CanvasObject{shadow, halo, face, inner, icon, indicator}
	return &bubbleRenderer{
		widget:    w,
		shadow:    shadow,
		halo:      halo,
		face:      face,
		inner:     inner,
		icon:      icon,
		indicator: indicator,
		objects:   objects,
	}
}

type bubbleRenderer struct {
	widget    *bubbleWidget
	shadow    *canvas.Circle
	halo      *canvas.Circle
	face      *canvas.Circle
	inner     *canvas.Circle
	icon      *canvas.Image
	indicator *canvas.Circle
	objects   []fyne.CanvasObject
}

func (r *bubbleRenderer) Layout(size fyne.Size) {
	r.shadow.Move(fyne.NewPos(7, 8))
	r.shadow.Resize(fyne.NewSize(size.Width-14, size.Height-13))

	r.halo.Move(fyne.NewPos(4, 4))
	r.halo.Resize(fyne.NewSize(size.Width-8, size.Height-8))

	r.face.Move(fyne.NewPos(5, 5))
	r.face.Resize(fyne.NewSize(size.Width-10, size.Height-10))

	r.inner.Move(fyne.NewPos(size.Width*0.23, size.Height*0.19))
	r.inner.Resize(fyne.NewSize(size.Width*0.2, size.Height*0.14))

	iconSize := fyne.NewSize(size.Width*0.28, size.Height*0.28)
	r.icon.Move(fyne.NewPos((size.Width-iconSize.Width)/2, (size.Height-iconSize.Height)/2))
	r.icon.Resize(iconSize)

	indicatorSize := fyne.NewSize(10, 10)
	r.indicator.Move(fyne.NewPos(size.Width-22, 14))
	r.indicator.Resize(indicatorSize)
}

func (r *bubbleRenderer) MinSize() fyne.Size {
	return r.widget.MinSize()
}

func (r *bubbleRenderer) Refresh() {
	canvas.Refresh(r.shadow)
	canvas.Refresh(r.halo)
	canvas.Refresh(r.face)
	canvas.Refresh(r.inner)
	canvas.Refresh(r.icon)
	canvas.Refresh(r.indicator)
}

func (r *bubbleRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *bubbleRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *bubbleRenderer) Destroy() {}
