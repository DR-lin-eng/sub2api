package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	rustsidecar "github.com/Wei-Shaw/sub2api/internal/rustbridge/sidecar"
)

func startRustSidecarProcess(cfg *config.Config) (func(), error) {
	if cfg == nil || !cfg.Rust.Sidecar.Enabled || !cfg.Rust.Sidecar.AutoStart {
		return func() {}, nil
	}

	binaryPath := strings.TrimSpace(cfg.Rust.Sidecar.BinaryPath)
	if binaryPath == "" {
		return nil, errors.New("rust sidecar binary path is empty")
	}
	socketPath := filepath.Clean(cfg.Rust.Sidecar.SocketPath)
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	cmd := exec.Command(binaryPath, cfg.Rust.Sidecar.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), rustSidecarProcessEnv(cfg)...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	client, err := rustsidecar.NewClient(cfg.Rust.Sidecar)
	if err != nil {
		_ = stopRustSidecarProcess(cmd, waitCh)
		return nil, err
	}

	deadline := time.Now().Add(time.Duration(cfg.Rust.Sidecar.HealthcheckTimeoutSeconds) * time.Second)
	if deadline.Before(time.Now()) {
		deadline = time.Now().Add(3 * time.Second)
	}
	for {
		select {
		case err := <-waitCh:
			_ = stopRustSidecarProcess(cmd, nil)
			return nil, fmt.Errorf("rust sidecar exited before healthy: %w", err)
		default:
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := client.Health(ctx)
		cancel()
		if err == nil {
			return func() {
				_ = stopRustSidecarProcess(cmd, waitCh)
			}, nil
		}
		if time.Now().After(deadline) {
			_ = stopRustSidecarProcess(cmd, waitCh)
			return nil, fmt.Errorf("rust sidecar healthcheck timeout: %w", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func rustSidecarProcessEnv(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	return []string{
		"SUB2API_RUST_SIDECAR_SOCKET=" + filepath.Clean(cfg.Rust.Sidecar.SocketPath),
		"SUB2API_RUST_UPSTREAM_SOCKET=" + filepath.Clean(cfg.Rust.Sidecar.UpstreamSocketPath),
		"SUB2API_RUST_REQUEST_TIMEOUT_MS=" + strconv.Itoa(cfg.Rust.Sidecar.RequestTimeoutSeconds*1000),
		"SUB2API_RUST_UPGRADE_IDLE_TIMEOUT_MS=" + strconv.Itoa(cfg.Rust.Sidecar.UpgradeIdleTimeoutSeconds*1000),
		"SUB2API_RUST_WS_MAX_MESSAGE_BYTES=" + strconv.Itoa(cfg.Rust.Sidecar.WebSocketMaxMessageBytes),
	}
}

func stopRustSidecarProcess(cmd *exec.Cmd, waitCh <-chan error) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Signal(os.Interrupt)

	if waitCh != nil {
		select {
		case <-waitCh:
			return nil
		case <-time.After(3 * time.Second):
		}
	}
	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	if waitCh != nil {
		select {
		case <-waitCh:
		case <-time.After(2 * time.Second):
		}
	}
	return nil
}
