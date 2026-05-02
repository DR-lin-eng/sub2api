package server

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/runtimeobs"
	"github.com/panjf2000/gnet/v2"
)

type nativeGnetHTTPRuntime struct {
	gnet.BuiltinEventEngine

	addr        string
	handler     http.Handler
	manifest    RouteManifest
	execRuntime *executableRuntimeConfig
	readTimeout time.Duration

	mu       sync.Mutex
	listener net.Listener
	client   *gnet.Client
	closed   bool
}

func newNativeGnetHTTPRuntime(cfg *config.Config, base *http.Server) IngressRuntime {
	if base == nil {
		return &nativeGnetHTTPRuntime{}
	}
	execRuntime := executableRuntimeForHTTPServer(base)
	if execRuntime == nil {
		execRuntime = buildExecutableRuntimeConfig(cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	}
	return &nativeGnetHTTPRuntime{
		addr:        serverAddr(base),
		handler:     base.Handler,
		manifest:    routeManifestForHTTPServer(base),
		execRuntime: execRuntime,
		readTimeout: base.ReadHeaderTimeout,
	}
}

func (r *nativeGnetHTTPRuntime) Name() string {
	return config.ServerRuntimeModeGnet
}

func (r *nativeGnetHTTPRuntime) Addr() string {
	return r.addr
}

func (r *nativeGnetHTTPRuntime) RouteManifest() RouteManifest {
	return cloneRouteManifest(r.manifest)
}

func (r *nativeGnetHTTPRuntime) Serve(listener net.Listener) error {
	if listener == nil {
		return errors.New("listener is nil")
	}
	if r.handler == nil {
		return errors.New("native gnet runtime handler is nil")
	}

	cli, err := gnet.NewClient(r)
	if err != nil {
		return err
	}
	if err := cli.Start(); err != nil {
		return err
	}

	r.mu.Lock()
	r.listener = listener
	r.client = cli
	r.closed = false
	r.mu.Unlock()

	defer func() {
		_ = cli.Stop()
		r.mu.Lock()
		r.client = nil
		r.listener = nil
		r.closed = true
		r.mu.Unlock()
	}()

	for {
		rawConn, err := listener.Accept()
		if err != nil {
			if isExpectedListenerClose(err) {
				return nil
			}
			return err
		}

		enrollConn := rawConn
		var initialInbound []byte
		if prefixed, ok := rawConn.(*prefixedConn); ok {
			initialInbound = prefixed.takePrefix()
			if prefixed.Conn != nil {
				enrollConn = prefixed.Conn
			}
		}

		requestCtx, cancel := context.WithCancel(context.Background())
		state := &nativeGnetConnState{
			runtime: r,
			rawConn: rawConn,
			ctx:     requestCtx,
			cancel:  cancel,
		}
		state.appendInbound(initialInbound)
		putClassifyPeekBuffer(initialInbound)
		gconn, err := cli.EnrollContext(enrollConn, state)
		if err != nil {
			cancel()
			_ = rawConn.Close()
			if isExpectedListenerClose(err) {
				return nil
			}
			return err
		}
		if len(initialInbound) > 0 {
			_ = gconn.Wake(nil)
		}
	}
}

func (r *nativeGnetHTTPRuntime) Shutdown(ctx context.Context) error {
	return r.Close()
}

func (r *nativeGnetHTTPRuntime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	if r.listener != nil {
		if err := r.listener.Close(); err != nil && !isExpectedListenerClose(err) {
			errs = append(errs, err)
		}
	}
	if r.client != nil {
		if err := r.client.Stop(); err != nil {
			errs = append(errs, err)
		}
	}
	r.closed = true
	return errors.Join(errs...)
}

func (r *nativeGnetHTTPRuntime) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	state, _ := c.Context().(*nativeGnetConnState)
	if state == nil {
		return nil, gnet.Close
	}
	state.gconn = c
	if r.readTimeout > 0 {
		_ = c.SetReadDeadline(time.Now().Add(r.readTimeout))
	}
	return nil, gnet.None
}

func (r *nativeGnetHTTPRuntime) OnClose(c gnet.Conn, err error) gnet.Action {
	state, _ := c.Context().(*nativeGnetConnState)
	if state != nil {
		state.close()
	}
	return gnet.None
}

