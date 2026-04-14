//go:build windows

package main

import (
	"bytes"
	projectassets "egg-analyze/assets"
	"image"
	"image/color"
	stddraw "image/draw"
	_ "image/png"
	"sync"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	xdraw "golang.org/x/image/draw"
)

type nativeBubbleConfig struct {
	onTap     func()
	onDragEnd func()
	dispatch  func(func())
}

type bubbleWindowState struct {
	mu          sync.Mutex
	hwnd        uintptr
	originalWnd uintptr
	onTap       func()
	onDragEnd   func()
	dispatch    func(func())
	mouseDown   bool
	downX       int32
	downY       int32
}

type winPoint struct {
	X int32
	Y int32
}

type winSize struct {
	CX int32
	CY int32
}

type blendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type rgbQuad struct {
	Blue     byte
	Green    byte
	Red      byte
	Reserved byte
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]rgbQuad
}

var (
	procGetWindowLongPtr   = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr   = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProc     = user32.NewProc("CallWindowProcW")
	procDefWindowProc      = user32.NewProc("DefWindowProcW")
	procSetCapture         = user32.NewProc("SetCapture")
	procGetDC              = user32.NewProc("GetDC")
	procReleaseDC          = user32.NewProc("ReleaseDC")
	procUpdateLayered      = user32.NewProc("UpdateLayeredWindow")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")

	bubbleWindowConfigs sync.Map
	bubbleWindowStates  sync.Map
	bubblePNGOnce       sync.Once
	bubbleSource        image.Image
	bubbleSourceErr     error
	bubbleWndProc       = syscall.NewCallback(nativeBubbleWndProc)
)

const (
	gwlExStyle      = -20
	gwlpWndProc     = -4
	wsExLayered     = 0x00080000
	ulwAlpha        = 0x00000002
	acSrcOver       = 0x00
	acSrcAlpha      = 0x01
	biRGB           = 0
	dibRGBColors    = 0
	wmMouseMove     = 0x0200
	wmLButtonDown   = 0x0201
	wmLButtonUp     = 0x0202
	wmCaptureChange = 0x0215
	wmNCDestroy     = 0x0082
)

func signedIndex(v int32) uintptr {
	return uintptr(uint32(v))
}

func setupNativeBubbleWindow(win fyne.Window, onTap func(), onDragEnd func()) bool {
	dispatch := func(fn func()) {
		go fn()
	}
	if queued, ok := win.(interface{ QueueEvent(func()) }); ok {
		dispatch = queued.QueueEvent
	}

	bubbleWindowConfigs.Store(win, nativeBubbleConfig{
		onTap:     onTap,
		onDragEnd: onDragEnd,
		dispatch:  dispatch,
	})
	return true
}

func lookupNativeBubbleWindow(win fyne.Window) (nativeBubbleConfig, bool) {
	value, ok := bubbleWindowConfigs.Load(win)
	if !ok {
		return nativeBubbleConfig{}, false
	}
	cfg, ok := value.(nativeBubbleConfig)
	return cfg, ok
}

func configureNativeLayeredBubbleWindow(hwnd uintptr, style nativeWindowStyle, cfg nativeBubbleConfig) {
	if hwnd == 0 {
		return
	}

	exStyle, _, _ := procGetWindowLongPtr.Call(hwnd, signedIndex(gwlExStyle))
	if exStyle&wsExLayered == 0 {
		_, _, _ = procSetWindowLongPtr.Call(hwnd, signedIndex(gwlExStyle), exStyle|wsExLayered)
	}

	insertAfter := hwndNoTopMost
	if style.Floating {
		insertAfter = hwndTopMost
	}
	_, _, _ = procSetWindowPos.Call(
		hwnd,
		insertAfter,
		0,
		0,
		0,
		0,
		swpNoMove|swpNoSize|swpNoActivate|swpShowWindow,
	)
	_, _, _ = procSetWindowRgn.Call(hwnd, 0, 1)

	ensureNativeBubbleSubclass(hwnd, cfg)

	var r rect
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if ret == 0 {
		return
	}
	updateNativeBubbleBitmap(hwnd, int(r.Right-r.Left), int(r.Bottom-r.Top), r.Left, r.Top)
}

