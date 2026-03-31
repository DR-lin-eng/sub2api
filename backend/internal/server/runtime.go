package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"golang.org/x/net/http2"
)

type IngressRuntime interface {
	Name() string
	Addr() string
	RouteManifest() RouteManifest
	Serve(listener net.Listener) error
	Shutdown(ctx context.Context) error
	Close() error
}

type netHTTPRuntime struct {
	name     string
	server   *http.Server
	manifest RouteManifest
}

func NewNetHTTPRuntimeFromServer(base *http.Server) IngressRuntime {
	return &netHTTPRuntime{
		name:     config.ServerRuntimeModeNetHTTP,
		server:   cloneHTTPServer(base),
		manifest: routeManifestForHTTPServer(base),
	}
}

func (r *netHTTPRuntime) Name() string {
	return r.name
}

func (r *netHTTPRuntime) Addr() string {
	if r == nil || r.server == nil {
		return ""
	}
	return r.server.Addr
}

func (r *netHTTPRuntime) RouteManifest() RouteManifest {
	return cloneRouteManifest(r.manifest)
}

func (r *netHTTPRuntime) Serve(listener net.Listener) error {
	if r == nil || r.server == nil {
		return errors.New("nethttp runtime server is nil")
	}
	return normalizeServeError(r.server.Serve(listener))
}

func (r *netHTTPRuntime) Shutdown(ctx context.Context) error {
	if r == nil || r.server == nil {
		return nil
	}
	return r.server.Shutdown(ctx)
}

func (r *netHTTPRuntime) Close() error {
	if r == nil || r.server == nil {
		return nil
	}
	return normalizeServeError(r.server.Close())
}

func (r *netHTTPRuntime) HTTPServer() *http.Server {
	if r == nil {
		return nil
	}
	return r.server
}

type hybridRuntime struct {
	name     string
	addr     string
	manifest RouteManifest

	sidecarRuntime            IngressRuntime
	routeH2CToSidecar         bool
	routeResponsesWSToSidecar bool
	h2cRuntime                IngressRuntime
	http1Runtime              IngressRuntime
	readTimeout               time.Duration

	mu  sync.Mutex
	mux *protocolMux
}

func NewHybridRuntimeFromServer(base *http.Server) IngressRuntime {
	return newSplitRuntime(nil, base, config.ServerRuntimeModeHybrid)
}

func newSplitRuntime(cfg *config.Config, base *http.Server, mode string) IngressRuntime {
	if base == nil {
		return &hybridRuntime{name: mode}
	}

	manifest := routeManifestForHTTPServer(base)
	h2cServer := cloneHTTPServer(base)
	h2cRuntime := IngressRuntime(&netHTTPRuntime{name: config.ServerRuntimeModeNetHTTP, server: h2cServer, manifest: manifest})
	var sidecarRuntime IngressRuntime
	routeH2CToSidecar := false
	routeResponsesWSToSidecar := false
	if cfg != nil && cfg.Rust.Sidecar.Enabled && cfg.Rust.Sidecar.H2CDelegateEnabled {
		if rt := newRustSidecarIngressRuntime(cfg, base, manifest); rt != nil {
			h2cRuntime = rt
			sidecarRuntime = rt
			routeH2CToSidecar = true
		}
	}
	if cfg != nil && cfg.Rust.Sidecar.Enabled && cfg.Rust.Sidecar.ResponsesWSEnabled {
		if sidecarRuntime == nil {
			if rt := newRustSidecarIngressRuntime(cfg, base, manifest); rt != nil {
				sidecarRuntime = rt
			}
		}
		if sidecarRuntime != nil {
			routeResponsesWSToSidecar = true
		}
	}

	return &hybridRuntime{
		name:                      mode,
		addr:                      base.Addr,
		manifest:                  manifest,
		sidecarRuntime:            sidecarRuntime,
		routeH2CToSidecar:         routeH2CToSidecar,
		routeResponsesWSToSidecar: routeResponsesWSToSidecar,
		h2cRuntime:                h2cRuntime,
		http1Runtime:              newNativeGnetHTTPRuntime(cfg, base),
		readTimeout:               base.ReadHeaderTimeout,
	}
}

