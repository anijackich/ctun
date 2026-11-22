//go:build !darwin

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func applySidebarMaterial(*application.WebviewWindow) {}

func applyTheme(*application.WebviewWindow, string) {}
