package sidecar

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestClientHealthOverUnixSocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "rust-sidecar.sock")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:                         "ok",
			Service:                        "test-sidecar",
			Version:                        "v0",
			ActiveConnections:              2,
			TotalConnections:               3,
			ActiveUpgrades:                 1,
			TotalUpgrades:                  4,
			TotalRequests:                  5,
			TotalRequestErrors:             1,
			UpstreamUnavailableTotal:       2,
			UpstreamHandshakeFailedTotal:   3,
			UpstreamRequestFailedTotal:     4,
			UpgradeErrorsTotal:             5,
			RelayBytesDownstreamToUpstream: 6,
			RelayBytesUpstreamToDownstream: 7,
			RelayFramesDownstreamToUpstream: 8,
			RelayFramesUpstreamToDownstream: 9,
			RelayCloseFramesTotal:           10,
			RelayPingFramesTotal:            11,
			RelayPongFramesTotal:            12,
		})
	})

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	defer ln.Close()

	server := &http.Server{Handler: mux}
	defer server.Close()

	go func() {
		_ = server.Serve(ln)
	}()

	client, err := NewClient(config.RustSidecarConfig{
		Enabled:               true,
		SocketPath:            socketPath,
		RequestTimeoutSeconds: 1,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if health.Status != "ok" {
		t.Fatalf("unexpected health status: %q", health.Status)
	}
	if health.Service != "test-sidecar" {
		t.Fatalf("unexpected service name: %q", health.Service)
	}
	if health.ActiveConnections != 2 || health.TotalConnections != 3 || health.ActiveUpgrades != 1 || health.TotalUpgrades != 4 {
		t.Fatalf("unexpected runtime counters: %+v", health)
	}
	if health.TotalRequests != 5 || health.TotalRequestErrors != 1 || health.UpstreamUnavailableTotal != 2 || health.UpstreamHandshakeFailedTotal != 3 || health.UpstreamRequestFailedTotal != 4 || health.UpgradeErrorsTotal != 5 || health.RelayBytesDownstreamToUpstream != 6 || health.RelayBytesUpstreamToDownstream != 7 || health.RelayFramesDownstreamToUpstream != 8 || health.RelayFramesUpstreamToDownstream != 9 || health.RelayCloseFramesTotal != 10 || health.RelayPingFramesTotal != 11 || health.RelayPongFramesTotal != 12 {
		t.Fatalf("unexpected runtime counters: %+v", health)
	}
}
