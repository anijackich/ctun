//go:build !unix

package core

import "context"

func writePID(string, int) error { return nil }

func removePID(string) {}

func StopOrphan(context.Context, string) (bool, error) { return false, nil }
