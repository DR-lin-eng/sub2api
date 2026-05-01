package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	rustsidecar "github.com/Wei-Shaw/sub2api/internal/rustbridge/sidecar"
)

type rustSidecarIngressRuntime struct {
	name      string
	addr      string
	manifest  RouteManifest
	cfg       config.RustSidecarConfig
	client    *rustsidecar.Client
}

func newRustSidecarIngressRuntime(cfg *config.Config, base *http.Server, manifest RouteManifest) *rustSidecarIngressRuntime {
	if cfg == nil {
		return nil
	}
	client, err := rustsidecar.NewClient(cfg.Rust.Sidecar)
	if err != nil {
		log.Printf("rust sidecar runtime disabled: %v", err)
		return nil
	}
	addr := ""
	if base != nil {
		addr = base.Addr
	}
	return &rustSidecarIngressRuntime{
		name:     "rust-sidecar-h2c",
		addr:     addr,
		manifest: cloneRouteManifest(manifest),
		cfg:      cfg.Rust.Sidecar,
		client:   client,
	}
}

func (r *rustSidecarIngressRuntime) Name() string {
	if r == nil {
		return ""
	}
	return r.name
}

func (r *rustSidecarIngressRuntime) Addr() string {
	if r == nil {
		return ""
	}
	return r.addr
}

func (r *rustSidecarIngressRuntime) RouteManifest() RouteManifest {
	if r == nil {
		return nil
	}
	return cloneRouteManifest(r.manifest)
}

func (r *rustSidecarIngressRuntime) Serve(listener net.Listener) error {
	if r == nil {
		return fmt.Errorf("rust sidecar runtime is nil")
	}
	if listener == nil {
		return fmt.Errorf("listener is nil")
	}
	if err := r.healthcheck(); err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return normalizeServeError(err)
		}
		go r.proxyConn(conn)
	}
}

func (r *rustSidecarIngressRuntime) Shutdown(ctx context.Context) error {
	_ = ctx
	return nil
}

func (r *rustSidecarIngressRuntime) Close() error {
	return nil
}

func (r *rustSidecarIngressRuntime) healthcheck() error {
	if r == nil || r.client == nil {
		return fmt.Errorf("rust sidecar client is not configured")
	}
	timeout := time.Duration(r.cfg.HealthcheckTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := r.client.Health(ctx)
	if err != nil && r.cfg.FailClosed {
		return fmt.Errorf("rust sidecar healthcheck failed: %w", err)
	}
	if err != nil {
		log.Printf("rust sidecar healthcheck warning: %v", err)
	}
	return nil
}

func (r *rustSidecarIngressRuntime) proxyConn(downstream net.Conn) {
	if r == nil || downstream == nil {
		return
	}
	timeout := time.Duration(r.cfg.RequestTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	connectTimeout := timeout
	if connectTimeout > 5*time.Second {
		connectTimeout = 5 * time.Second
	}
	upstream, err := rustsidecar.DialSidecar(r.client.SocketPath(), connectTimeout)
	if err != nil {
		_ = downstream.Close()
		return
	}

	done := make(chan struct{}, 2)

	go func() {
		_, _ = rustsidecar.RelayCopyOneWay(upstream, downstream)
		done <- struct{}{}
	}()
	go func() {
		_, _ = rustsidecar.RelayCopyOneWay(downstream, upstream)
		done <- struct{}{}
	}()
	go func() {
		<-done
		<-done
		_ = downstream.Close()
		_ = upstream.Close()
	}()
}
