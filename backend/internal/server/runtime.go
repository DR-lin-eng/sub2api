package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/runtimeobs"
	"golang.org/x/net/http2"
)

const (
	classifiedListenerQueueSize = 128
	maxClassifyPeekBytes        = 1024
)

var classifyPeekBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, maxClassifyPeekBytes)
	},
}

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
		runtimeobs.RecordGnetClassifyError()
		_ = conn.Close()
		return
	}
	switch target {
	case protocolTargetSidecar:
		runtimeobs.RecordGnetSidecarClassified()
		if !m.sidecar.enqueue(wrapped) {
			runtimeobs.RecordGnetSidecarEnqueueDrop()
			_ = wrapped.Close()
		}
	case protocolTargetH2C:
		runtimeobs.RecordGnetH2CClassified()
		if !m.h2c.enqueue(wrapped) {
			runtimeobs.RecordGnetH2CEnqueueDrop()
			_ = wrapped.Close()
		}
	default:
		runtimeobs.RecordGnetHTTP1Classified()
		if !m.http1.enqueue(wrapped) {
			runtimeobs.RecordGnetHTTP1EnqueueDrop()
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

	buffer := classifyPeekBufferPool.Get().([]byte)
	n := 0
	for n < len(buffer) {
		prevN := n
		readN, err := conn.Read(buffer[n:])
		n += readN
		if err != nil {
			if errors.Is(err, io.EOF) && n > 0 {
				break
			}
			putClassifyPeekBuffer(buffer)
			return protocolTargetHTTP1, nil, err
		}
		if n >= len(http2.ClientPreface) && bytes.HasPrefix(buffer[:n], []byte(http2.ClientPreface)) {
			break
		}
		if hasHTTPHeaderTerminatorIncremental(buffer[:n], prevN) {
			break
		}
	}
	if n == 0 {
		putClassifyPeekBuffer(buffer)
		return protocolTargetHTTP1, nil, io.ErrUnexpectedEOF
	}
	peeked := buffer[:n]
	wrapped := &prefixedConn{Conn: conn, prefix: peeked}
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
	if containsASCIIFold(peeked, "upgrade: h2c") || containsASCIIFold(peeked, "http2-settings:") {
		if mux != nil && mux.sidecarEnabled && mux.routeH2CToSidecar {
			return protocolTargetSidecar
		}
		return protocolTargetH2C
	}
	if mux != nil && mux.sidecarEnabled && mux.routeResponsesWSToSidecar && isOpenAIResponsesWebSocketPreface(peeked) {
		return protocolTargetSidecar
	}
	if containsASCIIFold(peeked, "upgrade: h2c") || containsASCIIFold(peeked, "http2-settings:") {
		return protocolTargetH2C
	}
	return protocolTargetHTTP1
}

func isOpenAIResponsesWebSocketPreface(peeked []byte) bool {
	if !containsASCIIFold(peeked, "upgrade: websocket") {
		return false
	}
	if hasPrefixASCIIFold(peeked, "get /v1/responses ") || hasPrefixASCIIFold(peeked, "get /responses ") {
		return true
	}
	return false
}

func hasHTTPHeaderTerminatorIncremental(data []byte, prevLen int) bool {
	if len(data) < 4 {
		return false
	}
	start := prevLen - 3
	if start < 0 {
		start = 0
	}
	limit := len(data) - 3
	for i := start; i < limit; i++ {
		if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			return true
		}
	}
	return false
}

func containsASCIIFold(data []byte, token string) bool {
	if len(token) == 0 {
		return true
	}
	if len(data) < len(token) {
		return false
	}
	limit := len(data) - len(token)
	for start := 0; start <= limit; start++ {
		if equalFoldASCII(data[start:start+len(token)], token) {
			return true
		}
	}
	return false
}

func hasPrefixASCIIFold(data []byte, prefix string) bool {
	if len(data) < len(prefix) {
		return false
	}
	return equalFoldASCII(data[:len(prefix)], prefix)
}

func equalFoldASCII(data []byte, token string) bool {
	if len(data) != len(token) {
		return false
	}
	for i := range data {
		if asciiLower(data[i]) != asciiLower(token[i]) {
			return false
		}
	}
	return true
}

func asciiLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

type prefixedConn struct {
	net.Conn
	prefix []byte
}

func (c *prefixedConn) SyscallConn() (syscall.RawConn, error) {
	if c == nil || c.Conn == nil {
		return nil, net.ErrClosed
	}
	sc, ok := c.Conn.(syscall.Conn)
	if !ok {
		return nil, errors.New("wrapped connection does not implement syscall.Conn")
	}
	return sc.SyscallConn()
}

func (c *prefixedConn) Read(p []byte) (int, error) {
	if c == nil {
		return 0, net.ErrClosed
	}
	if len(c.prefix) > 0 {
		n := copy(p, c.prefix)
		c.prefix = c.prefix[n:]
		if len(c.prefix) == 0 {
			c.releasePrefix()
		}
		if n > 0 {
			return n, nil
		}
	}
	return c.Conn.Read(p)
}

func (c *prefixedConn) Close() error {
	if c == nil {
		return net.ErrClosed
	}
	c.releasePrefix()
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}

func (c *prefixedConn) takePrefix() []byte {
	if c == nil || len(c.prefix) == 0 {
		return nil
	}
	prefix := c.prefix
	c.prefix = nil
	return prefix
}

func (c *prefixedConn) releasePrefix() {
	if c == nil || cap(c.prefix) == 0 {
		c.prefix = nil
		return
	}
	putClassifyPeekBuffer(c.prefix)
	c.prefix = nil
}

func putClassifyPeekBuffer(buf []byte) {
	if cap(buf) < maxClassifyPeekBytes {
		return
	}
	classifyPeekBufferPool.Put(buf[:maxClassifyPeekBytes])
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
		ch:   make(chan net.Conn, classifiedListenerQueueSize),
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
