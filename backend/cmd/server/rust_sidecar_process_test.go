package main

import (
	"os/exec"
	"runtime"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestStopRustSidecarProcessTerminatesChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell lifecycle test is unix-oriented")
	}

	cmd := exec.Command("/bin/sh", "-c", "trap 'exit 0' INT TERM; while true; do sleep 1; done")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start shell: %v", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	if err := stopRustSidecarProcess(cmd, waitCh); err != nil {
		t.Fatalf("stopRustSidecarProcess: %v", err)
	}
}

func TestRustSidecarProcessEnvIncludesRelayLimits(t *testing.T) {
	cfg := &config.Config{}
	cfg.Rust.Sidecar.SocketPath = "/tmp/sub2api-sidecar.sock"
	cfg.Rust.Sidecar.UpstreamSocketPath = "/tmp/sub2api-upstream.sock"
	cfg.Rust.Sidecar.RequestTimeoutSeconds = 7
	cfg.Rust.Sidecar.UpgradeIdleTimeoutSeconds = 13
	cfg.Rust.Sidecar.WebSocketMaxMessageBytes = 65536

	env := rustSidecarProcessEnv(cfg)
	envMap := make(map[string]string, len(env))
	for _, item := range env {
		for idx := 0; idx < len(item); idx++ {
			if item[idx] != '=' {
				continue
			}
			envMap[item[:idx]] = item[idx+1:]
			break
		}
	}

	if envMap["SUB2API_RUST_SIDECAR_SOCKET"] != "/tmp/sub2api-sidecar.sock" {
		t.Fatalf("unexpected sidecar socket env: %+v", envMap)
	}
	if envMap["SUB2API_RUST_UPSTREAM_SOCKET"] != "/tmp/sub2api-upstream.sock" {
		t.Fatalf("unexpected upstream socket env: %+v", envMap)
	}
	if envMap["SUB2API_RUST_REQUEST_TIMEOUT_MS"] != "7000" {
		t.Fatalf("unexpected request timeout env: %+v", envMap)
	}
	if envMap["SUB2API_RUST_UPGRADE_IDLE_TIMEOUT_MS"] != "13000" {
		t.Fatalf("unexpected upgrade idle timeout env: %+v", envMap)
	}
	if envMap["SUB2API_RUST_WS_MAX_MESSAGE_BYTES"] != "65536" {
		t.Fatalf("unexpected websocket max message env: %+v", envMap)
	}
}
