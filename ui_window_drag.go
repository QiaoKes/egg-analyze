package main

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type windowDragLabel struct {
	widget.BaseWidget
	label      string
	window     fyne.Window
	align      fyne.TextAlign
	nextDragAt time.Time
}

func newWindowDragLabel(label string, win fyne.Window, align fyne.TextAlign) *windowDragLabel {
	w := &windowDragLabel{label: label, window: win, align: align}
	w.ExtendBaseWidget(w)
	return w
}

func (w *windowDragLabel) MinSize() fyne.Size {
	txt := canvas.NewText(w.label, color.NRGBA{R: 0x18, G: 0x1F, B: 0x2D, A: 0xFF})
	txt.TextSize = 17
	txt.TextStyle = fyne.TextStyle{Bold: true}
	size := txt.MinSize()
	return fyne.NewSize(size.Width+8, size.Height+4)
}

func (w *windowDragLabel) Dragged(event *fyne.DragEvent) {
	if w.window == nil {
		return
	}
	if time.Now().After(w.nextDragAt) {
		w.nextDragAt = time.Now().Add(220 * time.Millisecond)
		beginNativeWindowDrag(w.window)
	}
}

func (w *windowDragLabel) DragEnd() {}

func (w *windowDragLabel) MouseDown(ev *desktop.MouseEvent) {
	if ev == nil || ev.Button != desktop.MouseButtonPrimary || w.window == nil {
		return
	}
	if time.Now().After(w.nextDragAt) {
		w.nextDragAt = time.Now().Add(220 * time.Millisecond)
		beginNativeWindowDrag(w.window)
	}
}

func (w *windowDragLabel) MouseUp(*desktop.MouseEvent) {}

func (w *windowDragLabel) CreateRenderer() fyne.WidgetRenderer {
	text := canvas.NewText(w.label, color.NRGBA{R: 0x18, G: 0x1F, B: 0x2D, A: 0xFF})
	text.TextStyle = fyne.TextStyle{Bold: true}
	text.TextSize = 17
	text.Alignment = w.align
	return &windowDragLabelRenderer{widget: w, text: text, objects: []fyne.CanvasObject{text}}
}

type windowDragLabelRenderer struct {
	widget  *windowDragLabel
	text    *canvas.Text
	objects []fyne.CanvasObject
}

func (r *windowDragLabelRenderer) Layout(size fyne.Size) {
	min := r.text.MinSize()
	x := float32(0)
	switch r.widget.align {
	case fyne.TextAlignLeading:
		x = 4
	case fyne.TextAlignTrailing:
		x = size.Width - min.Width - 4
	default:
		x = (size.Width - min.Width) / 2
	}
	if x < 0 {
		x = 0
	}
	r.text.Move(fyne.NewPos(x, (size.Height-min.Height)/2))
	r.text.Resize(min)
}

func (r *windowDragLabelRenderer) MinSize() fyne.Size {
	return r.widget.MinSize()
}

func (r *windowDragLabelRenderer) Refresh() {
	r.text.Text = r.widget.label
	r.text.Alignment = r.widget.align
	r.text.Refresh()
}

func (r *windowDragLabelRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *windowDragLabelRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *windowDragLabelRenderer) Destroy() {}

type windowDragArea struct {
	widget.BaseWidget
	window     fyne.Window
	nextDragAt time.Time
}

func newWindowDragArea(win fyne.Window) *windowDragArea {
	w := &windowDragArea{window: win}
	w.ExtendBaseWidget(w)
	return w
}

func (w *windowDragArea) MinSize() fyne.Size {
	return fyne.NewSize(1, 1)
}

func (w *windowDragArea) Dragged(*fyne.DragEvent) {
	if w.window == nil {
		return
	}
	if time.Now().After(w.nextDragAt) {
		w.nextDragAt = time.Now().Add(220 * time.Millisecond)
		beginNativeWindowDrag(w.window)
	}
}

func (w *windowDragArea) DragEnd() {}

func (w *windowDragArea) MouseDown(ev *desktop.MouseEvent) {
	if ev == nil || ev.Button != desktop.MouseButtonPrimary || w.window == nil {
		return
	}
	if time.Now().After(w.nextDragAt) {
		w.nextDragAt = time.Now().Add(220 * time.Millisecond)
		beginNativeWindowDrag(w.window)
	}
}

func (w *windowDragArea) MouseUp(*desktop.MouseEvent) {}

func (w *windowDragArea) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	return widget.NewSimpleRenderer(bg)
}

var _ fyne.Draggable = (*windowDragLabel)(nil)
var _ desktop.Mouseable = (*windowDragLabel)(nil)
var _ fyne.Draggable = (*windowDragArea)(nil)
var _ desktop.Mouseable = (*windowDragArea)(nil)