func (r *hybridRuntime) Name() string {
	return r.name
}

func (r *hybridRuntime) Addr() string {
	return r.addr
}

func (r *hybridRuntime) RouteManifest() RouteManifest {
	return cloneRouteManifest(r.manifest)
}

func (r *hybridRuntime) Serve(listener net.Listener) error {
	if listener == nil {
		return errors.New("listener is nil")
	}
	if r.h2cRuntime == nil || r.http1Runtime == nil {
		return errors.New("hybrid runtime delegates are nil")
	}

	mux := newProtocolMux(listener, r.readTimeout, r.sidecarRuntime != nil, r.routeH2CToSidecar, r.routeResponsesWSToSidecar)
	r.mu.Lock()
	r.mux = mux
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.mux = nil
		r.mu.Unlock()
	}()

	errWorkers := 3
	if r.sidecarRuntime != nil {
		errWorkers++
	}
	errCh := make(chan error, errWorkers)
	go func() { errCh <- normalizeServeError(mux.Serve()) }()
	go func() { errCh <- normalizeServeError(r.h2cRuntime.Serve(mux.H2CListener())) }()
	go func() { errCh <- normalizeServeError(r.http1Runtime.Serve(mux.HTTP1Listener())) }()
	if r.sidecarRuntime != nil {
		go func() { errCh <- normalizeServeError(r.sidecarRuntime.Serve(mux.SidecarListener())) }()
	}

	var firstErr error
	for i := 0; i < errWorkers; i++ {
		err := <-errCh
		if err != nil && !isExpectedListenerClose(err) && firstErr == nil {
			firstErr = err
			_ = mux.Close()
			if r.sidecarRuntime != nil {
				_ = r.sidecarRuntime.Close()
			}
			_ = r.h2cRuntime.Close()
			_ = r.http1Runtime.Close()
		}
	}
	return firstErr
}

