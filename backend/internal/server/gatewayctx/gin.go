package gatewayctx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

type ginGatewayContext struct {
	gin *gin.Context
}

func AdaptGinHandler(fn HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if fn == nil {
			return
		}
		fn(FromGin(c))
	}
}

func FromGin(c *gin.Context) GatewayContext {
	if c == nil {
		return nil
	}
	return &ginGatewayContext{gin: c}
}

func (c *ginGatewayContext) Context() context.Context {
	if c == nil || c.gin == nil || c.gin.Request == nil {
		return context.Background()
	}
	return c.gin.Request.Context()
}

func (c *ginGatewayContext) Request() *http.Request {
	if c == nil || c.gin == nil {
		return nil
	}
	return c.gin.Request
}

func (c *ginGatewayContext) SetRequest(req *http.Request) {
	if c == nil || c.gin == nil {
		return
	}
	c.gin.Request = req
}

func (c *ginGatewayContext) Value(key string) (any, bool) {
	if c == nil || c.gin == nil {
		return nil, false
	}
	return c.gin.Get(key)
}

func (c *ginGatewayContext) SetValue(key string, value any) {
	if c == nil || c.gin == nil {
		return
	}
	c.gin.Set(key, value)
}

func (c *ginGatewayContext) ClientIP() string {
	if c == nil || c.gin == nil {
		return ""
	}
	if ip := forwardedClientIPFromRequest(c.gin.Request); ip != "" {
		return ip
	}
	return normalizeForwardedClientIP(c.gin.ClientIP())
}

func (c *ginGatewayContext) Method() string {
	if req := c.Request(); req != nil {
		return req.Method
	}
	return ""
}

func (c *ginGatewayContext) Path() string {
	if req := c.Request(); req != nil && req.URL != nil {
		return req.URL.Path
	}
	return ""
}

func (c *ginGatewayContext) HeaderValue(name string) string {
	if c == nil || c.gin == nil {
		return ""
	}
	return c.gin.GetHeader(name)
}

func (c *ginGatewayContext) QueryValue(name string) string {
	if c == nil || c.gin == nil {
		return ""
	}
	return c.gin.Query(name)
}

func (c *ginGatewayContext) PathParam(name string) string {
	if c == nil || c.gin == nil {
		return ""
	}
	return c.gin.Param(name)
}

func (c *ginGatewayContext) BindJSON(target any) error {
	if c == nil || c.gin == nil {
		return nil
	}
	return c.gin.ShouldBindJSON(target)
}

func (c *ginGatewayContext) CookieValue(name string) (string, error) {
	if c == nil || c.gin == nil {
		return "", http.ErrNoCookie
	}
	return c.gin.Cookie(name)
}

func (c *ginGatewayContext) Abort() {
	if c == nil || c.gin == nil {
		return
	}
	c.gin.Abort()
}

func (c *ginGatewayContext) SetHeader(name, value string) {
	if c == nil || c.gin == nil {
		return
	}
	c.gin.Header(name, value)
}

func (c *ginGatewayContext) Header() http.Header {
	if c == nil || c.gin == nil {
		return http.Header{}
	}
	return c.gin.Writer.Header()
}

func (c *ginGatewayContext) SetStatus(status int) {
	if c == nil || c.gin == nil {
		return
	}
	c.gin.Status(status)
}

func (c *ginGatewayContext) SetCookie(cookie *http.Cookie) {
	if c == nil || c.gin == nil || cookie == nil {
		return
	}
	http.SetCookie(c.gin.Writer, cookie)
}

func (c *ginGatewayContext) Redirect(status int, location string) {
	if c == nil || c.gin == nil {
		return
	}
	c.gin.Redirect(status, location)
}

func (c *ginGatewayContext) ResponseWritten() bool {
	if c == nil || c.gin == nil || c.gin.Writer == nil {
		return false
	}
	return c.gin.Writer.Written()
}

func (c *ginGatewayContext) ResponseSize() int {
	if c == nil || c.gin == nil || c.gin.Writer == nil {
		return -1
	}
	return c.gin.Writer.Size()
}

