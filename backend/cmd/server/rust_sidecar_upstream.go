package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	serverruntime "github.com/Wei-Shaw/sub2api/internal/server"
)

func startRustSidecarUpstreamServer(cfg *config.Config, base *http.Server) (func(), error) {
	if cfg == nil || base == nil || base.Handler == nil {
		return func() {}, nil
	}
	if !cfg.Rust.Sidecar.Enabled {
		return func() {}, nil
	}

	socketPath := filepath.Clean(cfg.Rust.Sidecar.UpstreamSocketPath)
	if socketPath == "" || socketPath == "." || socketPath == "/" {
		return nil, errors.New("invalid rust sidecar upstream socket path")
	}
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}

	server := &http.Server{
		Handler:           serverruntime.BuildExecutablePreferredHandler(base),
		ReadHeaderTimeout: base.ReadHeaderTimeout,
		IdleTimeout:       base.IdleTimeout,
	}

	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			log.Printf("rust sidecar upstream server exited with error: %v", err)
		}
	}()

	stop := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		_ = ln.Close()
		_ = os.Remove(socketPath)
	}
	return stop, nil
}
