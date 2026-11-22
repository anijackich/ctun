//go:build darwin

package helper

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

func PeerUID(conn net.Conn) (uint32, error) {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, fmt.Errorf("helper: not a unix socket")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("helper: peer credentials: %w", err)
	}
	var cred *unix.Xucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		cred, credErr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	}); err != nil {
		return 0, fmt.Errorf("helper: peer credentials: %w", err)
	}
	if credErr != nil {
		return 0, fmt.Errorf("helper: peer credentials: %w", credErr)
	}
	return cred.Uid, nil
}
