package service

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var defaultNonstreamKeepaliveInterval = 10 * time.Second

const openAICompactKeepaliveContextKey = "openai_compact_nonstream_keepalive"

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

func getOpenAICompactKeepalive(c *gin.Context) *nonstreamKeepaliveController {
	if c == nil {
		return nil
	}
	raw, ok := c.Get(openAICompactKeepaliveContextKey)
	if !ok {
		return nil
	}
	ctrl, _ := raw.(*nonstreamKeepaliveController)
	return ctrl
}

func startOpenAICompactKeepalive(c *gin.Context, account *Account, reqStream bool) *nonstreamKeepaliveController {
	if c == nil || c.Writer == nil || reqStream || account == nil || account.Type != AccountTypeOAuth || !isOpenAIResponsesCompactPath(c) {
		return nil
	}
	if existing := getOpenAICompactKeepalive(c); existing != nil {
		return existing
	}

	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Status(http.StatusOK)

	ctrl := startNonstreamKeepalive(c, defaultNonstreamKeepaliveInterval)
	if ctrl != nil {
		c.Set(openAICompactKeepaliveContextKey, ctrl)
	}
	return ctrl
}

func stopOpenAICompactKeepalive(c *gin.Context) *nonstreamKeepaliveController {
	ctrl := getOpenAICompactKeepalive(c)
	if ctrl != nil {
		ctrl.stop()
	}
	return ctrl
}

func WriteOpenAICompactErrorAfterResponseStarted(c *gin.Context, errType, message string) bool {
	if c == nil || c.Writer == nil || !c.Writer.Written() || !isOpenAIResponsesCompactPath(c) {
		return false
	}
	ctrl := stopOpenAICompactKeepalive(c)
	if ctrl != nil && !ctrl.emittedAny() {
		return false
	}
	if errType == "" {
		errType = "upstream_error"
	}
	if message == "" {
		message = "Upstream request failed"
	}
	_, err := c.Writer.Write([]byte(`{"error":{"type":` + strconv.Quote(errType) + `,"message":` + strconv.Quote(message) + `}}`))
	return err == nil
}