func ensureNativeBubbleSubclass(hwnd uintptr, cfg nativeBubbleConfig) {
	value, loaded := bubbleWindowStates.Load(hwnd)
	if loaded {
		state := value.(*bubbleWindowState)
		state.mu.Lock()
		state.onTap = cfg.onTap
		state.onDragEnd = cfg.onDragEnd
		state.dispatch = cfg.dispatch
		state.mu.Unlock()
		return
	}

	originalWnd, _, _ := procGetWindowLongPtr.Call(hwnd, signedIndex(gwlpWndProc))
	state := &bubbleWindowState{
		hwnd:        hwnd,
		originalWnd: originalWnd,
		onTap:       cfg.onTap,
		onDragEnd:   cfg.onDragEnd,
		dispatch:    cfg.dispatch,
	}
	actual, loaded := bubbleWindowStates.LoadOrStore(hwnd, state)
	if loaded {
		existing := actual.(*bubbleWindowState)
		existing.mu.Lock()
		existing.onTap = cfg.onTap
		existing.onDragEnd = cfg.onDragEnd
		existing.dispatch = cfg.dispatch
		existing.mu.Unlock()
		return
	}

	_, _, _ = procSetWindowLongPtr.Call(hwnd, signedIndex(gwlpWndProc), bubbleWndProc)
}

func nativeBubbleWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	value, ok := bubbleWindowStates.Load(hwnd)
	if !ok {
		ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}
	state := value.(*bubbleWindowState)

	switch msg {
	case wmLButtonDown:
		x, y := decodePoint(lParam)
		state.mu.Lock()
		state.mouseDown = true
		state.downX = x
		state.downY = y
		state.mu.Unlock()
		_, _, _ = procSetCapture.Call(hwnd)
		return 0

	case wmMouseMove:
		x, y := decodePoint(lParam)
		shouldDrag := false
		var onDragEnd func()
		dispatch := func(fn func()) {
			go fn()
		}

		state.mu.Lock()
		if state.mouseDown && (absInt32(x-state.downX) >= 2 || absInt32(y-state.downY) >= 2) {
			state.mouseDown = false
			shouldDrag = true
			onDragEnd = state.onDragEnd
		}
		if state.dispatch != nil {
			dispatch = state.dispatch
		}
		state.mu.Unlock()

		if shouldDrag {
			_, _, _ = procReleaseCapture.Call()
			_, _, _ = procSendMessage.Call(hwnd, wmNCLButtonDown, htCaption, 0)
			if onDragEnd != nil {
				dispatch(onDragEnd)
			}
			return 0
		}

	case wmLButtonUp:
		var onTap func()
		dispatch := func(fn func()) {
			go fn()
		}
		handled := false

		state.mu.Lock()
		if state.mouseDown {
			state.mouseDown = false
			onTap = state.onTap
			handled = true
		}
		if state.dispatch != nil {
			dispatch = state.dispatch
		}
		state.mu.Unlock()

		_, _, _ = procReleaseCapture.Call()
		if handled {
			if onTap != nil {
				dispatch(onTap)
			}
			return 0
		}

	case wmCaptureChange:
		state.mu.Lock()
		state.mouseDown = false
		state.mu.Unlock()

	case wmNCDestroy:
		_, _, _ = procSetWindowLongPtr.Call(hwnd, signedIndex(gwlpWndProc), state.originalWnd)
		bubbleWindowStates.Delete(hwnd)
	}

	ret, _, _ := procCallWindowProc.Call(state.originalWnd, hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func updateNativeBubbleBitmap(hwnd uintptr, width, height int, x, y int32) {
	if width <= 0 || height <= 0 {
		return
	}

	img, err := renderNativeBubbleImage(width, height)
	if err != nil {
		return
	}

	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		return
	}
	defer procReleaseDC.Call(0, screenDC)

	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		return
	}
	defer procDeleteDC.Call(memDC)

	info := bitmapInfo{
		Header: bitmapInfoHeader{
			Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			Width:       int32(width),
			Height:      -int32(height),
			Planes:      1,
			BitCount:    32,
			Compression: biRGB,
		},
	}

	var bits unsafe.Pointer
	hBitmap, _, _ := procCreateDIBSection.Call(
		memDC,
		uintptr(unsafe.Pointer(&info)),
		dibRGBColors,
		uintptr(unsafe.Pointer(&bits)),
		0,
		0,
	)
	if hBitmap == 0 || bits == nil {
		return
	}
	defer procDeleteObject.Call(hBitmap)

	oldBitmap, _, _ := procSelectObject.Call(memDC, hBitmap)
	defer procSelectObject.Call(memDC, oldBitmap)

	copyPremultipliedBGRA(bits, img)

	blend := blendFunction{
		BlendOp:             acSrcOver,
		SourceConstantAlpha: 255,
		AlphaFormat:         acSrcAlpha,
	}
	dstPoint := winPoint{X: x, Y: y}
	srcPoint := winPoint{}
	size := winSize{CX: int32(width), CY: int32(height)}

	_, _, _ = procUpdateLayered.Call(
		hwnd,
		screenDC,
		uintptr(unsafe.Pointer(&dstPoint)),
		uintptr(unsafe.Pointer(&size)),
		memDC,
		uintptr(unsafe.Pointer(&srcPoint)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ulwAlpha,
	)
}

