package app

import "errors"

const (
	CodeTunUnavailable = "tun-unavailable"
	CodeCoreMissing    = "core-missing"
	CodeCoreFailed     = "core-failed"
	CodeNoProfile      = "no-profile"
	CodeProfileInvalid = "profile-invalid"
	CodeHelperMissing  = "helper-missing"
	CodeHelperOutdated = "helper-outdated"
	CodeSystemProxy    = "system-proxy"
)

var (
	errTunUnavailable = errors.New("TUN is not supported on this platform yet")
	errHelperMissing  = errors.New("TUN needs the system helper: install it in Settings")
	errHelperOutdated = errors.New("the system helper is outdated: reinstall it in Settings")
	errNoProfile      = errors.New("no subscription selected")
	errNotRunning     = errors.New("not connected")
	errLocalConfig    = errors.New("local configs have no subscription link")
	errInvalidLink    = errors.New("enter an http(s) subscription link")
	errNameNeeded     = errors.New("enter a name")
)
