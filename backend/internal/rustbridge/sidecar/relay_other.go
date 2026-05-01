//go:build !linux

package sidecar

import "net"

func relayCopyPlatform(dst net.Conn, src net.Conn) (int64, bool, error) {
	return 0, false, nil
}
