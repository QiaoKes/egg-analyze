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
	return fyne.NewSize(60, 60)
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
	if w.dragged {
		return
	}
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
	shadow := canvas.NewCircle(color.NRGBA{R: 0x0F, G: 0x17, B: 0x24, A: 0x12})
	body := canvas.NewCircle(color.NRGBA{R: 0xF8, G: 0xFA, B: 0xFE, A: 0xFF})
	body.StrokeColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x90}
	body.StrokeWidth = 1
	highlight := canvas.NewRectangle(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xA8})
	accent := canvas.NewCircle(color.NRGBA{R: 0x59, G: 0x89, B: 0xFF, A: 0xFF})
	ring := canvas.NewCircle(color.Transparent)
	ring.StrokeColor = color.NRGBA{R: 0x2E, G: 0x37, B: 0x4E, A: 0xFF}
	ring.StrokeWidth = 5
	cap := canvas.NewRectangle(color.NRGBA{R: 0x2E, G: 0x37, B: 0x4E, A: 0xFF})
	core := canvas.NewCircle(color.NRGBA{R: 0x2E, G: 0x37, B: 0x4E, A: 0xFF})
	return &bubbleRenderer{
		widget:    w,
		shadow:    shadow,
		body:      body,
		highlight: highlight,
		accent:    accent,
		ring:      ring,
		cap:       cap,
		core:      core,
		objects:   []fyne.CanvasObject{shadow, body, highlight, accent, ring, cap, core},
	}
}

type bubbleRenderer struct {
	widget    *bubbleWidget
	shadow    *canvas.Circle
	body      *canvas.Circle
	highlight *canvas.Rectangle
	accent    *canvas.Circle
	ring      *canvas.Circle
	cap       *canvas.Rectangle
	core      *canvas.Circle
	objects   []fyne.CanvasObject
}

func (r *bubbleRenderer) Layout(size fyne.Size) {
	minEdge := minFloat32(size.Width, size.Height)
	bodyInset := maxFloat32(4, minEdge*0.08)
	bodyEdge := maxFloat32(1, minEdge-bodyInset*2)
	bodyX := (size.Width - bodyEdge) / 2
	bodyY := (size.Height - bodyEdge) / 2

	shadowInset := maxFloat32(1, bodyEdge*0.02)
	shadowOffset := maxFloat32(1, bodyEdge*0.06)
	r.shadow.Move(fyne.NewPos(bodyX+shadowInset, bodyY+shadowInset+shadowOffset))
	r.shadow.Resize(fyne.NewSize(bodyEdge-shadowInset*2, bodyEdge-shadowInset*2))

	r.body.Move(fyne.NewPos(bodyX, bodyY))
	r.body.Resize(fyne.NewSize(bodyEdge, bodyEdge))

	highlightWidth := bodyEdge * 0.30
	highlightHeight := bodyEdge * 0.14
	r.highlight.CornerRadius = highlightHeight / 2
	r.highlight.Move(fyne.NewPos(bodyX+bodyEdge*0.16, bodyY+bodyEdge*0.16))
	r.highlight.Resize(fyne.NewSize(highlightWidth, highlightHeight))

	accentSize := bodyEdge * 0.16
	r.accent.Move(fyne.NewPos(bodyX+bodyEdge*0.72, bodyY+bodyEdge*0.16))
	r.accent.Resize(fyne.NewSize(accentSize, accentSize))

	ringSize := bodyEdge * 0.30
	r.ring.StrokeWidth = maxFloat32(3, bodyEdge*0.09)
	r.ring.Move(fyne.NewPos(bodyX+(bodyEdge-ringSize)/2, bodyY+bodyEdge*0.42))
	r.ring.Resize(fyne.NewSize(ringSize, ringSize))

	capWidth := bodyEdge * 0.28
	capHeight := bodyEdge * 0.11
	r.cap.CornerRadius = capHeight / 2
	r.cap.Move(fyne.NewPos(bodyX+(bodyEdge-capWidth)/2, bodyY+bodyEdge*0.34))
	r.cap.Resize(fyne.NewSize(capWidth, capHeight))

	coreSize := bodyEdge * 0.11
	r.core.Move(fyne.NewPos(bodyX+(bodyEdge-coreSize)/2, bodyY+bodyEdge*0.52))
	r.core.Resize(fyne.NewSize(coreSize, coreSize))
}

func (r *bubbleRenderer) MinSize() fyne.Size {
	return r.widget.MinSize()
}

func (r *bubbleRenderer) Refresh() {
	canvas.Refresh(r.shadow)
	canvas.Refresh(r.body)
	canvas.Refresh(r.highlight)
	canvas.Refresh(r.accent)
	canvas.Refresh(r.ring)
	canvas.Refresh(r.cap)
	canvas.Refresh(r.core)
}

func (r *bubbleRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *bubbleRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *bubbleRenderer) Destroy() {}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
