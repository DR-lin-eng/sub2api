package middleware

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

const defaultNonstreamProcessingKeepaliveInterval = 10 * time.Second

type responseWriterUnwrapper interface {
	Unwrap() http.ResponseWriter
}

type processingKeepaliveWriter struct {
	gin.ResponseWriter
	finalStarted atomic.Bool
}

func (w *processingKeepaliveWriter) markFinalStarted() {
	if w != nil {
		w.finalStarted.Store(true)
	}
}

func (w *processingKeepaliveWriter) WriteHeader(code int) {
	if code < 100 || code >= 200 {
		w.markFinalStarted()
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *processingKeepaliveWriter) WriteHeaderNow() {
	if status := w.Status(); status < 100 || status >= 200 {
		w.markFinalStarted()
	}
	w.ResponseWriter.WriteHeaderNow()
}

func (w *processingKeepaliveWriter) Write(data []byte) (int, error) {
	w.markFinalStarted()
	return w.ResponseWriter.Write(data)
}

func (w *processingKeepaliveWriter) WriteString(value string) (int, error) {
	w.markFinalStarted()
	return w.ResponseWriter.WriteString(value)
}

func (w *processingKeepaliveWriter) Flush() {
	w.markFinalStarted()
	w.ResponseWriter.Flush()
}

func (w *processingKeepaliveWriter) Unwrap() http.ResponseWriter {
	if w == nil {
		return nil
	}
	return unwrapHTTPResponseWriter(w.ResponseWriter)
}

func unwrapHTTPResponseWriter(writer http.ResponseWriter) http.ResponseWriter {
	seen := make(map[http.ResponseWriter]struct{})
	current := writer
	for current != nil {
		if _, ok := seen[current]; ok {
			break
		}
		seen[current] = struct{}{}
		unwrapper, ok := current.(responseWriterUnwrapper)
		if !ok {
			break
		}
		next := unwrapper.Unwrap()
		if next == nil || next == current {
			break
		}
		current = next
	}
	return current
}

func shouldSkipProcessingKeepalive(req *http.Request) bool {
	if req == nil {
		return true
	}
	switch req.Method {
	case http.MethodHead, http.MethodOptions:
		return true
	}
	if strings.EqualFold(strings.TrimSpace(req.Header.Get("Upgrade")), "websocket") {
		return true
	}
	connection := strings.ToLower(strings.TrimSpace(req.Header.Get("Connection")))
	return strings.Contains(connection, "upgrade")
}

func emitProcessingKeepalive(writer http.ResponseWriter) {
	if writer == nil {
		return
	}
	writer.WriteHeader(http.StatusProcessing)
	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func newNonstreamProcessingKeepalive(interval time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if interval <= 0 || c == nil || shouldSkipProcessingKeepalive(c.Request) {
			c.Next()
			return
		}

		keepaliveWriter := &processingKeepaliveWriter{ResponseWriter: c.Writer}
		c.Writer = keepaliveWriter

		rawWriter := unwrapHTTPResponseWriter(keepaliveWriter)
		if rawWriter == nil {
			c.Next()
			return
		}

		done := make(chan struct{})
		var stopOnce sync.Once
		stop := func() {
			stopOnce.Do(func() {
				close(done)
			})
		}
		defer stop()

		go func(req *http.Request, writer *processingKeepaliveWriter, base http.ResponseWriter) {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-req.Context().Done():
					return
				case <-ticker.C:
					if writer.finalStarted.Load() {
						return
					}
					emitProcessingKeepalive(base)
				}
			}
		}(c.Request, keepaliveWriter, rawWriter)

		c.Next()
	}
}

func NonstreamProcessingKeepalive() gin.HandlerFunc {
	return newNonstreamProcessingKeepalive(defaultNonstreamProcessingKeepaliveInterval)
}

// NonstreamProcessingKeepaliveForTest exposes a custom interval for package-external tests.
func NonstreamProcessingKeepaliveForTest(interval time.Duration) gin.HandlerFunc {
	return newNonstreamProcessingKeepalive(interval)
}
