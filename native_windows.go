//go:build windows

package main

import (
	"bytes"
	"fyne.io/fyne/v2"
	fynedriver "fyne.io/fyne/v2/driver"
	"golang.org/x/sys/windows"
	"image"
	_ "image/png"
	"sync"
	"unsafe"
)

type nativeWindowStyle struct {
	CornerRadius        float64
	Floating            bool
	Transparent         bool
	MovableByBackground bool
}

type nativeWindowFrame struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	gdi32                = windows.NewLazySystemDLL("gdi32.dll")
	procReleaseCapture   = user32.NewProc("ReleaseCapture")
	procSendMessage      = user32.NewProc("SendMessageW")
	procGetWindowRect    = user32.NewProc("GetWindowRect")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procMonitorFromWin   = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo   = user32.NewProc("GetMonitorInfoW")
	procSetWindowRgn     = user32.NewProc("SetWindowRgn")
	procCreateRectRgn    = gdi32.NewProc("CreateRectRgn")
	procCreateRoundRgn   = gdi32.NewProc("CreateRoundRectRgn")
	procCreateEllipseRgn = gdi32.NewProc("CreateEllipticRgn")
	procCombineRgn       = gdi32.NewProc("CombineRgn")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
)

const (
	monDefaultToNearest = 2
	wmNCLButtonDown     = 0x00A1
	htCaption           = 2

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010
	swpShowWindow = 0x0040

	hwndTopMost   = ^uintptr(0) // ((HWND)-1)
	hwndNoTopMost = ^uintptr(1) // ((HWND)-2)

	rgnOr = 2
)

type rect struct {
	Left, Top, Right, Bottom int32
}

type monitorInfo struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
}

var (
	bubbleMaskOnce sync.Once
	bubbleMaskImg  image.Image
	bubbleMaskErr  error
)

func configureNativeWindow(win fyne.Window, style nativeWindowStyle) {
	native, ok := win.(fynedriver.NativeWindow)
	if !ok {
		return
	}

	native.RunNative(func(context any) {
		winCtx, ok := context.(fynedriver.WindowsWindowContext)
		if !ok || winCtx.HWND == 0 {
			return
		}

		insertAfter := hwndNoTopMost
		if style.Floating {
			insertAfter = hwndTopMost
		}
		_, _, _ = procSetWindowPos.Call(
			winCtx.HWND,
			insertAfter,
			0,
			0,
			0,
			0,
			swpNoMove|swpNoSize|swpNoActivate|swpShowWindow,
		)

		var r rect
		ret, _, _ := procGetWindowRect.Call(winCtx.HWND, uintptr(unsafePointer(&r)))
		if ret != 0 {
			width := r.Right - r.Left
			height := r.Bottom - r.Top
			rgn := createWindowRegion(width, height, style)
			if rgn != 0 {
				ret, _, _ := procSetWindowRgn.Call(winCtx.HWND, rgn, 1)
				if ret == 0 {
					_, _, _ = procDeleteObject.Call(rgn)
				}
			}
		}
	})
}

func beginNativeWindowDrag(win fyne.Window) {
	native, ok := win.(fynedriver.NativeWindow)
	if !ok {
		return
	}

	native.RunNative(func(context any) {
		winCtx, ok := context.(fynedriver.WindowsWindowContext)
		if !ok || winCtx.HWND == 0 {
			return
		}
		_, _, _ = procReleaseCapture.Call()
		_, _, _ = procSendMessage.Call(winCtx.HWND, wmNCLButtonDown, htCaption, 0)
	})
}

func getNativeWindowFrame(win fyne.Window) (nativeWindowFrame, bool) {
	native, ok := win.(fynedriver.NativeWindow)
	if !ok {
		return nativeWindowFrame{}, false
	}

	frame := nativeWindowFrame{}
	found := false
	native.RunNative(func(context any) {
		winCtx, ok := context.(fynedriver.WindowsWindowContext)
		if !ok || winCtx.HWND == 0 {
			return
		}

		var r rect
		ret, _, _ := procGetWindowRect.Call(winCtx.HWND, uintptr(unsafePointer(&r)))
		if ret == 0 {
			return
		}

		frame = nativeWindowFrame{
			X:      float32(r.Left),
			Y:      float32(r.Top),
			Width:  float32(r.Right - r.Left),
			Height: float32(r.Bottom - r.Top),
		}
		found = true
	})
	return frame, found
}