func (r *nativeGnetHTTPRuntime) OnTraffic(c gnet.Conn) gnet.Action {
	state, _ := c.Context().(*nativeGnetConnState)
	if state == nil {
		return gnet.Close
	}
	if r.readTimeout > 0 {
		_ = c.SetReadDeadline(time.Now().Add(r.readTimeout))
	}

	buffered := c.InboundBuffered()
	if buffered > 0 {
		payload, err := c.Next(buffered)
		if err != nil {
			return gnet.Close
		}
		state.appendInbound(payload)
	}

	return state.startNext(c)
}

type nativeGnetConnState struct {
	runtime *nativeGnetHTTPRuntime
	rawConn net.Conn
	gconn   gnet.Conn

	ctx    context.Context
	cancel context.CancelFunc

	mu         sync.Mutex
	buf        []byte
	processing bool
	closed     bool
	detached   bool
}

func (s *nativeGnetConnState) appendInbound(payload []byte) {
	if len(payload) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf = append(s.buf, payload...)
}

func (s *nativeGnetConnState) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if s.cancel != nil {
		s.cancel()
	}
	if s.rawConn != nil {
		_ = s.rawConn.Close()
		s.rawConn = nil
	}
}

func (s *nativeGnetConnState) startNext(c gnet.Conn) gnet.Action {
	s.mu.Lock()
	if s.closed || s.detached || s.processing {
		s.mu.Unlock()
		return gnet.None
	}

	req, consumed, complete, parseErr := decodeBufferedRequest(s.buf, s.ctx, c.RemoteAddr())
	if parseErr != nil {
		s.mu.Unlock()
		return gnet.Close
	}
	if !complete {
		s.mu.Unlock()
		return gnet.None
	}

	remaining := len(s.buf) - consumed
	if remaining > 0 {
		copy(s.buf[:remaining], s.buf[consumed:])
	}
	s.buf = s.buf[:remaining]
	s.processing = true
	upgrade := isWebSocketUpgrade(req)
	if upgrade {
		s.detached = true
	}
	s.mu.Unlock()

	if upgrade {
		dupConn, err := duplicateConn(c)
		if err != nil {
			return gnet.Close
		}
		go s.handleDetachedRequest(req, dupConn)
		return gnet.Close
	}

	go s.handleHTTPRequest(req)
	return gnet.None
}

func (s *nativeGnetConnState) handleDetachedRequest(req *http.Request, dupConn net.Conn) {
	defer func() {
		if dupConn != nil {
			_ = dupConn.Close()
		}
	}()

	writer := newDetachedHTTPResponseWriter(dupConn, req)
	if !dispatchExecutableRoute(s.runtime.execRuntime, req, writer, req.RemoteAddr) {
		s.runtime.handler.ServeHTTP(writer, req)
	}
	_ = writer.finish()
}

func (s *nativeGnetConnState) handleHTTPRequest(req *http.Request) {
	writer := newGnetHTTPResponseWriter(s.gconn, req)
	if !dispatchExecutableRoute(s.runtime.execRuntime, req, writer, req.RemoteAddr) {
		s.runtime.handler.ServeHTTP(writer, req)
	}
	runtimeobs.RecordGnetHTTP1Response()
	_ = writer.finish()

	s.mu.Lock()
	if s.closed || s.detached {
		s.processing = false
		s.mu.Unlock()
		return
	}
	s.processing = false
	shouldWake := len(s.buf) > 0
	s.mu.Unlock()

	if writer.closeAfterResponse.Load() {
		return
	}
	if shouldWake {
		_ = s.gconn.Wake(nil)
	}
}

func decodeBufferedRequest(buffer []byte, baseCtx context.Context, remoteAddr net.Addr) (*http.Request, int, bool, error) {
	if len(buffer) == 0 {
		return nil, 0, false, nil
	}

	src := bytes.NewReader(buffer)
	reader := bufio.NewReader(src)
	req, err := http.ReadRequest(reader)
	if err != nil {
		if isPartialHTTPRequestError(err) {
			return nil, 0, false, nil
		}
		return nil, 0, false, err
	}

	if req.ContentLength == 0 && !headerContainsToken(req.Header, "Transfer-Encoding", "chunked") {
		_ = req.Body.Close()
		req.Body = http.NoBody
		req.ContentLength = 0
	} else {
		bodyBytes, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			if isPartialHTTPRequestError(err) {
				return nil, 0, false, nil
			}
			return nil, 0, false, err
		}

		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}
	if baseCtx != nil {
		req = req.WithContext(baseCtx)
	}
	if remoteAddr != nil {
		req.RemoteAddr = remoteAddr.String()
	}

	consumed := len(buffer) - src.Len() - reader.Buffered()
	if consumed < 0 || consumed > len(buffer) {
		return nil, 0, false, fmt.Errorf("invalid request consumption: %d", consumed)
	}
	return req, consumed, true, nil
}

