package service

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var defaultNonstreamKeepaliveInterval = 10 * time.Second

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
				if ctrl.writer.Written() {
					return
				}
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
