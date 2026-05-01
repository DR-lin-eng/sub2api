package sidecar

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestRelayCopyOneWayBufferedFallbackForInMemoryConns(t *testing.T) {
	srcRelay, srcPeer := net.Pipe()
	dstRelay, dstPeer := net.Pipe()
	defer srcRelay.Close()
	defer srcPeer.Close()
	defer dstRelay.Close()
	defer dstPeer.Close()
	setRelayTestDeadline(t, srcRelay, srcPeer, dstRelay, dstPeer)

	payload := bytes.Repeat([]byte("buffered-fallback:"), 4096)
	copyErr := make(chan error, 1)
	readErr := make(chan error, 1)

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
		got := make([]byte, len(payload))
		if _, err := io.ReadFull(dstPeer, got); err != nil {
			readErr <- err
			return
		}
		if !bytes.Equal(got, payload) {
			readErr <- fmt.Errorf("relay payload mismatch")
			return
		}
		readErr <- nil
	}()

	if _, err := srcPeer.Write(payload); err != nil {
		t.Fatalf("write source payload: %v", err)
	}
	if err := srcPeer.Close(); err != nil {
		t.Fatalf("close source peer: %v", err)
	}

	if err := <-readErr; err != nil {
		t.Fatalf("read relayed payload: %v", err)
	}
	if err := <-copyErr; err != nil {
		t.Fatalf("copy relay: %v", err)
	}
}

func setRelayTestDeadline(t *testing.T, conns ...net.Conn) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for _, conn := range conns {
		if err := conn.SetDeadline(deadline); err != nil {
			t.Fatalf("set deadline: %v", err)
		}
	}
}
