//go:build !darwin

package main

import "fyne.io/fyne/v2"

type nativeWindowStyle struct {
	CornerRadius        float64
	Floating            bool
	Transparent         bool
	MovableByBackground bool
}

func configureNativeWindow(fyne.Window, nativeWindowStyle) {}
