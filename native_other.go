//go:build !darwin

package main

import "fyne.io/fyne/v2"

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

func configureNativeWindow(fyne.Window, nativeWindowStyle) {}

func getNativeWindowFrame(fyne.Window) (nativeWindowFrame, bool) {
	return nativeWindowFrame{}, false
}

func getNativeVisibleFrame(fyne.Window) (nativeWindowFrame, bool) {
	return nativeWindowFrame{}, false
}

func setNativeWindowOrigin(fyne.Window, float32, float32) {}
