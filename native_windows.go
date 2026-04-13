//go:build windows

package main

import (
	"fyne.io/fyne/v2"
	fynedriver "fyne.io/fyne/v2/driver"
	"golang.org/x/sys/windows"
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
	user32             = windows.NewLazySystemDLL("user32.dll")
	gdi32              = windows.NewLazySystemDLL("gdi32.dll")
	procGetWindowRect  = user32.NewProc("GetWindowRect")
	procSetWindowPos   = user32.NewProc("SetWindowPos")
	procMonitorFromWin = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo = user32.NewProc("GetMonitorInfoW")
	procSetWindowRgn   = user32.NewProc("SetWindowRgn")
	procCreateRoundRgn = gdi32.NewProc("CreateRoundRectRgn")
	procDeleteObject   = gdi32.NewProc("DeleteObject")
)

const (
	monDefaultToNearest = 2

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010
	swpShowWindow = 0x0040

	hwndTopMost   = ^uintptr(0) // ((HWND)-1)
	hwndNoTopMost = ^uintptr(1) // ((HWND)-2)
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
			diameter := int32(style.CornerRadius * 2)
			if diameter < 2 {
				diameter = 2
			}
			rgn, _, _ := procCreateRoundRgn.Call(
				0,
				0,
				uintptr((r.Right-r.Left)+1),
				uintptr((r.Bottom-r.Top)+1),
				uintptr(diameter),
				uintptr(diameter),
			)
			if rgn != 0 {
				ret, _, _ := procSetWindowRgn.Call(winCtx.HWND, rgn, 1)
				if ret == 0 {
					_, _, _ = procDeleteObject.Call(rgn)
				}
			}
		}
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
