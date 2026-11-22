//go:build !darwin && !linux

package helper

import (
	"errors"
	"net"
)

func PeerUID(net.Conn) (uint32, error) {
	return 0, errors.New("helper: peer credentials are not supported on this platform")
}