func (r *hybridRuntime) Shutdown(ctx context.Context) error {
	var errs []error

	r.mu.Lock()
	if r.mux != nil {
		if err := r.mux.Close(); err != nil && !isExpectedListenerClose(err) {
			errs = append(errs, err)
		}
	}
	r.mu.Unlock()

	if r.http1Runtime != nil {
		if err := r.http1Runtime.Shutdown(ctx); err != nil && !isExpectedListenerClose(err) {
			errs = append(errs, err)
		}
	}
	if r.sidecarRuntime != nil && r.sidecarRuntime != r.h2cRuntime {
		if err := r.sidecarRuntime.Shutdown(ctx); err != nil && !isExpectedListenerClose(err) {
			errs = append(errs, err)
		}
	}
	if r.h2cRuntime != nil {
		if err := r.h2cRuntime.Shutdown(ctx); err != nil && !isExpectedListenerClose(err) {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (r *hybridRuntime) Close() error {
	var errs []error

	r.mu.Lock()
	if r.mux != nil {
		if err := r.mux.Close(); err != nil && !isExpectedListenerClose(err) {
			errs = append(errs, err)
		}
	}
	r.mu.Unlock()

	if r.http1Runtime != nil {
		if err := r.http1Runtime.Close(); err != nil && !isExpectedListenerClose(err) {
			errs = append(errs, err)
		}
	}
	if r.sidecarRuntime != nil && r.sidecarRuntime != r.h2cRuntime {
		if err := r.sidecarRuntime.Close(); err != nil && !isExpectedListenerClose(err) {
			errs = append(errs, err)
		}
	}
	if r.h2cRuntime != nil {
		if err := r.h2cRuntime.Close(); err != nil && !isExpectedListenerClose(err) {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func NewGnetRuntimeFromServer(base *http.Server) IngressRuntime {
	return newSplitRuntime(nil, base, config.ServerRuntimeModeGnet)
}

func ResolveIngressRuntime(cfg *config.Config, base *http.Server) IngressRuntime {
	mode := config.ServerRuntimeModeNetHTTP
	if cfg != nil {
		mode = config.NormalizeServerRuntimeMode(cfg.Server.RuntimeMode)
	}
	switch mode {
	case config.ServerRuntimeModeHybrid:
		return newSplitRuntime(cfg, base, config.ServerRuntimeModeHybrid)
	case config.ServerRuntimeModeGnet:
		return newSplitRuntime(cfg, base, config.ServerRuntimeModeGnet)
	default:
		return NewNetHTTPRuntimeFromServer(base)
	}
}

func normalizeServeError(err error) error {
	if isExpectedListenerClose(err) {
		return nil
	}
	return err
}

func isExpectedListenerClose(err error) bool {
	return err == nil || errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed)
}

func serverAddr(base *http.Server) string {
	if base == nil {
		return ""
	}
	return base.Addr
}

func cloneHTTPServer(base *http.Server) *http.Server {
	if base == nil {
		return nil
	}
	cloned := *base
	return &cloned
}

type protocolTarget int

const (
	protocolTargetHTTP1 protocolTarget = iota + 1
	protocolTargetH2C
	protocolTargetSidecar
)

type protocolMux struct {
	base                      net.Listener
	readTimeout               time.Duration
	sidecarEnabled            bool
	routeH2CToSidecar         bool
	routeResponsesWSToSidecar bool
	http1                     *classifiedListener
	h2c                       *classifiedListener
	sidecar                   *classifiedListener

	closeOnce sync.Once
}

func newProtocolMux(base net.Listener, readTimeout time.Duration, sidecarEnabled bool, routeH2CToSidecar bool, routeResponsesWSToSidecar bool) *protocolMux {
	return &protocolMux{
		base:                      base,
		readTimeout:               readTimeout,
		sidecarEnabled:            sidecarEnabled,
		routeH2CToSidecar:         routeH2CToSidecar,
		routeResponsesWSToSidecar: routeResponsesWSToSidecar,
		http1:                     newClassifiedListener(base.Addr()),
		h2c:                       newClassifiedListener(base.Addr()),
		sidecar:                   newClassifiedListener(base.Addr()),
	}
}

func (m *protocolMux) HTTP1Listener() net.Listener {
	return m.http1
}

func (m *protocolMux) H2CListener() net.Listener {
	return m.h2c
}

func (m *protocolMux) SidecarListener() net.Listener {
	return m.sidecar
}

func (m *protocolMux) Close() error {
	var err error
	m.closeOnce.Do(func() {
		err = m.base.Close()
		_ = m.http1.Close()
		_ = m.h2c.Close()
		_ = m.sidecar.Close()
	})
	return err
}

func (m *protocolMux) Serve() error {
	for {
		conn, err := m.base.Accept()
		if err != nil {
			return err
		}
		go m.dispatch(conn)
	}
}

func (m *protocolMux) dispatch(conn net.Conn) {
	target, wrapped, err := classifyConn(conn, m.readTimeout, m)
	if err != nil {
		_ = conn.Close()
		return
	}
	switch target {
	case protocolTargetSidecar:
		if !m.sidecar.enqueue(wrapped) {
			_ = wrapped.Close()
		}
	case protocolTargetH2C:
		if !m.h2c.enqueue(wrapped) {
			_ = wrapped.Close()
		}
	default:
		if !m.http1.enqueue(wrapped) {
			_ = wrapped.Close()
		}
	}
}

func classifyConn(conn net.Conn, timeout time.Duration, mux *protocolMux) (protocolTarget, net.Conn, error) {
	if conn == nil {
		return protocolTargetHTTP1, nil, errors.New("conn is nil")
	}
	if timeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		defer func() { _ = conn.SetReadDeadline(time.Time{}) }()
	}

	const maxPeekBytes = 1024
	buffer := make([]byte, maxPeekBytes)
	n := 0
	for n < len(buffer) {
		limit := n + 256
		if limit > len(buffer) {
			limit = len(buffer)
		}
		readN, err := conn.Read(buffer[n:limit])
		n += readN
		if err != nil {
			if errors.Is(err, io.EOF) && n > 0 {
				break
			}
			return protocolTargetHTTP1, nil, err
		}
		if n >= len(http2.ClientPreface) && bytes.HasPrefix(buffer[:n], []byte(http2.ClientPreface)) {
			break
		}
		if bytes.Contains(buffer[:n], []byte("\r\n\r\n")) {
			break
		}
	}
	if n == 0 {
		return protocolTargetHTTP1, nil, io.ErrUnexpectedEOF
	}
	peeked := buffer[:n]
	wrapped := &prefixedConn{Conn: conn, prefix: bytes.NewReader(peeked)}
	return classifyPrefaceWithOptions(peeked, mux), wrapped, nil
}

func classifyPreface(peeked []byte) protocolTarget {
	return classifyPrefaceWithOptions(peeked, nil)
}

func classifyPrefaceWithOptions(peeked []byte, mux *protocolMux) protocolTarget {
	if bytes.HasPrefix(peeked, []byte(http2.ClientPreface)) {
		if mux != nil && mux.sidecarEnabled && mux.routeH2CToSidecar {
			return protocolTargetSidecar
		}
		return protocolTargetH2C
	}
	lower := strings.ToLower(string(peeked))
	if strings.Contains(lower, "upgrade: h2c") || strings.Contains(lower, "http2-settings:") {
		if mux != nil && mux.sidecarEnabled && mux.routeH2CToSidecar {
			return protocolTargetSidecar
		}
		return protocolTargetH2C
	}
	if mux != nil && mux.sidecarEnabled && mux.routeResponsesWSToSidecar && isOpenAIResponsesWebSocketPreface(lower) {
		return protocolTargetSidecar
	}
	if strings.Contains(lower, "upgrade: h2c") || strings.Contains(lower, "http2-settings:") {
		return protocolTargetH2C
	}
	return protocolTargetHTTP1
}

func isOpenAIResponsesWebSocketPreface(lower string) bool {
	if !strings.Contains(lower, "upgrade: websocket") {
		return false
	}
	if strings.HasPrefix(lower, "get /v1/responses ") || strings.HasPrefix(lower, "get /responses ") {
		return true
	}
	return false
}

type prefixedConn struct {
	net.Conn
	prefix io.Reader
}

func (c *prefixedConn) Read(p []byte) (int, error) {
	if c == nil {
		return 0, net.ErrClosed
	}
	if c.prefix != nil {
		n, err := c.prefix.Read(p)
		if err == nil {
			return n, nil
		}
		if errors.Is(err, io.EOF) {
			c.prefix = nil
			if n > 0 {
				return n, nil
			}
		} else {
			return n, err
		}
	}
	return c.Conn.Read(p)
}

type classifiedListener struct {
	addr net.Addr
	ch   chan net.Conn
	done chan struct{}

	closeOnce sync.Once
}

func newClassifiedListener(addr net.Addr) *classifiedListener {
	return &classifiedListener{
		addr: addr,
		ch:   make(chan net.Conn),
		done: make(chan struct{}),
	}
}

func (l *classifiedListener) Accept() (net.Conn, error) {
	select {
	case <-l.done:
		return nil, net.ErrClosed
	case conn := <-l.ch:
		if conn == nil {
			return nil, net.ErrClosed
		}
		return conn, nil
	}
}

func (l *classifiedListener) Close() error {
	l.closeOnce.Do(func() {
		close(l.done)
	})
	return nil
}

func (l *classifiedListener) Addr() net.Addr {
	return l.addr
}

func (l *classifiedListener) enqueue(conn net.Conn) bool {
	select {
	case <-l.done:
		return false
	case l.ch <- conn:
		return true
	}
}
