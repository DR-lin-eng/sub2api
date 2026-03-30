package service

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var defaultNonstreamKeepaliveInterval = 10 * time.Second

const requestNonstreamKeepaliveContextKey = "request_nonstream_keepalive"

type nonstreamKeepaliveController struct {
	writer  gin.ResponseWriter
	flusher http.Flusher
	done    chan struct{}
	stopped chan struct{}
	emitted atomic.Bool
}

func startNonstreamKeepalive(c *gin.Context, interval time.Duration) *nonstreamKeepaliveController {
	if c == nil || c.Writer == nil || interval <= 0 {
		return nil
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil
	}
	state := getRequestOutputState(c)

	ctrl := &nonstreamKeepaliveController{
		writer:  c.Writer,
		flusher: flusher,
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}

	go func(req *http.Request) {
		defer close(ctrl.stopped)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctrl.done:
				return
			case <-req.Context().Done():
				return
			case <-ticker.C:
				_, err := ctrl.writer.WriteString("\n")
				if err != nil {
					return
				}
				ctrl.flusher.Flush()
				ctrl.emitted.Store(true)
				if state != nil {
					state.protocol.CompareAndSwap(int32(gatewayResponseProtocolUnknown), int32(gatewayResponseProtocolBufferedJSON))
					state.protocolBegan.Store(true)
				}
			}
		}
	}(c.Request)

	return ctrl
}

func (c *nonstreamKeepaliveController) stop() {
	if c == nil {
		return
	}
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	<-c.stopped
}

func (c *nonstreamKeepaliveController) emittedAny() bool {
	return c != nil && c.emitted.Load()
}

func getRequestNonstreamKeepalive(c *gin.Context) *nonstreamKeepaliveController {
	if c == nil {
		return nil
	}
	raw, ok := c.Get(requestNonstreamKeepaliveContextKey)
	if !ok {
		return nil
	}
	ctrl, _ := raw.(*nonstreamKeepaliveController)
	return ctrl
}

func ensureBufferedJSONResponseHeaders(c *gin.Context) {
	if c == nil || c.Writer == nil {
		return
	}
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Status(http.StatusOK)
}

func startBufferedJSONKeepalive(c *gin.Context) *nonstreamKeepaliveController {
	if c == nil || c.Writer == nil {
		return nil
	}
	if existing := getRequestNonstreamKeepalive(c); existing != nil {
		return existing
	}
	ensureBufferedJSONResponseHeaders(c)
	ctrl := startNonstreamKeepalive(c, defaultNonstreamKeepaliveInterval)
	if ctrl != nil {
		c.Set(requestNonstreamKeepaliveContextKey, ctrl)
	}
	return ctrl
}

func startOpenAINonstreamKeepalive(c *gin.Context, reqStream bool) *nonstreamKeepaliveController {
	if c == nil || c.Writer == nil || reqStream {
		return nil
	}
	return startBufferedJSONKeepalive(c)
}

func startAnthropicBufferedKeepalive(c *gin.Context) *nonstreamKeepaliveController {
	if c == nil || c.Writer == nil {
		return nil
	}
	return startBufferedJSONKeepalive(c)
}

func getOpenAICompactKeepalive(c *gin.Context) *nonstreamKeepaliveController {
	return getRequestNonstreamKeepalive(c)
}

func startOpenAICompactKeepalive(c *gin.Context, account *Account, reqStream bool) *nonstreamKeepaliveController {
	if c == nil || c.Writer == nil || reqStream || account == nil || account.Type != AccountTypeOAuth || !isOpenAIResponsesCompactPath(c) {
		return nil
	}
	return startBufferedJSONKeepalive(c)
}

func stopOpenAICompactKeepalive(c *gin.Context) *nonstreamKeepaliveController {
	return stopRequestNonstreamKeepalive(c)
}

func stopRequestNonstreamKeepalive(c *gin.Context) *nonstreamKeepaliveController {
	ctrl := getRequestNonstreamKeepalive(c)
	if ctrl != nil {
		ctrl.stop()
	}
	return ctrl
}

func writeBufferedJSONAfterResponseStarted(c *gin.Context, body []byte) bool {
	if c == nil || c.Writer == nil || !c.Writer.Written() || len(body) == 0 || requestPayloadStarted(c) {
		return false
	}
	ctrl := stopRequestNonstreamKeepalive(c)
	if ctrl == nil && !RequestUsesBufferedJSON(c) {
		return false
	}
	if ctrl != nil && !ctrl.emittedAny() {
		return false
	}
	markRequestPayloadStarted(c, gatewayResponseProtocolBufferedJSON)
	_, err := c.Writer.Write(body)
	return err == nil
}

func WriteOpenAIErrorAfterResponseStarted(c *gin.Context, errType, message string) bool {
	if errType == "" {
		errType = "upstream_error"
	}
	if message == "" {
		message = "Upstream request failed"
	}
	return writeBufferedJSONAfterResponseStarted(c, []byte(`{"error":{"type":`+strconv.Quote(errType)+`,"message":`+strconv.Quote(message)+`}}`))
}

func WriteAnthropicBufferedErrorAfterResponseStarted(c *gin.Context, errType, message string) bool {
	if errType == "" {
		errType = "api_error"
	}
	if message == "" {
		message = "Upstream request failed"
	}
	return writeBufferedJSONAfterResponseStarted(c, []byte(`{"type":"error","error":{"type":`+strconv.Quote(errType)+`,"message":`+strconv.Quote(message)+`}}`))
}

func WriteOpenAICompactErrorAfterResponseStarted(c *gin.Context, errType, message string) bool {
	if !isOpenAIResponsesCompactPath(c) {
		return false
	}
	return WriteOpenAIErrorAfterResponseStarted(c, errType, message)
}
