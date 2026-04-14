//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>

typedef struct {
	CGFloat x;
	CGFloat y;
	CGFloat width;
	CGFloat height;
} CodexRect;

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

static CodexRect codexGetWindowFrame(uintptr_t windowPtr) {
	CodexRect rect = {0, 0, 0, 0};
	if (windowPtr == 0) {
		return rect;
	}

	NSWindow *window = (__bridge NSWindow *)(void *)windowPtr;
	if (window == nil) {
		return rect;
	}

	NSRect frame = [window frame];
	rect.x = frame.origin.x;
	rect.y = frame.origin.y;
	rect.width = frame.size.width;
	rect.height = frame.size.height;
	return rect;
}

static CodexRect codexGetVisibleFrame(uintptr_t windowPtr) {
	CodexRect rect = {0, 0, 0, 0};
	if (windowPtr == 0) {
		return rect;
	}

	NSWindow *window = (__bridge NSWindow *)(void *)windowPtr;
	if (window == nil) {
		return rect;
	}

	NSScreen *screen = [window screen];
	if (screen == nil) {
		screen = [NSScreen mainScreen];
	}
	if (screen == nil) {
		return rect;
	}

	NSRect frame = [screen visibleFrame];
	rect.x = frame.origin.x;
	rect.y = frame.origin.y;
	rect.width = frame.size.width;
	rect.height = frame.size.height;
	return rect;
}

static void codexSetWindowOrigin(uintptr_t windowPtr, CGFloat x, CGFloat y) {
	if (windowPtr == 0) {
		return;
	}

	NSWindow *window = (__bridge NSWindow *)(void *)windowPtr;
	if (window == nil) {
		return;
	}

	NSRect frame = [window frame];
	frame.origin = NSMakePoint(x, y);
	[window setFrame:frame display:YES];
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

type nativeWindowFrame struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
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

func getNativeWindowFrame(win fyne.Window) (nativeWindowFrame, bool) {
	native, ok := win.(fynedriver.NativeWindow)
	if !ok {
		return nativeWindowFrame{}, false
	}

	frame := nativeWindowFrame{}
	found := false
	native.RunNative(func(context any) {
		macCtx, ok := context.(fynedriver.MacWindowContext)
		if !ok || macCtx.NSWindow == 0 {
			return
		}
		rect := C.codexGetWindowFrame(C.uintptr_t(macCtx.NSWindow))
		frame = nativeWindowFrame{
			X:      float32(rect.x),
			Y:      float32(rect.y),
			Width:  float32(rect.width),
			Height: float32(rect.height),
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
		macCtx, ok := context.(fynedriver.MacWindowContext)
		if !ok || macCtx.NSWindow == 0 {
			return
		}
		rect := C.codexGetVisibleFrame(C.uintptr_t(macCtx.NSWindow))
		frame = nativeWindowFrame{
			X:      float32(rect.x),
			Y:      float32(rect.y),
			Width:  float32(rect.width),
			Height: float32(rect.height),
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
		macCtx, ok := context.(fynedriver.MacWindowContext)
		if !ok || macCtx.NSWindow == 0 {
			return
		}
		C.codexSetWindowOrigin(C.uintptr_t(macCtx.NSWindow), C.CGFloat(x), C.CGFloat(y))
	})
}

func setupNativeBubbleWindow(fyne.Window, func(), func()) bool {
	return false
}

func beginNativeWindowDrag(fyne.Window) {}

func boolToObjc(v bool) C.int {
	if v {
		return 1
	}
	return 0
}
