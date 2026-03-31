package sidecar

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const BypassHeader = "X-Sub2API-Rust-Sidecar-Bypass"

var relayBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 64*1024)
		return &buf
	},
}

func HasBypassHeader(req *http.Request) bool {
	if req == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(req.Header.Get(BypassHeader)), "1")
}

func TunnelUpgradedRequest(req *http.Request, hijacker http.Hijacker, socketPath string) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}
	if hijacker == nil {
		return fmt.Errorf("hijacker is nil")
	}
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return fmt.Errorf("sidecar socket path is empty")
	}

	clientConn, rw, err := hijacker.Hijack()
	if err != nil {
		return err
	}

	sidecarConn, err := (&net.Dialer{Timeout: 5 * time.Second}).Dial("unix", socketPath)
	if err != nil {
		_, _ = clientConn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\nContent-Type: application/json\r\nContent-Length: 61\r\n\r\n{\"code\":\"RUST_SIDECAR_UNAVAILABLE\",\"message\":\"sidecar unavailable\"}"))
		_ = clientConn.Close()
		return err
	}

	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()
	cloned.Header.Set(BypassHeader, "1")
	if err := cloned.Write(sidecarConn); err != nil {
		_ = sidecarConn.Close()
		_ = clientConn.Close()
		return err
	}

	if rw != nil && rw.Reader != nil {
		if buffered := rw.Reader.Buffered(); buffered > 0 {
			data, err := rw.Reader.Peek(buffered)
			if err == nil && len(data) > 0 {
				if _, writeErr := sidecarConn.Write(data); writeErr != nil {
					_ = sidecarConn.Close()
					_ = clientConn.Close()
					return writeErr
				}
				_, _ = rw.Reader.Discard(buffered)
			}
		}
	}
	done := make(chan struct{}, 2)

	go func() {
		relayCopyOneWay(sidecarConn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		relayCopyOneWay(clientConn, sidecarConn)
		done <- struct{}{}
	}()
	go func() {
		<-done
		<-done
		_ = clientConn.Close()
		_ = sidecarConn.Close()
	}()
	return nil
}

func relayCopyOneWay(dst net.Conn, src net.Conn) (int64, error) {
	bufPtr := relayBufferPool.Get().(*[]byte)
	defer relayBufferPool.Put(bufPtr)
	if dst == nil || src == nil {
		return 0, nil
	}
	n, err := io.CopyBuffer(dst, src, *bufPtr)
	relayCloseWrite(dst)
	relayCloseRead(src)
	return n, err
}

type relayCloseWriter interface {
	CloseWrite() error
}

type relayCloseReader interface {
	CloseRead() error
}

func relayCloseWrite(conn net.Conn) {
	if conn == nil {
		return
	}
	if cw, ok := conn.(relayCloseWriter); ok {
		_ = cw.CloseWrite()
	}
}

func relayCloseRead(conn net.Conn) {
	if conn == nil {
		return
	}
	if cr, ok := conn.(relayCloseReader); ok {
		_ = cr.CloseRead()
	}
}
