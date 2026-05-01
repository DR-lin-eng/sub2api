//go:build !linux

package sidecar

import "net"

func configureSidecarDialer(dialer *net.Dialer) {
}

func tuneRelayConn(conn net.Conn) {
}
