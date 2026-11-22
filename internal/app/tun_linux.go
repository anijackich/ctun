//go:build linux

package app

import (
	"ctun/internal/core"
	"ctun/internal/helper"
)

func newTunRunner(dir string, onChange func(core.Event)) runner {
	return newLocalRunner(dir, func() (string, error) { return helper.LinuxCore, nil }, onChange)
}
