package gatewayctx

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	coderws "github.com/coder/websocket"
)

type nativeGatewayContext struct {
	req      *http.Request
	writer   any
	values   map[string]any
	params   map[string]string
	clientIP string
	aborted  bool
}

type nativeHeaderWriter interface {
	Header() http.Header
	WriteHeader(statusCode int)
	Write([]byte) (int, error)
}

type nativeFlusher interface {
	Flush()
}

type nativeWritten interface {
	Written() bool
}

type nativeSizer interface {
	Size() int
}

func NewNative(req *http.Request, writer any, params map[string]string, clientIP string) GatewayContext {
	return &nativeGatewayContext{
		req:      req,
		writer:   writer,
		values:   make(map[string]any),
		params:   cloneStringMap(params),
		clientIP: clientIP,
	}
}

func (c *nativeGatewayContext) Context() context.Context {
	if c == nil || c.req == nil {
		return context.Background()
	}
	return c.req.Context()
}

func (c *nativeGatewayContext) Request() *http.Request {
	if c == nil {
		return nil
	}
	return c.req
}

func (c *nativeGatewayContext) SetRequest(req *http.Request) {
	if c == nil {
		return
	}
	c.req = req
}

func (c *nativeGatewayContext) Value(key string) (any, bool) {
	if c == nil || c.values == nil {
		return nil, false
	}
	v, ok := c.values[key]
	return v, ok
}

func (c *nativeGatewayContext) SetValue(key string, value any) {
	if c == nil {
		return
	}
	if c.values == nil {
		c.values = make(map[string]any)
	}
	c.values[key] = value
}

func (c *nativeGatewayContext) ClientIP() string {
	if c == nil {
		return ""
	}
	if c.clientIP != "" {
		return c.clientIP
	}
	if c.req == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(c.req.RemoteAddr)
	if err == nil {
		return host
	}
	return c.req.RemoteAddr
}

func (c *nativeGatewayContext) Method() string {
	if c == nil || c.req == nil {
		return ""
	}
	return c.req.Method
}

func (c *nativeGatewayContext) Path() string {
	if c == nil || c.req == nil || c.req.URL == nil {
		return ""
	}
	return c.req.URL.Path
}

func (c *nativeGatewayContext) HeaderValue(name string) string {
	if c == nil || c.req == nil {
		return ""
	}
	return c.req.Header.Get(name)
}

func (c *nativeGatewayContext) QueryValue(name string) string {
	if c == nil || c.req == nil || c.req.URL == nil {
		return ""
	}
	return c.req.URL.Query().Get(name)
}

func (c *nativeGatewayContext) PathParam(name string) string {
	if c == nil || c.params == nil {
		return ""
	}
	return c.params[name]
}

func (c *nativeGatewayContext) BindJSON(target any) error {
	if c == nil || c.req == nil || c.req.Body == nil {
		return nil
	}
	defer func() { _ = c.req.Body.Close() }()
	return json.NewDecoder(c.req.Body).Decode(target)
}

func (c *nativeGatewayContext) CookieValue(name string) (string, error) {
	if c == nil || c.req == nil {
		return "", http.ErrNoCookie
	}
	cookie, err := c.req.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func (c *nativeGatewayContext) Abort() {
	if c == nil {
		return
	}
	c.aborted = true
}

func (c *nativeGatewayContext) SetHeader(name, value string) {
	header := c.Header()
	if header == nil {
		return
	}
	header.Set(name, value)
}

func (c *nativeGatewayContext) Header() http.Header {
	if hw, ok := c.writer.(nativeHeaderWriter); ok {
		return hw.Header()
	}
	return http.Header{}
}

func (c *nativeGatewayContext) SetStatus(status int) {
	if hw, ok := c.writer.(nativeHeaderWriter); ok {
		hw.WriteHeader(status)
	}
}

func (c *nativeGatewayContext) SetCookie(cookie *http.Cookie) {
	if c == nil || cookie == nil {
		return
	}
	http.SetCookie(c.responseWriter(), cookie)
}

func (c *nativeGatewayContext) Redirect(status int, location string) {
	if c == nil {
		return
	}
	http.Redirect(c.responseWriter(), c.req, location, status)
}

func (c *nativeGatewayContext) ResponseWritten() bool {
	if w, ok := c.writer.(nativeWritten); ok {
		return w.Written()
	}
	return false
}

func (c *nativeGatewayContext) ResponseSize() int {
	if w, ok := c.writer.(nativeSizer); ok {
		return w.Size()
	}
	return -1
}

func (c *nativeGatewayContext) WriteJSON(status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		c.SetStatus(http.StatusInternalServerError)
		return
	}
	c.SetHeader("Content-Type", "application/json")
	_, _ = c.WriteBytes(status, body)
}

