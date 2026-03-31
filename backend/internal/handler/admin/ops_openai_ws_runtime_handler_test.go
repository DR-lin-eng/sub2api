package admin

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	ffi "github.com/Wei-Shaw/sub2api/internal/rustbridge/ffi"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func newOpsOpenAIWSRuntimeRouter(handler *OpsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/openai-ws-runtime", handler.GetOpenAIWSRuntime)
	return r
}

func newOpsOpenAIWSRuntimeService(cfg *config.Config) *service.OpsService {
	if cfg == nil {
		cfg = &config.Config{
			Ops: config.OpsConfig{Enabled: true},
		}
	}
	runtimeCfg := &config.Config{
		Ops:  cfg.Ops,
		Rust: cfg.Rust,
	}
	return service.NewOpsService(nil, nil, runtimeCfg, nil, nil, nil, nil, &service.OpenAIGatewayService{}, nil, nil, nil)
}

func TestOpsHandler_GetOpenAIWSRuntime_IncludesRustFFIMetrics(t *testing.T) {
	before := ffi.SnapshotMetrics()
	_ = ffi.BuildETagFromBytes([]byte("ops-runtime-rust-ffi"))
	_ = ffi.ParseOpenAIWSFrameSummary([]byte(`{"type":"response.completed","response":{"id":"resp_ops"}}`))

	h := NewOpsHandler(newOpsOpenAIWSRuntimeService(nil))
	r := newOpsOpenAIWSRuntimeRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/openai-ws-runtime", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if gjson.Get(rec.Body.String(), "code").Int() != 0 {
		t.Fatalf("expected success envelope, got body=%s", rec.Body.String())
	}
	if !gjson.Get(rec.Body.String(), "data.rust_ffi.total.calls").Exists() {
		t.Fatalf("expected rust_ffi metrics in runtime response, got body=%s", rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_ffi.total.calls").Int(); got < before.Total.Calls+1 {
		t.Fatalf("expected rust_ffi total calls to advance, before=%d got=%d body=%s", before.Total.Calls, got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_ffi.hash.calls").Int(); got < before.Hash.Calls+1 {
		t.Fatalf("expected rust_ffi hash calls to advance, before=%d got=%d body=%s", before.Hash.Calls, got, rec.Body.String())
	}
	if !gjson.Get(rec.Body.String(), "data.rust_ffi.event_parse.calls").Exists() {
		t.Fatalf("expected rust_ffi event_parse metrics in runtime response, got body=%s", rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_ffi.event_parse.calls").Int(); got < before.EventParse.Calls+1 {
		t.Fatalf("expected rust_ffi event_parse calls to advance, before=%d got=%d body=%s", before.EventParse.Calls, got, rec.Body.String())
	}
}

func TestOpsHandler_GetOpenAIWSRuntime_IncludesRustSidecarHealth(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "rust-sidecar.sock")
	var healthHits atomic.Int64

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		healthHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":                              "ok",
			"service":                             "test-sidecar",
			"version":                             "v0",
			"active_connections":                  2,
			"total_connections":                   3,
			"active_upgrades":                     1,
			"total_upgrades":                      4,
			"total_requests":                      5,
			"total_request_errors":                1,
			"upstream_unavailable_total":          2,
			"upstream_handshake_failed_total":     3,
			"upstream_request_failed_total":       4,
			"upgrade_errors_total":                5,
			"relay_bytes_downstream_to_upstream":  6,
			"relay_bytes_upstream_to_downstream":  7,
			"relay_frames_downstream_to_upstream": 8,
			"relay_frames_upstream_to_downstream": 9,
			"relay_close_frames_total":            10,
			"relay_ping_frames_total":             11,
			"relay_pong_frames_total":             12,
		})
	})

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	defer ln.Close()

	server := &http.Server{Handler: mux}
	defer server.Close()

	go func() {
		_ = server.Serve(ln)
	}()

	cfg := &config.Config{
		Ops: config.OpsConfig{Enabled: true},
		Rust: config.RustConfig{
			Sidecar: config.RustSidecarConfig{
				Enabled:               true,
				SocketPath:            socketPath,
				RequestTimeoutSeconds: 1,
			},
		},
	}

	h := NewOpsHandler(newOpsOpenAIWSRuntimeService(cfg))
	r := newOpsOpenAIWSRuntimeRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/openai-ws-runtime", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if gjson.Get(rec.Body.String(), "code").Int() != 0 {
		t.Fatalf("expected success envelope, got body=%s", rec.Body.String())
	}
	if !gjson.Get(rec.Body.String(), "data.rust_sidecar.enabled").Bool() {
		t.Fatalf("expected rust_sidecar.enabled=true, got body=%s", rec.Body.String())
	}
	if !gjson.Get(rec.Body.String(), "data.rust_sidecar.available").Bool() {
		t.Fatalf("expected rust_sidecar.available=true, got body=%s", rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_sidecar.active_connections").Int(); got != 2 {
		t.Fatalf("expected active_connections=2, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_sidecar.total_upgrades").Int(); got != 4 {
		t.Fatalf("expected total_upgrades=4, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_sidecar.total_requests").Int(); got != 5 {
		t.Fatalf("expected total_requests=5, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_sidecar.upgrade_errors_total").Int(); got != 5 {
		t.Fatalf("expected upgrade_errors_total=5, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_sidecar.relay_bytes_upstream_to_downstream").Int(); got != 7 {
		t.Fatalf("expected relay_bytes_upstream_to_downstream=7, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_sidecar.relay_frames_downstream_to_upstream").Int(); got != 8 {
		t.Fatalf("expected relay_frames_downstream_to_upstream=8, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_sidecar.relay_close_frames_total").Int(); got != 10 {
		t.Fatalf("expected relay_close_frames_total=10, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.rust_sidecar.relay_pong_frames_total").Int(); got != 12 {
		t.Fatalf("expected relay_pong_frames_total=12, got=%d body=%s", got, rec.Body.String())
	}

	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/openai-ws-runtime", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected second status 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	if hits := healthHits.Load(); hits != 1 {
		t.Fatalf("expected rust sidecar health endpoint to be cached, got hits=%d", hits)
	}
}