func getNativeVisibleFrame(win fyne.Window) (nativeWindowFrame, bool) {
	native, ok := win.(fynedriver.NativeWindow)
	if !ok {
		return nativeWindowFrame{}, false
	}

	frame := nativeWindowFrame{}
	found := false
	native.RunNative(func(context any) {
		winCtx, ok := context.(fynedriver.WindowsWindowContext)
		if !ok || winCtx.HWND == 0 {
			return
		}

		monitor, _, _ := procMonitorFromWin.Call(winCtx.HWND, monDefaultToNearest)
		if monitor == 0 {
			return
		}

		info := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
		ret, _, _ := procGetMonitorInfo.Call(monitor, uintptr(unsafePointer(&info)))
		if ret == 0 {
			return
		}

		frame = nativeWindowFrame{
			X:      float32(info.RcWork.Left),
			Y:      float32(info.RcWork.Top),
			Width:  float32(info.RcWork.Right - info.RcWork.Left),
			Height: float32(info.RcWork.Bottom - info.RcWork.Top),
		}
		found = true
	})
	return frame, found
}

func setNativeWindowOrigin(win fyne.Window, x, y float32) {
	native, ok := win.(fynedriver.NativeWindow)
	if !ok {
		return
	}

	native.RunNative(func(context any) {
		winCtx, ok := context.(fynedriver.WindowsWindowContext)
		if !ok || winCtx.HWND == 0 {
			return
		}

		_, _, _ = procSetWindowPos.Call(
			winCtx.HWND,
			0,
			uintptr(int32(x)),
			uintptr(int32(y)),
			0,
			0,
			swpNoSize|swpNoZOrder|swpNoActivate|swpShowWindow,
		)
	})
}

func unsafePointer[T any](v *T) uintptr {
	return uintptr(unsafe.Pointer(v))
}

func createWindowRegion(width, height int32, style nativeWindowStyle) uintptr {
	if width <= 0 || height <= 0 {
		return 0
	}
	if width == height && width <= 64 {
		if rgn := createBubbleAlphaRegion(width, height); rgn != 0 {
			return rgn
		}
	}
	if style.CornerRadius*2 >= float64(minInt32(width, height))-2 {
		rgn, _, _ := procCreateEllipseRgn.Call(0, 0, uintptr(width+1), uintptr(height+1))
		return rgn
	}

	diameter := int32(style.CornerRadius * 2)
	if diameter < 2 {
		diameter = 2
	}
	rgn, _, _ := procCreateRoundRgn.Call(
		0,
		0,
		uintptr(width+1),
		uintptr(height+1),
		uintptr(diameter),
		uintptr(diameter),
	)
	return rgn
}

func minInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func createBubbleAlphaRegion(width, height int32) uintptr {
	img, err := loadBubbleMaskImage()
	if err != nil || img == nil {
		return 0
	}

	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return 0
	}

	mainRgn, _, _ := procCreateRectRgn.Call(0, 0, 0, 0)
	if mainRgn == 0 {
		return 0
	}

	const alphaThreshold = 72
	for y := int32(0); y < height; y++ {
		srcY := bounds.Min.Y + int((float64(y)+0.5)*float64(srcH)/float64(height))
		if srcY >= bounds.Max.Y {
			srcY = bounds.Max.Y - 1
		}

		runStart := int32(-1)
		for x := int32(0); x < width; x++ {
			srcX := bounds.Min.X + int((float64(x)+0.5)*float64(srcW)/float64(width))
			if srcX >= bounds.Max.X {
				srcX = bounds.Max.X - 1
			}

			_, _, _, a := img.At(srcX, srcY).RGBA()
			opaque := uint8(a>>8) >= alphaThreshold
			if opaque && runStart < 0 {
				runStart = x
			}
			if (!opaque || x == width-1) && runStart >= 0 {
				runEnd := x
				if opaque && x == width-1 {
					runEnd = x + 1
				}
				tmpRgn, _, _ := procCreateRectRgn.Call(
					uintptr(runStart),
					uintptr(y),
					uintptr(runEnd),
					uintptr(y+1),
				)
				if tmpRgn != 0 {
					_, _, _ = procCombineRgn.Call(mainRgn, mainRgn, tmpRgn, rgnOr)
					_, _, _ = procDeleteObject.Call(tmpRgn)
				}
				runStart = -1
			}
		}
	}

	return mainRgn
}

func loadBubbleMaskImage() (image.Image, error) {
	bubbleMaskOnce.Do(func() {
		bubbleMaskImg, _, bubbleMaskErr = image.Decode(bytes.NewReader(bubblePNG))
	})
	return bubbleMaskImg, bubbleMaskErr
}