func (c *nativeGatewayContext) WriteBytes(status int, payload []byte) (int, error) {
	hw, ok := c.writer.(nativeHeaderWriter)
	if !ok {
		return 0, nil
	}
	if status > 0 {
		hw.WriteHeader(status)
	}
	return hw.Write(payload)
}

func (c *nativeGatewayContext) WriteReader(status int, contentType string, reader io.Reader, size int64) error {
	if c == nil {
		return nil
	}
	if strings.TrimSpace(contentType) != "" {
		c.SetHeader("Content-Type", contentType)
	}
	if size >= 0 {
		c.SetHeader("Content-Length", strconv.FormatInt(size, 10))
	}
	if status > 0 {
		c.SetStatus(status)
	}
	_, err := io.Copy(c.responseWriter(), reader)
	return err
}

func (c *nativeGatewayContext) ServeFile(path string) error {
	if c == nil || c.req == nil {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	http.ServeFile(c.responseWriter(), c.req, path)
	return nil
}

func (c *nativeGatewayContext) ServeFileAttachment(path, filename string) error {
	if c == nil || c.req == nil {
		return nil
	}
	if strings.TrimSpace(filename) == "" {
		filename = filepath.Base(path)
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	c.SetHeader("Content-Disposition", `attachment; filename="`+filename+`"`)
	http.ServeFile(c.responseWriter(), c.req, path)
	return nil
}

func (c *nativeGatewayContext) Flush() error {
	if flusher, ok := c.writer.(nativeFlusher); ok {
		flusher.Flush()
	}
	return nil
}

func (c *nativeGatewayContext) WriteSSEComment(comment string) error {
	line := comment
	if line == "" {
		line = ":"
	}
	if line != "" && line[0] != ':' {
		line = ":" + line
	}
	c.SetHeader("Content-Type", "text/event-stream")
	_, err := c.WriteBytes(http.StatusOK, []byte(line+"\n\n"))
	if err != nil {
		return err
	}
	return c.Flush()
}

func (c *nativeGatewayContext) AcceptWebSocket(opts WebSocketAcceptOptions) (WebSocketConn, error) {
	if c == nil || c.req == nil {
		return nil, ErrWebSocketNotSupported
	}
	conn, err := coderws.Accept(c.responseWriter(), c.req, &coderws.AcceptOptions{
		CompressionMode: websocketCompressionMode(opts.CompressionEnabled),
		Subprotocols:    opts.Subprotocols,
	})
	if err != nil {
		return nil, err
	}
	return &coderWebSocketConn{conn: conn}, nil
}

func (c *nativeGatewayContext) Native() any {
	if c == nil {
		return nil
	}
	return c.writer
}

func (c *nativeGatewayContext) responseWriter() http.ResponseWriter {
	if rw, ok := c.writer.(http.ResponseWriter); ok {
		return rw
	}
	return nativeHTTPResponseWriterAdapter{ctx: c}
}

type nativeHTTPResponseWriterAdapter struct {
	ctx *nativeGatewayContext
}

func (w nativeHTTPResponseWriterAdapter) Header() http.Header {
	if w.ctx == nil {
		return http.Header{}
	}
	return w.ctx.Header()
}

func (w nativeHTTPResponseWriterAdapter) WriteHeader(statusCode int) {
	if w.ctx == nil {
		return
	}
	w.ctx.SetStatus(statusCode)
}

func (w nativeHTTPResponseWriterAdapter) Write(payload []byte) (int, error) {
	if w.ctx == nil {
		return 0, nil
	}
	return w.ctx.WriteBytes(0, payload)
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
