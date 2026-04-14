package curlcffi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultRequestPath = "/request"
	maxBodyBytes       = 4 << 20
)

// Config controls how the curl_cffi sidecar client talks to the sidecar service.
type Config struct {
	BaseURL             string
	Impersonate         string
	TimeoutSeconds      int
	SessionReuseEnabled bool
	SessionTTLSeconds   int
}

// Request describes one HTTP request that should be executed by curl_cffi sidecar.
type Request struct {
	Method            string
	URL               string
	Headers           map[string]string
	Body              []byte
	ProxyURL          string
	Impersonate       string
	TimeoutSeconds    int
	SessionKey        string
	SessionTTLSeconds int
}

// Response is the normalized sidecar HTTP response.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

type Client struct {
	endpoint            string
	httpClient          *http.Client
	impersonate         string
	timeoutSeconds      int
	sessionReuseEnabled bool
	sessionTTLSeconds   int
}

// NewClient builds a sidecar client with normalized defaults.
func NewClient(cfg Config) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		return nil, fmt.Errorf("curl_cffi sidecar base_url is required")
	}
	if cfg.TimeoutSeconds < 0 {
		return nil, fmt.Errorf("curl_cffi sidecar timeout_seconds must be non-negative")
	}
	if cfg.SessionTTLSeconds < 0 {
		return nil, fmt.Errorf("curl_cffi sidecar session_ttl_seconds must be non-negative")
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid curl_cffi sidecar base_url: %w", err)
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + defaultRequestPath
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	requestTimeout := 90 * time.Second
	if cfg.TimeoutSeconds > 0 {
		requestTimeout = time.Duration(cfg.TimeoutSeconds+5) * time.Second
	}

	return &Client{
		endpoint:            baseURL.String(),
		httpClient:          &http.Client{Timeout: requestTimeout},
		impersonate:         strings.TrimSpace(cfg.Impersonate),
		timeoutSeconds:      cfg.TimeoutSeconds,
		sessionReuseEnabled: cfg.SessionReuseEnabled,
		sessionTTLSeconds:   cfg.SessionTTLSeconds,
	}, nil
}

func (c *Client) EnabledForSessionReuse() bool {
	return c != nil && c.sessionReuseEnabled
}

func (c *Client) DefaultSessionTTLSeconds() int {
	if c == nil {
		return 0
	}
	return c.sessionTTLSeconds
}

func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	if c == nil {
		return nil, fmt.Errorf("curl_cffi sidecar client is nil")
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("curl_cffi sidecar request url is required")
	}

	payload := sidecarRequest{
		Method:         method,
		URL:            strings.TrimSpace(req.URL),
		Headers:        normalizeHeaders(req.Headers),
		ProxyURL:       strings.TrimSpace(req.ProxyURL),
		Impersonate:    firstNonEmpty(strings.TrimSpace(req.Impersonate), c.impersonate),
		TimeoutSeconds: pickPositive(req.TimeoutSeconds, c.timeoutSeconds),
		SessionKey:     strings.TrimSpace(req.SessionKey),
	}
	if req.SessionTTLSeconds > 0 {
		payload.SessionTTLSeconds = req.SessionTTLSeconds
	} else if payload.SessionKey != "" && c.sessionTTLSeconds > 0 {
		payload.SessionTTLSeconds = c.sessionTTLSeconds
	}
	if len(req.Body) > 0 {
		payload.BodyBase64 = base64.StdEncoding.EncodeToString(req.Body)
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal curl_cffi sidecar request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(rawPayload))
	if err != nil {
		return nil, fmt.Errorf("build curl_cffi sidecar request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("curl_cffi sidecar request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	rawRespBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("curl_cffi sidecar status %d: %s", resp.StatusCode, strings.TrimSpace(string(rawRespBody)))
	}

	var sidecarResp sidecarResponse
	if err := json.Unmarshal(rawRespBody, &sidecarResp); err != nil {
		return nil, fmt.Errorf("decode curl_cffi sidecar response: %w", err)
	}

	statusCode := sidecarResp.StatusCode
	if statusCode == 0 {
		statusCode = sidecarResp.Status
	}
	if statusCode == 0 {
		return nil, fmt.Errorf("curl_cffi sidecar response missing status_code")
	}

	body := []byte(sidecarResp.Body)
	if strings.TrimSpace(sidecarResp.BodyBase64) != "" {
		decoded, decodeErr := base64.StdEncoding.DecodeString(sidecarResp.BodyBase64)
		if decodeErr != nil {
			return nil, fmt.Errorf("decode curl_cffi sidecar body_base64: %w", decodeErr)
		}
		body = decoded
	}

	return &Response{
		StatusCode: statusCode,
		Headers:    normalizeResponseHeaders(sidecarResp.Headers),
		Body:       body,
	}, nil
}

type sidecarRequest struct {
	Method            string            `json:"method"`
	URL               string            `json:"url"`
	Headers           map[string]string `json:"headers,omitempty"`
	BodyBase64        string            `json:"body_base64,omitempty"`
	ProxyURL          string            `json:"proxy_url,omitempty"`
	Impersonate       string            `json:"impersonate,omitempty"`
	TimeoutSeconds    int               `json:"timeout_seconds,omitempty"`
	SessionKey        string            `json:"session_key,omitempty"`
	SessionTTLSeconds int               `json:"session_ttl_seconds,omitempty"`
}

type sidecarResponse struct {
	StatusCode int            `json:"status_code,omitempty"`
	Status     int            `json:"status,omitempty"`
	Headers    map[string]any `json:"headers,omitempty"`
	Body       string         `json:"body,omitempty"`
	BodyBase64 string         `json:"body_base64,omitempty"`
}

func normalizeHeaders(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		out[trimmedKey] = strings.TrimSpace(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeResponseHeaders(in map[string]any) http.Header {
	if len(in) == 0 {
		return http.Header{}
	}
	headers := make(http.Header, len(in))
	for key, value := range in {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		switch typed := value.(type) {
		case string:
			headers.Set(trimmedKey, typed)
		case []any:
			for _, entry := range typed {
				if text, ok := entry.(string); ok && strings.TrimSpace(text) != "" {
					headers.Add(trimmedKey, text)
				}
			}
		}
	}
	return headers
}

func pickPositive(value int, fallback int) int {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
