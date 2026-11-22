//go:build !darwin && !linux

package main

import "errors"

var errUnsupported = errors.New("not supported on this platform yet")

func serveCmd([]string) error   { return errUnsupported }
func installCmd([]string) error { return errUnsupported }
func uninstall() error          { return errUnsupported }
