//go:build darwin

package app

import (
	"ctun/internal/core"
	"ctun/internal/helper"
)

func newTunRunner(_ string, onChange func(core.Event)) runner {
	return &helper.Runner{Socket: helper.SocketPath, OnChange: onChange}
}
