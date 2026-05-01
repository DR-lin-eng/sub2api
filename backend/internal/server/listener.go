package server

import (
	"context"
	"fmt"
	"net"
	"os"
)

type optimizedListener struct {
	net.Listener
}

func ListenTCPOptimized(network, address string) (net.Listener, error) {
	base, err := listenOptimizedBaseListener(context.Background(), network, address)
	if err != nil {
		return nil, err
	}
	return &optimizedListener{Listener: base}, nil
}

func (l *optimizedListener) Accept() (net.Conn, error) {
	if l == nil || l.Listener == nil {
		return nil, fmt.Errorf("listener is nil")
	}
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	optimizeAcceptedConn(conn)
	return conn, nil
}

func (l *optimizedListener) File() (*os.File, error) {
	type fileListener interface {
		File() (*os.File, error)
	}
	if l == nil || l.Listener == nil {
		return nil, fmt.Errorf("listener is nil")
	}
	fl, ok := l.Listener.(fileListener)
	if !ok {
		return nil, fmt.Errorf("listener type %T does not expose File()", l.Listener)
	}
	return fl.File()
}
