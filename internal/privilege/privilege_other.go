//go:build !darwin && !linux

package privilege

import (
	"context"
	"errors"
)

var errUnsupported = errors.New("privilege: TUN is not supported on this platform yet")

func Check(context.Context) State { return StateUnsupported }

func Install(context.Context, string, string) error { return errUnsupported }

func Uninstall(context.Context) error { return errUnsupported }
