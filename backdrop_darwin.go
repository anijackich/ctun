//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

// Sidebar material on Wails' backdrop; unlike NSGlassEffectView it ignores web content, so
// repaints don't make it flicker.
static void useSidebarMaterial(void *handle) {
	NSWindow *window = (NSWindow *)handle;
	for (NSView *view in window.contentView.subviews) {
		if (![view isKindOfClass:[NSVisualEffectView class]]) continue;
		NSVisualEffectView *effect = (NSVisualEffectView *)view;
		effect.material = NSVisualEffectMaterialSidebar;
		effect.blendingMode = NSVisualEffectBlendingModeBehindWindow;
		effect.state = NSVisualEffectStateFollowsWindowActiveState;
	}
}

// appearance: 0 system, 1 light, 2 dark. The web view and the backdrop follow the window.
static void setAppearance(void *handle, int appearance) {
	NSWindow *window = (NSWindow *)handle;
	switch (appearance) {
	case 1: window.appearance = [NSAppearance appearanceNamed:NSAppearanceNameAqua]; break;
	case 2: window.appearance = [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua]; break;
	default: window.appearance = nil;
	}
}
*/
import "C"

import "github.com/wailsapp/wails/v3/pkg/application"

func applyTheme(window *application.WebviewWindow, theme string) {
	appearance := C.int(0)
	switch theme {
	case "light":
		appearance = 1
	case "dark":
		appearance = 2
	}
	application.InvokeSync(func() {
		if handle := window.NativeWindow(); handle != nil {
			C.setAppearance(handle, appearance)
		}
	})
}

func applySidebarMaterial(window *application.WebviewWindow) {
	application.InvokeSync(func() {
		if handle := window.NativeWindow(); handle != nil {
			C.useSidebarMaterial(handle)
		}
	})
}
