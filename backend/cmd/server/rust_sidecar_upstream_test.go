package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestStartRustSidecarUpstreamServerServesHTTPOverUnixSocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "rust-upstream.sock")
	cfg := &config.Config{}
	cfg.Rust.Sidecar.Enabled = true
	cfg.Rust.Sidecar.UpstreamSocketPath = socketPath

	base := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Upstream", "go")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ok"))
		}),
		ReadHeaderTimeout: time.Second,
		IdleTimeout:       time.Second,
	}

	stop, err := startRustSidecarUpstreamServer(cfg, base)
	if err != nil {
		t.Fatalf("startRustSidecarUpstreamServer: %v", err)
	}
	defer stop()

	client := &http.Client{
		Timeout: time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		},
	}

	req, err := http.NewRequest(http.MethodGet, "http://rust-sidecar-upstream/health", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if string(body) != "ok" {
		t.Fatalf("unexpected body: %q", string(body))
	}
	if resp.Header.Get("X-Upstream") != "go" {
		t.Fatalf("unexpected upstream header: %q", resp.Header.Get("X-Upstream"))
	}
}
