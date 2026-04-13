//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>

static void codexConfigureWindow(uintptr_t windowPtr, CGFloat cornerRadius, int floating, int transparent, int movableByBackground) {
	if (windowPtr == 0) {
		return;
	}

	NSWindow *window = (__bridge NSWindow *)(void *)windowPtr;
	if (window == nil) {
		return;
	}

	NSWindowCollectionBehavior behavior =
		NSWindowCollectionBehaviorCanJoinAllSpaces |
		NSWindowCollectionBehaviorFullScreenAuxiliary |
		NSWindowCollectionBehaviorStationary;
	[window setCollectionBehavior:behavior];
	[window setReleasedWhenClosed:NO];
	[window setHidesOnDeactivate:NO];
	[window setMovableByWindowBackground:(movableByBackground ? YES : NO)];
	[window setOpaque:NO];
	[window setHasShadow:YES];
	[window setBackgroundColor:[NSColor clearColor]];
	[window setIgnoresMouseEvents:NO];
	[window setLevel:(floating ? NSStatusWindowLevel : NSNormalWindowLevel)];
	[window setTitlebarAppearsTransparent:YES];
	[window setTitleVisibility:NSWindowTitleHidden];

	NSView *contentView = [window contentView];
	if (contentView != nil) {
		[contentView setWantsLayer:YES];
		contentView.layer.cornerRadius = cornerRadius;
		contentView.layer.masksToBounds = YES;
		contentView.layer.backgroundColor = (transparent ? [[NSColor clearColor] CGColor] : [[NSColor colorWithWhite:1 alpha:0.001] CGColor]);

		NSView *frameView = [contentView superview];
		if (frameView != nil) {
			[frameView setWantsLayer:YES];
			frameView.layer.cornerRadius = cornerRadius;
			frameView.layer.masksToBounds = YES;
			frameView.layer.backgroundColor = [[NSColor clearColor] CGColor];
		}
	}

	[window orderFrontRegardless];
}
*/
import "C"

import (
	"fyne.io/fyne/v2"
	fynedriver "fyne.io/fyne/v2/driver"
)

type nativeWindowStyle struct {
	CornerRadius        float64
	Floating            bool
	Transparent         bool
	MovableByBackground bool
}

func configureNativeWindow(win fyne.Window, style nativeWindowStyle) {
	native, ok := win.(fynedriver.NativeWindow)
	if !ok {
		return
	}

	native.RunNative(func(context any) {
		macCtx, ok := context.(fynedriver.MacWindowContext)
		if !ok || macCtx.NSWindow == 0 {
			return
		}

		C.codexConfigureWindow(
			C.uintptr_t(macCtx.NSWindow),
			C.CGFloat(style.CornerRadius),
			boolToObjc(style.Floating),
			boolToObjc(style.Transparent),
			boolToObjc(style.MovableByBackground),
		)
	})
}

func boolToObjc(v bool) C.int {
	if v {
		return 1
	}
	return 0
}