func (c *ginGatewayContext) WriteJSON(status int, value any) {
	if c == nil || c.gin == nil {
		return
	}
	c.gin.JSON(status, value)
}

func (c *ginGatewayContext) WriteBytes(status int, payload []byte) (int, error) {
	if c == nil || c.gin == nil {
		return 0, nil
	}
	if status > 0 {
		c.gin.Status(status)
	}
	return c.gin.Writer.Write(payload)
}

func (c *ginGatewayContext) WriteReader(status int, contentType string, reader io.Reader, size int64) error {
	if c == nil || c.gin == nil {
		return nil
	}
	extraHeaders := map[string]string{}
	if strings.TrimSpace(contentType) != "" {
		extraHeaders["Content-Type"] = contentType
	}
	c.gin.DataFromReader(status, size, contentType, reader, extraHeaders)
	return nil
}

func (c *ginGatewayContext) ServeFile(path string) error {
	if c == nil || c.gin == nil {
		return nil
	}
	c.gin.File(path)
	return nil
}

func (c *ginGatewayContext) ServeFileAttachment(path, filename string) error {
	if c == nil || c.gin == nil {
		return nil
	}
	if strings.TrimSpace(filename) == "" {
		filename = filepath.Base(path)
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	c.gin.FileAttachment(path, filename)
	return nil
}

func (c *ginGatewayContext) Flush() error {
	if c == nil || c.gin == nil {
		return nil
	}
	flusher, ok := c.gin.Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("gin writer does not implement http.Flusher")
	}
	flusher.Flush()
	return nil
}

func (c *ginGatewayContext) WriteSSEComment(comment string) error {
	if c == nil || c.gin == nil {
		return nil
	}
	c.SetHeader("Content-Type", "text/event-stream")
	line := strings.TrimRight(comment, "\r\n")
	if line == "" {
		line = ":"
	} else if !strings.HasPrefix(line, ":") {
		line = ":" + line
	}
	if _, err := c.WriteBytes(http.StatusOK, []byte(line+"\n\n")); err != nil {
		return err
	}
	return c.Flush()
}

func (c *ginGatewayContext) AcceptWebSocket(opts WebSocketAcceptOptions) (WebSocketConn, error) {
	if c == nil || c.gin == nil {
		return nil, ErrWebSocketNotSupported
	}
	conn, err := coderws.Accept(c.gin.Writer, c.gin.Request, &coderws.AcceptOptions{
		CompressionMode: websocketCompressionMode(opts.CompressionEnabled),
		Subprotocols:    opts.Subprotocols,
	})
	if err != nil {
		return nil, err
	}
	return &coderWebSocketConn{conn: conn}, nil
}

func (c *ginGatewayContext) Native() any {
	if c == nil {
		return nil
	}
	return c.gin
}

type coderWebSocketConn struct {
	conn *coderws.Conn
}

func (c *coderWebSocketConn) Read(ctx context.Context) (MessageType, []byte, error) {
	if c == nil || c.conn == nil {
		return MessageText, nil, ErrWebSocketNotSupported
	}
	msgType, payload, err := c.conn.Read(ctx)
	switch msgType {
	case coderws.MessageBinary:
		return MessageBinary, payload, err
	default:
		return MessageText, payload, err
	}
}

func (c *coderWebSocketConn) Write(ctx context.Context, msgType MessageType, payload []byte) error {
	if c == nil || c.conn == nil {
		return ErrWebSocketNotSupported
	}
	coderType := coderws.MessageText
	if msgType == MessageBinary {
		coderType = coderws.MessageBinary
	}
	return c.conn.Write(ctx, coderType, payload)
}

func (c *coderWebSocketConn) CloseNow() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.CloseNow()
}

func (c *coderWebSocketConn) Native() any {
	if c == nil {
		return nil
	}
	return c.conn
}

func websocketCompressionMode(enabled bool) coderws.CompressionMode {
	if enabled {
		return coderws.CompressionContextTakeover
	}
	return coderws.CompressionDisabled
}
