package middleware

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type processingRecorder struct {
	header      http.Header
	body        bytes.Buffer
	infoCodes   []int
	finalStatus int
	flushCount  int
}

func newProcessingRecorder() *processingRecorder {
	return &processingRecorder{header: make(http.Header)}
}

func (w *processingRecorder) Header() http.Header {
	return w.header
}

func (w *processingRecorder) WriteHeader(code int) {
	if code == http.StatusSwitchingProtocols {
		w.finalStatus = code
		return
	}
	if code >= 100 && code < 200 {
		w.infoCodes = append(w.infoCodes, code)
		return
	}
	if w.finalStatus == 0 {
		w.finalStatus = code
	}
}

func (w *processingRecorder) Write(data []byte) (int, error) {
	if w.finalStatus == 0 {
		w.finalStatus = http.StatusOK
	}
	return w.body.Write(data)
}

func (w *processingRecorder) Flush() {
	w.flushCount++
}

func (w *processingRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, fmt.Errorf("hijack not supported")
}

func (w *processingRecorder) CloseNotify() <-chan bool {
	ch := make(chan bool)
	return ch
}

func (w *processingRecorder) Push(string, *http.PushOptions) error {
	return http.ErrNotSupported
}

func TestNonstreamProcessingKeepalive_SlowHandlerEmits102AndPreservesFinalResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NonstreamProcessingKeepaliveForTest(20 * time.Millisecond))
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(55 * time.Millisecond)
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	rec := newProcessingRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	router.ServeHTTP(rec, req)

	require.NotEmpty(t, rec.infoCodes)
	require.Equal(t, http.StatusProcessing, rec.infoCodes[0])
	require.Equal(t, http.StatusCreated, rec.finalStatus)
	require.Contains(t, rec.body.String(), `"ok":true`)
}

func TestNonstreamProcessingKeepalive_FastHandlerDoesNotEmit102(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NonstreamProcessingKeepaliveForTest(50 * time.Millisecond))
	router.GET("/fast", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := newProcessingRecorder()
	req := httptest.NewRequest(http.MethodGet, "/fast", nil)
	router.ServeHTTP(rec, req)

	require.Empty(t, rec.infoCodes)
	require.Equal(t, http.StatusOK, rec.finalStatus)
	require.Equal(t, "ok", rec.body.String())
}

func TestNonstreamProcessingKeepalive_FinalWriteStopsFurther102(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NonstreamProcessingKeepaliveForTest(20 * time.Millisecond))
	router.GET("/write-first", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
		time.Sleep(50 * time.Millisecond)
	})

	rec := newProcessingRecorder()
	req := httptest.NewRequest(http.MethodGet, "/write-first", nil)
	router.ServeHTTP(rec, req)

	require.Empty(t, rec.infoCodes)
	require.Equal(t, http.StatusOK, rec.finalStatus)
}

func TestNonstreamProcessingKeepalive_UpgradeRequestSkips102(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NonstreamProcessingKeepaliveForTest(20 * time.Millisecond))
	router.GET("/ws", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.String(http.StatusSwitchingProtocols, "")
	})

	rec := newProcessingRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	router.ServeHTTP(rec, req)

	require.Empty(t, rec.infoCodes)
	require.Equal(t, http.StatusSwitchingProtocols, rec.finalStatus)
}

func TestNonstreamProcessingKeepalive_HeadAndOptionsSkip102(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NonstreamProcessingKeepaliveForTest(20 * time.Millisecond))
	router.HEAD("/head", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.Status(http.StatusNoContent)
	})
	router.OPTIONS("/options", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.Status(http.StatusNoContent)
	})

	headRec := newProcessingRecorder()
	headReq := httptest.NewRequest(http.MethodHead, "/head", nil)
	router.ServeHTTP(headRec, headReq)
	require.Empty(t, headRec.infoCodes)

	optionsRec := newProcessingRecorder()
	optionsReq := httptest.NewRequest(http.MethodOptions, "/options", nil)
	router.ServeHTTP(optionsRec, optionsReq)
	require.Empty(t, optionsRec.infoCodes)
}

func TestNonstreamProcessingKeepalive_StreamDelayedStartAllows102BeforeSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NonstreamProcessingKeepaliveForTest(20 * time.Millisecond))
	router.GET("/stream", func(c *gin.Context) {
		time.Sleep(35 * time.Millisecond)
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)
		flusher, ok := c.Writer.(http.Flusher)
		require.True(t, ok)
		_, err := c.Writer.Write([]byte("data: hello\n\n"))
		require.NoError(t, err)
		flusher.Flush()
	})

	rec := newProcessingRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	router.ServeHTTP(rec, req)

	require.NotEmpty(t, rec.infoCodes)
	require.Equal(t, http.StatusProcessing, rec.infoCodes[0])
	require.Equal(t, http.StatusOK, rec.finalStatus)
	require.Contains(t, rec.body.String(), "data: hello")
}

func TestNonstreamProcessingKeepalive_StreamQuickStartDoesNotEmit102(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NonstreamProcessingKeepaliveForTest(40 * time.Millisecond))
	router.GET("/stream-fast", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond)
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)
		flusher, ok := c.Writer.(http.Flusher)
		require.True(t, ok)
		_, err := c.Writer.Write([]byte("data: hi\n\n"))
		require.NoError(t, err)
		flusher.Flush()
	})

	rec := newProcessingRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stream-fast", nil)
	router.ServeHTTP(rec, req)

	require.Empty(t, rec.infoCodes)
	require.Equal(t, http.StatusOK, rec.finalStatus)
	require.Contains(t, rec.body.String(), "data: hi")
}

func TestNonstreamProcessingKeepalive_StopsAfterHandlerReturns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NonstreamProcessingKeepaliveForTest(15 * time.Millisecond))
	router.GET("/stop", func(c *gin.Context) {
		time.Sleep(20 * time.Millisecond)
		c.String(http.StatusOK, "done")
	})

	rec := newProcessingRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stop", nil)
	router.ServeHTTP(rec, req)

	infoCount := len(rec.infoCodes)
	time.Sleep(40 * time.Millisecond)
	require.Equal(t, infoCount, len(rec.infoCodes))
}
