//go:build !linux

package server

import (
	"context"
	"net"
)

func listenOptimizedBaseListener(_ context.Context, network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

func optimizeAcceptedConn(conn net.Conn) {
	_ = conn
}
