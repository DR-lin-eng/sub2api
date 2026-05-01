//go:build linux

package server

import (
	"context"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	listenerFastOpenQueue    = 256
	listenerDeferAcceptSec   = 1
	connKeepAliveIdleSec     = 60
	connKeepAliveIntervalSec = 15
	connKeepAliveProbes      = 4
)

func listenOptimizedBaseListener(ctx context.Context, network, address string) (net.Listener, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var sockErr error
			if err := c.Control(func(fd uintptr) {
				sockErr = optimizeListenerFD(int(fd))
			}); err != nil {
				return err
			}
			return sockErr
		},
	}
	return lc.Listen(ctx, network, address)
}

func optimizeListenerFD(fd int) error {
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	_ = unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_FASTOPEN, listenerFastOpenQueue)
	_ = unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_DEFER_ACCEPT, listenerDeferAcceptSec)
	return nil
}

func optimizeAcceptedConn(conn net.Conn) {
	type syscallConn interface {
		SyscallConn() (syscall.RawConn, error)
	}

	sc, ok := conn.(syscallConn)
	if !ok {
		return
	}
	rawConn, err := sc.SyscallConn()
	if err != nil {
		return
	}
	_ = rawConn.Control(func(fd uintptr) {
		socketFD := int(fd)
		_ = unix.SetsockoptInt(socketFD, unix.IPPROTO_TCP, unix.TCP_NODELAY, 1)
		_ = unix.SetsockoptInt(socketFD, unix.IPPROTO_TCP, unix.TCP_QUICKACK, 1)
		_ = unix.SetsockoptInt(socketFD, unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1)
		_ = unix.SetsockoptInt(socketFD, unix.IPPROTO_TCP, unix.TCP_KEEPIDLE, connKeepAliveIdleSec)
		_ = unix.SetsockoptInt(socketFD, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, connKeepAliveIntervalSec)
		_ = unix.SetsockoptInt(socketFD, unix.IPPROTO_TCP, unix.TCP_KEEPCNT, connKeepAliveProbes)
	})
}
