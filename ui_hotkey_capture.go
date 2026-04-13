package main

import (
	"image/color"

	"egg-analyze/internal/hotkey"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type hotkeyCaptureWidget struct {
	widget.BaseWidget

	current   string
	pressed   fyne.KeyModifier
	focused   bool
	onConfirm func(string)
	onCancel  func()
}

func newHotkeyCaptureWidget(current string, onConfirm func(string), onCancel func()) *hotkeyCaptureWidget {
	w := &hotkeyCaptureWidget{
		current:   hotkey.NormalizeShortcut(current),
		onConfirm: onConfirm,
		onCancel:  onCancel,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *hotkeyCaptureWidget) MinSize() fyne.Size {
	return fyne.NewSize(360, 120)
}

func (w *hotkeyCaptureWidget) Tapped(*fyne.PointEvent) {
	w.focus()
}

func (w *hotkeyCaptureWidget) FocusGained() {
	w.focused = true
	w.Refresh()
}

func (w *hotkeyCaptureWidget) FocusLost() {
	w.focused = false
	w.pressed = 0
	w.Refresh()
}

func (w *hotkeyCaptureWidget) TypedRune(rune) {}

func (w *hotkeyCaptureWidget) TypedKey(event *fyne.KeyEvent) {
	w.handleTypedKey(event)
}

func (w *hotkeyCaptureWidget) KeyDown(event *fyne.KeyEvent) {
	w.handleKeyDown(event)
}

func (w *hotkeyCaptureWidget) KeyUp(event *fyne.KeyEvent) {
	w.handleKeyUp(event)
}

func (w *hotkeyCaptureWidget) handleTypedKey(event *fyne.KeyEvent) {
	if w.pressed != 0 {
		return
	}
	switch event.Name {
	case fyne.KeyEscape:
		if w.onCancel != nil {
			w.onCancel()
		}
	case fyne.KeyBackspace, fyne.KeyDelete:
		w.current = ""
		w.Refresh()
	}
}

func (w *hotkeyCaptureWidget) handleKeyDown(event *fyne.KeyEvent) {
	if modifier, ok := hotkey.ModifierFromKey(event.Name); ok {
		w.pressed |= modifier
		w.Refresh()
		return
	}
	if hotkey.IsModifierKey(event.Name) {
		return
	}
	if w.pressed == 0 {
		switch event.Name {
		case fyne.KeyEscape:
			if w.onCancel != nil {
				w.onCancel()
			}
			return
		case fyne.KeyBackspace, fyne.KeyDelete:
			w.current = ""
			w.Refresh()
			return
		}
	}

	value, err := hotkey.FormatShortcut(w.pressed, event.Name)
	if err != nil {
		return
	}
	w.current = hotkey.NormalizeShortcut(value)
	w.Refresh()
	if w.onConfirm != nil {
		w.onConfirm(value)
	}
}

func (w *hotkeyCaptureWidget) handleKeyUp(event *fyne.KeyEvent) {
	if modifier, ok := hotkey.ModifierFromKey(event.Name); ok {
		w.pressed &^= modifier
		w.Refresh()
	}
}

func (w *hotkeyCaptureWidget) CreateRenderer() fyne.WidgetRenderer {
	border := canvas.NewRectangle(color.NRGBA{R: 0xC8, G: 0xB0, B: 0x7D, A: 0xFF})
	fill := canvas.NewRectangle(color.NRGBA{R: 0xFF, G: 0xFB, B: 0xF2, A: 0xFF})
	title := canvas.NewText("点击这里后，直接按下新的快捷键", theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	value := canvas.NewText(w.displayValue(), theme.Color(theme.ColorNamePrimary))
	value.TextStyle = fyne.TextStyle{Bold: true}
	value.TextSize = 24
	hint := canvas.NewText("支持 Ctrl / Shift / Alt / Win。按 Esc 取消。", theme.Color(theme.ColorNamePlaceHolder))
	hint.TextSize = 13

	renderer := &hotkeyCaptureRenderer{
		widget:  w,
		border:  border,
		fill:    fill,
		title:   title,
		value:   value,
		hint:    hint,
		objects: []fyne.CanvasObject{border, fill, title, value, hint},
	}
	renderer.Refresh()
	return renderer
}

func (w *hotkeyCaptureWidget) focus() {
	if canvasForObject := fyne.CurrentApp().Driver().CanvasForObject(w); canvasForObject != nil {
		canvasForObject.Focus(w)
	}
}

func (w *hotkeyCaptureWidget) displayValue() string {
	if w.pressed != 0 {
		if preview := hotkey.NormalizeShortcut(w.modifierPreview()); preview != "" {
			return preview + "+..."
		}
	}
	if w.current != "" {
		return w.current
	}
	return "等待按键"
}

func (w *hotkeyCaptureWidget) modifierPreview() string {
	value, err := hotkey.FormatShortcut(w.pressed, fyne.KeySpace)
	if err != nil {
		return ""
	}
	return value[:len(value)-len("Space")-1]
}

type hotkeyCaptureRenderer struct {
	widget  *hotkeyCaptureWidget
	border  *canvas.Rectangle
	fill    *canvas.Rectangle
	title   *canvas.Text
	value   *canvas.Text
	hint    *canvas.Text
	objects []fyne.CanvasObject
}

func (r *hotkeyCaptureRenderer) Layout(size fyne.Size) {
	r.border.Resize(size)

	fillPos := fyne.NewPos(1, 1)
	fillSize := fyne.NewSize(size.Width-2, size.Height-2)
	if fillSize.Width < 0 {
		fillSize.Width = 0
	}
	if fillSize.Height < 0 {
		fillSize.Height = 0
	}
	r.fill.Move(fillPos)
	r.fill.Resize(fillSize)

	r.title.Move(fyne.NewPos(18, 16))
	r.value.Move(fyne.NewPos(18, 46))
	r.hint.Move(fyne.NewPos(18, size.Height-30))
}

func (r *hotkeyCaptureRenderer) MinSize() fyne.Size {
	return r.widget.MinSize()
}

func (r *hotkeyCaptureRenderer) Refresh() {
	if r.widget.focused {
		r.border.FillColor = theme.Color(theme.ColorNamePrimary)
	} else {
		r.border.FillColor = color.NRGBA{R: 0xC8, G: 0xB0, B: 0x7D, A: 0xFF}
	}
	r.fill.FillColor = color.NRGBA{R: 0xFF, G: 0xFB, B: 0xF2, A: 0xFF}
	r.value.Text = r.widget.displayValue()
	r.value.Color = theme.Color(theme.ColorNamePrimary)
	r.border.Refresh()
	r.fill.Refresh()
	r.title.Refresh()
	r.value.Refresh()
	r.hint.Refresh()
}

func (r *hotkeyCaptureRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *hotkeyCaptureRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *hotkeyCaptureRenderer) Destroy() {}
