package service

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

var defaultNonstreamKeepaliveInterval = 10 * time.Second

const requestNonstreamKeepaliveContextKey = "request_nonstream_keepalive"

type nonstreamKeepaliveController struct {
	writer  gatewayctx.GatewayContext
	done    chan struct{}
	stopped chan struct{}
	emitted atomic.Bool
}

func startNonstreamKeepalive(c *gin.Context, interval time.Duration) *nonstreamKeepaliveController {
	return startNonstreamKeepaliveContext(gatewayctx.FromGin(c), interval)
}

func startNonstreamKeepaliveContext(c gatewayctx.GatewayContext, interval time.Duration) *nonstreamKeepaliveController {
	if c == nil || interval <= 0 {
		return nil
	}

	ctrl := &nonstreamKeepaliveController{
		writer:  c,
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}

	go func(ctx gatewayctx.GatewayContext) {
		defer close(ctrl.stopped)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctrl.done:
				return
			case <-ctx.Context().Done():
				return
			case <-ticker.C:
				_, err := ctrl.writer.WriteBytes(0, []byte("\n"))
				if err != nil {
					return
				}
				_ = ctrl.writer.Flush()
				ctrl.emitted.Store(true)
				MarkRequestBufferedJSONStarted(c)
			}
		}
	}(c)

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

func getOpenAICompactKeepalive(c *gin.Context) *nonstreamKeepaliveController {
	return getOpenAICompactKeepaliveContext(gatewayctx.FromGin(c))
}

func getOpenAICompactKeepaliveContext(c gatewayctx.GatewayContext) *nonstreamKeepaliveController {
	if c == nil {
		return nil
	}
	raw, ok := c.Value(requestNonstreamKeepaliveContextKey)
	if !ok {
		return nil
	}
	ctrl, _ := raw.(*nonstreamKeepaliveController)
	return ctrl
}

func startOpenAICompactKeepalive(c *gin.Context, account *Account, reqStream bool) *nonstreamKeepaliveController {
	return startOpenAICompactKeepaliveContext(gatewayctx.FromGin(c), account, reqStream)
}

func startOpenAICompactKeepaliveContext(c gatewayctx.GatewayContext, account *Account, reqStream bool) *nonstreamKeepaliveController {
	if c == nil || reqStream || !isOpenAIResponsesCompactPathContext(c) {
		return nil
	}
	if account != nil && account.Type != AccountTypeOAuth {
		return nil
	}
	if existing := getOpenAICompactKeepaliveContext(c); existing != nil {
		return existing
	}

	c.SetHeader("Cache-Control", "no-cache, no-transform")
	c.SetHeader("X-Accel-Buffering", "no")
	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.SetStatus(200)

	ctrl := startNonstreamKeepaliveContext(c, defaultNonstreamKeepaliveInterval)
	if ctrl != nil {
		c.SetValue(requestNonstreamKeepaliveContextKey, ctrl)
	}
	return ctrl
}

func stopOpenAICompactKeepalive(c *gin.Context) *nonstreamKeepaliveController {
	return stopOpenAICompactKeepaliveContext(gatewayctx.FromGin(c))
}

func stopOpenAICompactKeepaliveContext(c gatewayctx.GatewayContext) *nonstreamKeepaliveController {
	ctrl := getOpenAICompactKeepaliveContext(c)
	if ctrl != nil {
		ctrl.stop()
	}
	return ctrl
}

func WriteOpenAICompactErrorAfterResponseStarted(c *gin.Context, errType, message string) bool {
	return WriteOpenAICompactErrorAfterResponseStartedContext(gatewayctx.FromGin(c), errType, message)
}

func WriteOpenAICompactErrorAfterResponseStartedContext(c gatewayctx.GatewayContext, errType, message string) bool {
	if c == nil || !c.ResponseWritten() || !isOpenAIResponsesCompactPathContext(c) {
		return false
	}
	return WriteOpenAIErrorAfterResponseStartedContext(c, errType, message)
}

func WriteOpenAIErrorAfterResponseStartedContext(c gatewayctx.GatewayContext, errType, message string) bool {
	if c == nil || !c.ResponseWritten() || requestPayloadStarted(c) {
		return false
	}
	ctrl := stopOpenAICompactKeepaliveContext(c)
	if ctrl != nil && !ctrl.emittedAny() {
		return false
	}
	if errType == "" {
		errType = "upstream_error"
	}
	if message == "" {
		message = "Upstream request failed"
	}
	markRequestPayloadStarted(c, gatewayResponseProtocolBufferedJSON)
	_, err := c.WriteBytes(0, []byte(`{"error":{"type":`+strconv.Quote(errType)+`,"message":`+strconv.Quote(message)+`}}`))
	return err == nil
}

func WriteAnthropicBufferedErrorAfterResponseStartedContext(c gatewayctx.GatewayContext, errType, message string) bool {
	if c == nil || !c.ResponseWritten() || requestPayloadStarted(c) {
		return false
	}
	ctrl := stopOpenAICompactKeepaliveContext(c)
	if ctrl != nil && !ctrl.emittedAny() {
		return false
	}
	if errType == "" {
		errType = "api_error"
	}
	if message == "" {
		message = "Upstream request failed"
	}
	markRequestPayloadStarted(c, gatewayResponseProtocolBufferedJSON)
	_, err := c.WriteBytes(0, []byte(`{"type":"error","error":{"type":`+strconv.Quote(errType)+`,"message":`+strconv.Quote(message)+`}}`))
	return err == nil
}
