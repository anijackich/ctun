//go:build !darwin

package sysnet

func newDNSBackend(runner) backend[[]string] { return nil }