func renderNativeBubbleImage(width, height int) (*image.NRGBA, error) {
	src, err := loadNativeBubbleSource()
	if err != nil {
		return nil, err
	}

	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	cardInset := maxInt(2, width/14)
	cardRect := image.Rect(cardInset, cardInset, width-cardInset, height-cardInset)
	cardRect = fitSquareRect(cardRect)
	if cardRect.Dx() <= 0 || cardRect.Dy() <= 0 {
		cardRect = image.Rect(0, 0, width, height)
	}

	srcSquare := centerSquare(src.Bounds())
	xdraw.CatmullRom.Scale(dst, cardRect, src, srcSquare, stddraw.Over, nil)

	radius := maxInt(8, cardRect.Dx()/5)
	applyRoundedRectAlpha(dst, cardRect, radius)
	return dst, nil
}

func loadNativeBubbleSource() (image.Image, error) {
	bubblePNGOnce.Do(func() {
		bubbleSource, _, bubbleSourceErr = image.Decode(bytes.NewReader(projectassets.BubblePNG))
	})
	return bubbleSource, bubbleSourceErr
}

func copyPremultipliedBGRA(dst unsafe.Pointer, img *image.NRGBA) {
	pixels := unsafe.Slice((*byte)(dst), len(img.Pix))
	for i := 0; i < len(img.Pix); i += 4 {
		r := uint32(img.Pix[i+0])
		g := uint32(img.Pix[i+1])
		b := uint32(img.Pix[i+2])
		a := uint32(img.Pix[i+3])

		pixels[i+0] = byte((b * a) / 255)
		pixels[i+1] = byte((g * a) / 255)
		pixels[i+2] = byte((r * a) / 255)
		pixels[i+3] = byte(a)
	}
}

func decodePoint(lParam uintptr) (int32, int32) {
	x := int32(int16(lParam & 0xFFFF))
	y := int32(int16((lParam >> 16) & 0xFFFF))
	return x, y
}

func absInt32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func fitSquareRect(r image.Rectangle) image.Rectangle {
	size := r.Dx()
	if r.Dy() < size {
		size = r.Dy()
	}
	if size <= 0 {
		return image.Rectangle{}
	}
	x := r.Min.X + (r.Dx()-size)/2
	y := r.Min.Y + (r.Dy()-size)/2
	return image.Rect(x, y, x+size, y+size)
}

func centerSquare(r image.Rectangle) image.Rectangle {
	size := r.Dx()
	if r.Dy() < size {
		size = r.Dy()
	}
	if size <= 0 {
		return image.Rectangle{}
	}
	x := r.Min.X + (r.Dx()-size)/2
	y := r.Min.Y + (r.Dy()-size)/2
	return image.Rect(x, y, x+size, y+size)
}

func applyRoundedRectAlpha(img *image.NRGBA, rect image.Rectangle, radius int) {
	if img == nil || rect.Empty() {
		return
	}
	if radius < 1 {
		return
	}
	maxRadius := rect.Dx() / 2
	if rect.Dy()/2 < maxRadius {
		maxRadius = rect.Dy() / 2
	}
	if radius > maxRadius {
		radius = maxRadius
	}

	corners := [4]image.Point{
		{X: rect.Min.X + radius, Y: rect.Min.Y + radius},
		{X: rect.Max.X - radius - 1, Y: rect.Min.Y + radius},
		{X: rect.Min.X + radius, Y: rect.Max.Y - radius - 1},
		{X: rect.Max.X - radius - 1, Y: rect.Max.Y - radius - 1},
	}

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if insideRoundedRect(x, y, rect, radius, corners) {
				continue
			}
			img.SetNRGBA(x, y, color.NRGBA{})
		}
	}
}

func insideRoundedRect(x, y int, rect image.Rectangle, radius int, corners [4]image.Point) bool {
	if x >= rect.Min.X+radius && x < rect.Max.X-radius {
		return true
	}
	if y >= rect.Min.Y+radius && y < rect.Max.Y-radius {
		return true
	}

	for _, c := range corners {
		dx := x - c.X
		dy := y - c.Y
		if dx*dx+dy*dy <= radius*radius {
			return true
		}
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
