//go:build !darwin && !linux

package sysnet

func newProxyBackend(runner) backend[Proxy] { return nil }
