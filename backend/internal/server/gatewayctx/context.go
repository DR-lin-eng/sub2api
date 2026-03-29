package gatewayctx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

var ErrWebSocketNotSupported = errors.New("websocket upgrade is not supported by this gateway context")

type MessageType int

const (
	MessageText MessageType = iota + 1
	MessageBinary
)

type WebSocketConn interface {
	Read(ctx context.Context) (MessageType, []byte, error)
	Write(ctx context.Context, msgType MessageType, payload []byte) error
	CloseNow() error
	Native() any
}

type WebSocketAcceptOptions struct {
	CompressionEnabled bool
	Subprotocols       []string
}

type GatewayContext interface {
	Context() context.Context
	Request() *http.Request
	SetRequest(*http.Request)
	Value(string) (any, bool)
	SetValue(string, any)
	ClientIP() string
	Method() string
	Path() string
	HeaderValue(name string) string
	QueryValue(name string) string
	PathParam(name string) string
	BindJSON(target any) error
	Abort()
	SetHeader(name, value string)
	Header() http.Header
	SetStatus(status int)
	ResponseWritten() bool
	ResponseSize() int
	WriteJSON(status int, value any)
	WriteBytes(status int, payload []byte) (int, error)
	Flush() error
	WriteSSEComment(comment string) error
	AcceptWebSocket(opts WebSocketAcceptOptions) (WebSocketConn, error)
	Native() any
}

type HandlerFunc func(GatewayContext)

type SSEOptions struct {
	ContentType  string
	CacheControl string
	RequestID    string
}

func WriteJSON(ctx GatewayContext, status int, value any) {
	if ctx == nil {
		return
	}
	ctx.WriteJSON(status, value)
}

func WriteStatus(ctx GatewayContext, status int) {
	if ctx == nil {
		return
	}
	ctx.SetStatus(status)
}

func WriteSSEComment(ctx GatewayContext, comment string) error {
	if ctx == nil {
		return nil
	}
	return ctx.WriteSSEComment(comment)
}

func PrepareSSE(ctx GatewayContext, opts SSEOptions) {
	if ctx == nil {
		return
	}
	contentType := strings.TrimSpace(opts.ContentType)
	if contentType == "" {
		contentType = "text/event-stream"
	}
	cacheControl := strings.TrimSpace(opts.CacheControl)
	if cacheControl == "" {
		cacheControl = "no-cache"
	}
	ctx.SetHeader("Content-Type", contentType)
	ctx.SetHeader("Cache-Control", cacheControl)
	ctx.SetHeader("Connection", "keep-alive")
	ctx.SetHeader("X-Accel-Buffering", "no")
	if requestID := strings.TrimSpace(opts.RequestID); requestID != "" {
		ctx.SetHeader("x-request-id", requestID)
	}
	headers := ctx.Header()
	headers.Del("Content-Encoding")
	headers.Del("Content-Length")
	headers.Del("Transfer-Encoding")
}

func WriteSSEDataRaw(ctx GatewayContext, raw string) error {
	if ctx == nil {
		return nil
	}
	if _, err := ctx.WriteBytes(0, []byte("data: "+raw+"\n\n")); err != nil {
		return err
	}
	return ctx.Flush()
}

func WriteSSEEvent(ctx GatewayContext, event string, payload any) error {
	if ctx == nil {
		return nil
	}
	var frame strings.Builder
	if trimmedEvent := strings.TrimSpace(event); trimmedEvent != "" {
		frame.WriteString("event: ")
		frame.WriteString(trimmedEvent)
		frame.WriteByte('\n')
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	frame.WriteString("data: ")
	frame.Write(body)
	frame.WriteString("\n\n")
	if _, err := ctx.WriteBytes(0, []byte(frame.String())); err != nil {
		return err
	}
	return ctx.Flush()
}

func WriteSSEData(ctx GatewayContext, payload any) error {
	if ctx == nil {
		return nil
	}

	var data string
	switch v := payload.(type) {
	case string:
		data = v
	case []byte:
		data = string(v)
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return err
		}
		data = string(encoded)
	}

	ctx.SetHeader("Content-Type", "text/event-stream")
	if _, err := ctx.WriteBytes(http.StatusOK, []byte("data: "+normalizeSSEData(data)+"\n\n")); err != nil {
		return err
	}
	return ctx.Flush()
}

func normalizeSSEData(data string) string {
	if data == "" {
		return ""
	}
	lines := strings.Split(data, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, "\r")
	}
	return strings.Join(lines, "\ndata: ")
}