func isPartialHTTPRequestError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "unexpected eof") ||
		strings.Contains(msg, "short body") ||
		strings.Contains(msg, "malformed chunked encoding")
}

func isWebSocketUpgrade(req *http.Request) bool {
	if req == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(req.Header.Get("Upgrade")), "websocket") &&
		strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}

func duplicateConn(c gnet.Conn) (net.Conn, error) {
	if c == nil {
		return nil, errors.New("gnet conn is nil")
	}
	fd, err := c.Dup()
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "gnet-dup")
	if file == nil {
		return nil, errors.New("failed to wrap duplicated fd")
	}
	defer file.Close()
	return net.FileConn(file)
}

type gnetHTTPResponseWriter struct {
	conn    gnet.Conn
	request *http.Request
	header  http.Header

	mu         sync.Mutex
	statusCode int
	wroteHead  bool
	headSent   bool
	finished   bool
	chunked    bool
	size       int
	writeErr   error
	bodyBuffer []byte
	bodyInline [512]byte
	bodyLen    int

	closeAfterResponse atomic.Bool
}

func newGnetHTTPResponseWriter(conn gnet.Conn, req *http.Request) *gnetHTTPResponseWriter {
	return &gnetHTTPResponseWriter{
		conn:    conn,
		request: req,
		header:  make(http.Header),
		size:    -1,
	}
}

func (w *gnetHTTPResponseWriter) Header() http.Header {
	return w.header
}

func (w *gnetHTTPResponseWriter) Written() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.wroteHead
}

func (w *gnetHTTPResponseWriter) StatusCode() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.statusCode
}

func (w *gnetHTTPResponseWriter) Size() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.size
}

func (w *gnetHTTPResponseWriter) WriteHeader(statusCode int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.prepareHeaderLocked(statusCode)
}

func (w *gnetHTTPResponseWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.finished {
		return 0, http.ErrHandlerTimeout
	}
	if !w.wroteHead {
		w.prepareHeaderLocked(http.StatusOK)
	}
	if !responseHasBody(w.statusCode, w.request) {
		return len(p), nil
	}

	if w.headSent {
		runtimeobs.RecordGnetHTTP1DirectWriteAfterFlush()
		payload := append([]byte(nil), p...)
		if w.chunked {
			payload = encodeChunk(payload)
		}
		if err := w.asyncWriteLocked(payload); err != nil {
			return 0, err
		}
	} else {
		w.appendBodyLocked(p)
	}
	if w.size < 0 {
		w.size = 0
	}
	w.size += len(p)
	return len(p), nil
}

func (w *gnetHTTPResponseWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.finished {
		return
	}
	if !w.wroteHead {
		w.prepareHeaderLocked(http.StatusOK)
	}
	if !responseHasBody(w.statusCode, w.request) {
		if !w.headSent {
			w.sendHeadLocked()
		}
		return
	}
	if w.headSent {
		return
	}
	runtimeobs.RecordGnetHTTP1ChunkedFlushFallback()
	w.header.Del("Content-Length")
	w.header.Set("Transfer-Encoding", "chunked")
	w.chunked = true
	if err := w.sendHeadLocked(); err != nil {
		return
	}
	if w.bufferedBodyLenLocked() > 0 {
		payload := encodeChunk(w.bodyBytesLocked())
		w.resetBodyLocked()
		_ = w.asyncWriteLocked(payload)
	}
}

