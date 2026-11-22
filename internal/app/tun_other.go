//go:build !darwin && !linux

package app

import (
	"errors"

	"ctun/internal/core"
)

func newTunRunner(dir string, onChange func(core.Event)) runner {
	return newLocalRunner(dir, func() (string, error) {
		return "", errors.New("TUN is not supported on this platform yet")
	}, onChange)
}
