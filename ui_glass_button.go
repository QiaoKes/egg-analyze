package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type glassButtonStyle int

const (
	glassButtonSecondary glassButtonStyle = iota
	glassButtonPrimary
	glassButtonCompact
)

type glassButton struct {
	widget.BaseWidget
	label string
	icon  fyne.Resource
	style glassButtonStyle
	onTap func()
}

func newGlassButton(label string, icon fyne.Resource, style glassButtonStyle, onTap func()) *glassButton {
	b := &glassButton{
		label: label,
		icon:  icon,
		style: style,
		onTap: onTap,
	}
	b.ExtendBaseWidget(b)
	return b
}

func (b *glassButton) MinSize() fyne.Size {
	switch b.style {
	case glassButtonCompact:
		return fyne.NewSize(50, 50)
	case glassButtonPrimary:
		return fyne.NewSize(180, 68)
	default:
		return fyne.NewSize(180, 62)
	}
}

func (b *glassButton) Tapped(*fyne.PointEvent) {
	if b.onTap != nil {
		b.onTap()
	}
}

func (b *glassButton) TappedSecondary(*fyne.PointEvent) {}

func (b *glassButton) CreateRenderer() fyne.WidgetRenderer {
	shadow := canvas.NewRectangle(color.NRGBA{R: 0x10, G: 0x17, B: 0x24, A: 0x08})
	shadow.CornerRadius = 18
	bg := canvas.NewRectangle(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x5E})
	bg.CornerRadius = 18
	shine := canvas.NewRectangle(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x18})
	shine.CornerRadius = 16
	border := canvas.NewRectangle(color.Transparent)
	border.CornerRadius = 18
	border.StrokeWidth = 1
	border.StrokeColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x54}

	var icon *widget.Icon
	if b.icon != nil {
		icon = widget.NewIcon(b.icon)
	} else {
		icon = widget.NewIcon(theme.ViewRefreshIcon())
		icon.Hide()
	}
	label := canvas.NewText(b.label, color.NRGBA{R: 0x18, G: 0x1F, B: 0x2C, A: 0xFF})
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.TextSize = 18

	r := &glassButtonRenderer{
		button:  b,
		shadow:  shadow,
		bg:      bg,
		shine:   shine,
		border:  border,
		icon:    icon,
		label:   label,
		objects: []fyne.CanvasObject{shadow, bg, shine, border, icon, label},
	}
	r.applyStyle()
	return r
}

type glassButtonRenderer struct {
	button  *glassButton
	shadow  *canvas.Rectangle
	bg      *canvas.Rectangle
	shine   *canvas.Rectangle
	border  *canvas.Rectangle
	icon    *widget.Icon
	label   *canvas.Text
	objects []fyne.CanvasObject
}

func (r *glassButtonRenderer) applyStyle() {
	switch r.button.style {
	case glassButtonPrimary:
		r.bg.FillColor = color.NRGBA{R: 0x76, G: 0xA1, B: 0xF8, A: 0xB8}
		r.border.StrokeColor = color.NRGBA{R: 0xE2, G: 0xEC, B: 0xFF, A: 0x72}
		r.shine.FillColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x14}
		r.label.Color = color.White
	case glassButtonCompact:
		r.bg.FillColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x46}
		r.border.StrokeColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x4E}
		r.shine.FillColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x10}
		r.label.Color = color.NRGBA{R: 0x18, G: 0x1F, B: 0x2C, A: 0xFF}
	default:
		r.bg.FillColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x42}
		r.border.StrokeColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x46}
		r.shine.FillColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x10}
		r.label.Color = color.NRGBA{R: 0x18, G: 0x1F, B: 0x2C, A: 0xFF}
	}
	if r.button.icon != nil {
		r.icon.SetResource(r.button.icon)
		r.icon.Show()
	} else {
		r.icon.Hide()
	}
	r.label.Text = r.button.label
	r.label.Refresh()
	canvas.Refresh(r.bg)
	canvas.Refresh(r.shine)
	canvas.Refresh(r.border)
}

func (r *glassButtonRenderer) Layout(size fyne.Size) {
	r.shadow.Move(fyne.NewPos(0, 4))
	r.shadow.Resize(fyne.NewSize(size.Width, size.Height-1))
	r.bg.Resize(size)
	r.border.Resize(size)

	shineHeight := fyne.Max(14, size.Height*0.34)
	r.shine.Move(fyne.NewPos(4, 4))
	r.shine.Resize(fyne.NewSize(size.Width-8, shineHeight))

	if r.button.style == glassButtonCompact {
		iconSize := fyne.NewSize(18, 18)
		r.icon.Move(fyne.NewPos((size.Width-iconSize.Width)/2, (size.Height-iconSize.Height)/2))
		r.icon.Resize(iconSize)
		r.label.Hide()
		return
	}

	r.label.Show()
	iconSize := fyne.NewSize(22, 22)
	labelSize := r.label.MinSize()
	totalWidth := labelSize.Width
	if r.button.icon != nil {
		totalWidth += iconSize.Width + 10
	}
	startX := (size.Width - totalWidth) / 2
	centerY := size.Height / 2

	if r.button.icon != nil {
		r.icon.Move(fyne.NewPos(startX, centerY-iconSize.Height/2))
		r.icon.Resize(iconSize)
		r.label.Move(fyne.NewPos(startX+iconSize.Width+10, centerY-labelSize.Height/2))
	} else {
		r.label.Move(fyne.NewPos(startX, centerY-labelSize.Height/2))
	}
	r.label.Resize(labelSize)
}

func (r *glassButtonRenderer) MinSize() fyne.Size {
	return r.button.MinSize()
}

func (r *glassButtonRenderer) Refresh() {
	r.applyStyle()
	r.icon.Refresh()
}

func (r *glassButtonRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *glassButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *glassButtonRenderer) Destroy() {}

var _ fyne.WidgetRenderer = (*glassButtonRenderer)(nil)
var _ fyne.Tappable = (*glassButton)(nil)
var _ fyne.CanvasObject = (*glassButton)(nil)

func newPrimaryCaptureButton(onTap func()) *glassButton {
	return newGlassButton("截图分析", nil, glassButtonPrimary, onTap)
}

func newSecondaryActionButton(label string, icon fyne.Resource, onTap func()) *glassButton {
	return newGlassButton(label, icon, glassButtonSecondary, onTap)
}

func newCompactGlassButton(icon fyne.Resource, onTap func()) *glassButton {
	return newGlassButton("", icon, glassButtonCompact, onTap)
}