func (w *gnetHTTPResponseWriter) finish() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.finished {
		return w.writeErr
	}
	if !w.wroteHead {
		w.prepareHeaderLocked(http.StatusOK)
	}
	if !w.headSent {
		if responseHasBody(w.statusCode, w.request) {
			if !w.chunked && !headerContainsToken(w.header, "Transfer-Encoding", "chunked") {
				if w.header.Get("Content-Length") == "" {
					w.header.Set("Content-Length", strconv.Itoa(w.bufferedBodyLenLocked()))
				}
				if err := w.sendBufferedResponseLocked(); err != nil {
					w.finished = true
					return err
				}
			} else {
				runtimeobs.RecordGnetHTTP1ChunkedHeaderFallback()
				w.header.Del("Content-Length")
				w.header.Set("Transfer-Encoding", "chunked")
				w.chunked = true
				if err := w.sendHeadLocked(); err != nil {
					w.finished = true
					return err
				}
				if w.bufferedBodyLenLocked() > 0 {
					payload := encodeChunk(w.bodyBytesLocked())
					w.resetBodyLocked()
					if err := w.asyncWriteLocked(payload); err != nil {
						w.finished = true
						return err
					}
				}
			}
		} else {
			if err := w.sendBufferedResponseLocked(); err != nil {
				w.finished = true
				return err
			}
		}
	}
	if w.chunked {
		if err := w.asyncWriteLocked([]byte("0\r\n\r\n")); err != nil {
			w.finished = true
			return err
		}
	}
	w.finished = true
	if w.closeAfterResponse.Load() && w.conn != nil {
		if err := w.conn.CloseWithCallback(nil); err != nil && w.writeErr == nil {
			w.writeErr = err
		}
	}
	return w.writeErr
}

func (w *gnetHTTPResponseWriter) prepareHeaderLocked(statusCode int) {
	if w.wroteHead {
		return
	}
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}
	w.statusCode = statusCode
	w.wroteHead = true

	hasBody := responseHasBody(statusCode, w.request)
	if !hasBody {
		w.header.Del("Transfer-Encoding")
		if w.header.Get("Content-Length") == "" {
			w.header.Set("Content-Length", "0")
		}
	}

	if shouldCloseAfterResponse(w.request) {
		w.closeAfterResponse.Store(true)
		w.header.Set("Connection", "close")
	}
	if w.header.Get("Date") == "" {
		w.header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}
}

func (w *gnetHTTPResponseWriter) sendHeadLocked() error {
	if w.headSent {
		return w.writeErr
	}
	if w.writeErr != nil {
		return w.writeErr
	}
	w.headSent = true
	if err := w.asyncWriteLocked(encodeHTTPResponseHead(w.statusCode, w.header)); err != nil {
		w.writeErr = err
	}
	return w.writeErr
}

func (w *gnetHTTPResponseWriter) sendBufferedResponseLocked() error {
	if w.headSent {
		return w.writeErr
	}
	if w.writeErr != nil {
		return w.writeErr
	}
	head := encodeHTTPResponseHead(w.statusCode, w.header)
	w.headSent = true
	runtimeobs.RecordGnetHTTP1BufferedResponse(w.bodyBuffer == nil, w.header.Get("Content-Length") != "")
	body := w.bodyBytesLocked()
	if len(body) == 0 {
		if err := w.asyncWriteLocked(head); err != nil {
			w.writeErr = err
		}
		return w.writeErr
	}

	payload := make([]byte, 0, len(head)+len(body))
	payload = append(payload, head...)
	payload = append(payload, body...)
	w.resetBodyLocked()
	if err := w.asyncWriteLocked(payload); err != nil {
		w.writeErr = err
	}
	return w.writeErr
}

func (w *gnetHTTPResponseWriter) asyncWriteLocked(payload []byte) error {
	if len(payload) == 0 || w.conn == nil {
		return nil
	}
	if w.writeErr != nil {
		return w.writeErr
	}
	runtimeobs.RecordGnetHTTP1AsyncWrite()
	data := append([]byte(nil), payload...)
	if err := w.conn.AsyncWrite(data, nil); err != nil {
		w.writeErr = err
	}
	return w.writeErr
}

func (w *gnetHTTPResponseWriter) appendBodyLocked(payload []byte) {
	if len(payload) == 0 {
		return
	}
	if w.bodyBuffer != nil {
		w.bodyBuffer = append(w.bodyBuffer, payload...)
		return
	}
	if w.bodyLen+len(payload) <= len(w.bodyInline) {
		copy(w.bodyInline[w.bodyLen:], payload)
		w.bodyLen += len(payload)
		return
	}

	w.bodyBuffer = make([]byte, 0, w.bodyLen+len(payload))
	runtimeobs.RecordGnetHTTP1HeapBufferSpill()
	if w.bodyLen > 0 {
		w.bodyBuffer = append(w.bodyBuffer, w.bodyInline[:w.bodyLen]...)
	}
	w.bodyLen = 0
	w.bodyBuffer = append(w.bodyBuffer, payload...)
}

