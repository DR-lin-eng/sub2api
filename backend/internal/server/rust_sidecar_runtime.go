package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
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
	closeOnce sync.Once
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
	upstream, err := (&net.Dialer{Timeout: timeout}).Dial("unix", r.client.SocketPath())
	if err != nil {
		_ = downstream.Close()
		return
	}

	done := make(chan struct{}, 2)

	go func() {
		_, _ = relaySidecarCopy(upstream, downstream)
		done <- struct{}{}
	}()
	go func() {
		_, _ = relaySidecarCopy(downstream, upstream)
		done <- struct{}{}
	}()
	go func() {
		<-done
		<-done
		_ = downstream.Close()
		_ = upstream.Close()
	}()
}

func relaySidecarCopy(dst net.Conn, src net.Conn) (int64, error) {
	bufPtr := rustsidecarRelayBufferPool.Get().(*[]byte)
	defer rustsidecarRelayBufferPool.Put(bufPtr)
	if dst == nil || src == nil {
		return 0, nil
	}
	n, err := io.CopyBuffer(dst, src, *bufPtr)
	relaySidecarCloseWrite(dst)
	relaySidecarCloseRead(src)
	return n, err
}

var rustsidecarRelayBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 64*1024)
		return &buf
	},
}

type rustsidecarCloseWriter interface {
	CloseWrite() error
}

type rustsidecarCloseReader interface {
	CloseRead() error
}

func relaySidecarCloseWrite(conn net.Conn) {
	if conn == nil {
		return
	}
	if cw, ok := conn.(rustsidecarCloseWriter); ok {
		_ = cw.CloseWrite()
	}
}

func relaySidecarCloseRead(conn net.Conn) {
	if conn == nil {
		return
	}
	if cr, ok := conn.(rustsidecarCloseReader); ok {
		_ = cr.CloseRead()
	}
}
