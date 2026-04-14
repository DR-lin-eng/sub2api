package curlcffi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_ValidateBaseURL(t *testing.T) {
	_, err := NewClient(Config{})
	if err == nil {
		t.Fatal("NewClient() expected error when base_url is empty")
	}
}

func TestClient_Do_Success(t *testing.T) {
	type capturedPayload struct {
		Method            string            `json:"method"`
		URL               string            `json:"url"`
		Headers           map[string]string `json:"headers"`
		ProxyURL          string            `json:"proxy_url"`
		Impersonate       string            `json:"impersonate"`
		TimeoutSeconds    int               `json:"timeout_seconds"`
		SessionKey        string            `json:"session_key"`
		SessionTTLSeconds int               `json:"session_ttl_seconds"`
	}

	var captured capturedPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/request" {
			t.Fatalf("path = %s, want /request", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":200,"headers":{"content-type":"application/json"},"body":"{\"ok\":true}"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:             server.URL,
		Impersonate:         "chrome131",
		TimeoutSeconds:      60,
		SessionReuseEnabled: true,
		SessionTTLSeconds:   3600,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.Do(context.Background(), Request{
		Method:         http.MethodGet,
		URL:            "https://chatgpt.com/api/auth/session",
		Headers:        map[string]string{"Cookie": "a=b"},
		ProxyURL:       "http://127.0.0.1:1080",
		SessionKey:     "key-1",
		Impersonate:    "",
		TimeoutSeconds: 0,
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status_code = %d, want 200", resp.StatusCode)
	}
	if string(resp.Body) != "{\"ok\":true}" {
		t.Fatalf("body = %q, want %q", string(resp.Body), "{\"ok\":true}")
	}
	if captured.Method != http.MethodGet {
		t.Fatalf("payload.method = %q, want GET", captured.Method)
	}
	if captured.URL != "https://chatgpt.com/api/auth/session" {
		t.Fatalf("payload.url = %q", captured.URL)
	}
	if captured.Impersonate != "chrome131" {
		t.Fatalf("payload.impersonate = %q, want chrome131", captured.Impersonate)
	}
	if captured.TimeoutSeconds != 60 {
		t.Fatalf("payload.timeout_seconds = %d, want 60", captured.TimeoutSeconds)
	}
	if captured.SessionKey != "key-1" {
		t.Fatalf("payload.session_key = %q, want key-1", captured.SessionKey)
	}
	if captured.SessionTTLSeconds != 3600 {
		t.Fatalf("payload.session_ttl_seconds = %d, want 3600", captured.SessionTTLSeconds)
	}
}

func TestClient_Do_ParsesBodyBase64(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":201,"headers":{"x-a":["1","2"]},"body_base64":"` + encoded + `"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.Do(context.Background(), Request{
		Method: http.MethodPost,
		URL:    "https://example.com/demo",
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status_code = %d, want 201", resp.StatusCode)
	}
	if string(resp.Body) != "hello" {
		t.Fatalf("body = %q, want hello", string(resp.Body))
	}
	if got := resp.Headers.Values("x-a"); len(got) != 2 {
		t.Fatalf("header x-a values len = %d, want 2", len(got))
	}
}
