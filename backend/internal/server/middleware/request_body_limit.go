package middleware

import (
	"errors"
	"io"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

// RequestBodyLimit 使用 MaxBytesReader 限制请求体大小。
func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		ApplyRequestBodyLimitContext(gatewayctx.FromGin(c), maxBytes)
		c.Next()
	}
}

func ApplyRequestBodyLimitContext(c gatewayctx.GatewayContext, maxBytes int64) {
	if c == nil || c.Request() == nil || c.Request().Body == nil || maxBytes <= 0 {
		return
	}
	req := c.Request().Clone(c.Request().Context())
	req.Body = &maxBytesReadCloser{ReadCloser: c.Request().Body, limit: maxBytes, remaining: maxBytes}
	c.SetRequest(req)
}

type maxBytesReadCloser struct {
	io.ReadCloser
	limit     int64
	remaining int64
	exceeded  bool
}

func (r *maxBytesReadCloser) Read(p []byte) (int, error) {
	if r == nil || r.ReadCloser == nil {
		return 0, io.EOF
	}
	if r.exceeded {
		return 0, &http.MaxBytesError{Limit: r.limit}
	}
	if r.remaining <= 0 {
		r.exceeded = true
		return 0, &http.MaxBytesError{Limit: r.limit}
	}
	if int64(len(p)) > r.remaining {
		p = p[:int(r.remaining)]
	}
	n, err := r.ReadCloser.Read(p)
	r.remaining -= int64(n)
	if errors.Is(err, io.EOF) && r.remaining < 0 {
		r.exceeded = true
		return n, &http.MaxBytesError{Limit: r.limit}
	}
	if err == nil && r.remaining == 0 {
		var one [1]byte
		peekN, peekErr := r.ReadCloser.Read(one[:])
		if peekN > 0 || (peekErr != nil && !errors.Is(peekErr, io.EOF)) {
			r.exceeded = true
			return n, &http.MaxBytesError{Limit: r.limit}
		}
	}
	return n, err
}
