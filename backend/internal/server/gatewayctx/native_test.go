package gatewayctx

import (
	"net/http"
	"testing"
)

func TestNativeGatewayContextClientIP_IgnoresForwardedHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.RemoteAddr = "203.0.113.88:54321"
	req.Header.Set("CF-Connecting-IP", "203.0.113.7")
	req.Header.Set("X-Real-IP", "198.51.100.9")
	req.Header.Set("X-Forwarded-For", "198.51.100.10, 10.0.0.8")

	ctx := NewNative(req, nil, nil, req.RemoteAddr)
	if got := ctx.ClientIP(); got != "203.0.113.88" {
		t.Fatalf("ClientIP() = %q, want %q", got, "203.0.113.88")
	}
}

func TestNativeGatewayContextClientIP_UsesInjectedTrustedClientIP(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.RemoteAddr = "10.0.0.8:54321"
	req.Header.Set("X-Forwarded-For", "10.1.1.1, 198.51.100.77, 10.0.0.8")

	ctx := NewNative(req, nil, nil, "198.51.100.77")
	if got := ctx.ClientIP(); got != "198.51.100.77" {
		t.Fatalf("ClientIP() = %q, want %q", got, "198.51.100.77")
	}
}

func TestNativeGatewayContextClientIP_FallsBackToRemoteAddr(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.RemoteAddr = "203.0.113.88:54321"

	ctx := NewNative(req, nil, nil, req.RemoteAddr)
	if got := ctx.ClientIP(); got != "203.0.113.88" {
		t.Fatalf("ClientIP() = %q, want %q", got, "203.0.113.88")
	}
}
