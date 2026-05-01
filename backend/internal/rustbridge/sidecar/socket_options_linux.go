//go:build linux

package sidecar

import (
	"net"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const relayUnixSocketBufferBytes = 256 * 1024

func configureSidecarDialer(dialer *net.Dialer) {
	if dialer == nil {
		return
	}
	previous := dialer.Control
	dialer.Control = func(network, address string, conn syscall.RawConn) error {
		if previous != nil {
			if err := previous(network, address, conn); err != nil {
				return err
			}
		}
		tuneRawSocket(network, conn)
		return nil
	}
}

func tuneRelayConn(conn net.Conn) {
	if conn == nil {
		return
	}
	sysConn, ok := conn.(syscall.Conn)
	if !ok {
		return
	}
	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return
	}
	network := ""
	if addr := conn.LocalAddr(); addr != nil {
		network = addr.Network()
	}
	tuneRawSocket(network, rawConn)
}

func tuneRawSocket(network string, rawConn syscall.RawConn) {
	network = strings.ToLower(network)
	_ = rawConn.Control(func(fd uintptr) {
		if strings.HasPrefix(network, "tcp") {
			_ = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_NODELAY, 1)
			_ = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_QUICKACK, 1)
		}
		if strings.HasPrefix(network, "unix") {
			_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_RCVBUF, relayUnixSocketBufferBytes)
			_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_SNDBUF, relayUnixSocketBufferBytes)
		}
	})
}
