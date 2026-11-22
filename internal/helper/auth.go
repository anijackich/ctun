package helper

import (
	"fmt"
	"net"
)

func AllowUID(uid uint32) func(net.Conn) error {
	return func(conn net.Conn) error {
		peer, err := PeerUID(conn)
		if err != nil {
			return err
		}
		if peer != uid {
			return fmt.Errorf("helper: uid %d is not allowed", peer)
		}
		return nil
	}
}
