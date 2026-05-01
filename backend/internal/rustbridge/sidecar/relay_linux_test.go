//go:build linux

package sidecar

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"testing"
)

func TestRelayCopyOneWayLinuxSocketPair(t *testing.T) {
	srcRelay, srcPeer := newRelayTCPPair(t)
	dstRelay, dstPeer := newRelayUnixPair(t)
	defer srcRelay.Close()
	defer srcPeer.Close()
	defer dstRelay.Close()
	defer dstPeer.Close()
	setRelayTestDeadline(t, srcRelay, srcPeer, dstRelay, dstPeer)

	payload := bytes.Repeat([]byte("linux-splice-relay:"), 32768)
	copyErr := make(chan error, 1)
	readErr := make(chan error, 1)
	writeErr := make(chan error, 1)

	go func() {
		n, err := relayCopyOneWay(dstRelay, srcRelay)
		if err != nil {
			copyErr <- err
			return
		}
		if n != int64(len(payload)) {
			copyErr <- fmt.Errorf("relay copied %d bytes, want %d", n, len(payload))
			return
		}
		copyErr <- nil
	}()

	go func() {
		got, err := io.ReadAll(dstPeer)
		if err != nil {
			readErr <- err
			return
		}
		if !bytes.Equal(got, payload) {
			readErr <- fmt.Errorf("relay payload mismatch: got %d bytes, want %d", len(got), len(payload))
			return
		}
		readErr <- nil
	}()

	go func() {
		if _, err := srcPeer.Write(payload); err != nil {
			writeErr <- err
			return
		}
		relayCloseWrite(srcPeer)
		writeErr <- nil
	}()

	if err := <-writeErr; err != nil {
		t.Fatalf("write source payload: %v", err)
	}
	if err := <-copyErr; err != nil {
		t.Fatalf("copy relay: %v", err)
	}
	if err := <-readErr; err != nil {
		t.Fatalf("read relayed payload: %v", err)
	}
}

func newRelayTCPPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	defer ln.Close()

	acceptCh := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		acceptCh <- conn
	}()

	clientConn, err := net.Dial("tcp4", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial tcp: %v", err)
	}
	select {
	case serverConn := <-acceptCh:
		return serverConn, clientConn
	case err := <-acceptErr:
		clientConn.Close()
		t.Fatalf("accept tcp: %v", err)
		return nil, nil
	}
}

func newRelayUnixPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "relay.sock")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	defer ln.Close()

	acceptCh := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		acceptCh <- conn
	}()

	clientConn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("dial unix: %v", err)
	}
	select {
	case serverConn := <-acceptCh:
		return serverConn, clientConn
	case err := <-acceptErr:
		clientConn.Close()
		t.Fatalf("accept unix: %v", err)
		return nil, nil
	}
}
