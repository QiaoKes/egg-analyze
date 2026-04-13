package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type windowDragLabel struct {
	widget.BaseWidget
	label   string
	window  fyne.Window
	align   fyne.TextAlign
	started bool
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
	return fyne.NewSize(size.Width+12, size.Height+10)
}

func (w *windowDragLabel) Dragged(event *fyne.DragEvent) {
	if w.window == nil {
		return
	}
	if !w.started {
		w.started = true
		beginNativeWindowDrag(w.window)
	}
}

func (w *windowDragLabel) DragEnd() {
	w.started = false
}

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

var _ fyne.Draggable = (*windowDragLabel)(nil)
