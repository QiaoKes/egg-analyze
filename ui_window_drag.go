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
	started bool
}

func newWindowDragLabel(label string, win fyne.Window) *windowDragLabel {
	w := &windowDragLabel{label: label, window: win}
	w.ExtendBaseWidget(w)
	return w
}

func (w *windowDragLabel) MinSize() fyne.Size {
	txt := canvas.NewText(w.label, color.NRGBA{R: 0x18, G: 0x1F, B: 0x2D, A: 0xFF})
	txt.TextSize = 17
	txt.TextStyle = fyne.TextStyle{Bold: true}
	size := txt.MinSize()
	return fyne.NewSize(size.Width+6, size.Height+4)
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
	return &windowDragLabelRenderer{widget: w, text: text, objects: []fyne.CanvasObject{text}}
}

type windowDragLabelRenderer struct {
	widget  *windowDragLabel
	text    *canvas.Text
	objects []fyne.CanvasObject
}

func (r *windowDragLabelRenderer) Layout(size fyne.Size) {
	min := r.text.MinSize()
	r.text.Move(fyne.NewPos((size.Width-min.Width)/2, (size.Height-min.Height)/2))
	r.text.Resize(min)
}

func (r *windowDragLabelRenderer) MinSize() fyne.Size {
	return r.widget.MinSize()
}

func (r *windowDragLabelRenderer) Refresh() {
	r.text.Text = r.widget.label
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