func (w *gnetHTTPResponseWriter) bodyBytesLocked() []byte {
	if w.bodyBuffer != nil {
		return w.bodyBuffer
	}
	return w.bodyInline[:w.bodyLen]
}

func (w *gnetHTTPResponseWriter) bufferedBodyLenLocked() int {
	if w.bodyBuffer != nil {
		return len(w.bodyBuffer)
	}
	return w.bodyLen
}

func (w *gnetHTTPResponseWriter) resetBodyLocked() {
	w.bodyBuffer = nil
	w.bodyLen = 0
}

func headerContainsToken(header http.Header, key, token string) bool {
	if header == nil {
		return false
	}
	for _, value := range header.Values(key) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

type detachedHTTPResponseWriter struct {
	conn    net.Conn
	request *http.Request
	header  http.Header

	statusCode int
	wroteHead  bool
	size       int
	hijacked   bool
	readWriter *bufio.ReadWriter
}

func newDetachedHTTPResponseWriter(conn net.Conn, req *http.Request) *detachedHTTPResponseWriter {
	return &detachedHTTPResponseWriter{
		conn:    conn,
		request: req,
		header:  make(http.Header),
		size:    -1,
	}
}

func (w *detachedHTTPResponseWriter) Header() http.Header {
	return w.header
}

func (w *detachedHTTPResponseWriter) Written() bool {
	return w.wroteHead
}

func (w *detachedHTTPResponseWriter) StatusCode() int {
	return w.statusCode
}

func (w *detachedHTTPResponseWriter) Size() int {
	return w.size
}

func (w *detachedHTTPResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHead || w.hijacked {
		return
	}
	w.statusCode = statusCode
	w.wroteHead = true
	if w.size < 0 {
		w.size = 0
	}
	_, _ = w.conn.Write(encodeHTTPResponseHead(statusCode, w.header))
}

func (w *detachedHTTPResponseWriter) Write(p []byte) (int, error) {
	if w.hijacked {
		return 0, http.ErrHijacked
	}
	if !w.wroteHead {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.conn.Write(p)
	if n > 0 {
		if w.size < 0 {
			w.size = 0
		}
		w.size += n
	}
	return n, err
}

func (w *detachedHTTPResponseWriter) Flush() {
}

func (w *detachedHTTPResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.hijacked {
		return nil, nil, http.ErrHijacked
	}
	w.hijacked = true
	if w.readWriter == nil {
		w.readWriter = bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn))
	}
	return w.conn, w.readWriter, nil
}

func (w *detachedHTTPResponseWriter) finish() error {
	if w.hijacked {
		return nil
	}
	if !w.wroteHead {
		w.WriteHeader(http.StatusOK)
	}
	return nil
}

func encodeHTTPResponseHead(statusCode int, header http.Header) []byte {
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}
	var buf bytes.Buffer
	reason := http.StatusText(statusCode)
	if reason == "" {
		reason = "Status"
	}
	buf.WriteString("HTTP/1.1 ")
	buf.WriteString(strconv.Itoa(statusCode))
	buf.WriteByte(' ')
	buf.WriteString(reason)
	buf.WriteString("\r\n")
	for key, values := range header {
		for _, value := range values {
			buf.WriteString(key)
			buf.WriteString(": ")
			buf.WriteString(value)
			buf.WriteString("\r\n")
		}
	}
	buf.WriteString("\r\n")
	return buf.Bytes()
}

func encodeChunk(payload []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(strconv.FormatInt(int64(len(payload)), 16))
	buf.WriteString("\r\n")
	buf.Write(payload)
	buf.WriteString("\r\n")
	return buf.Bytes()
}

func responseHasBody(statusCode int, req *http.Request) bool {
	if req != nil && strings.EqualFold(req.Method, http.MethodHead) {
		return false
	}
	return !((statusCode >= 100 && statusCode < 200) || statusCode == http.StatusNoContent || statusCode == http.StatusNotModified)
}

func shouldCloseAfterResponse(req *http.Request) bool {
	if req == nil {
		return false
	}
	return req.Close
}
