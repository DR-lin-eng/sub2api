package sidecar

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const defaultBaseURL = "http://rust-sidecar"

type Client struct {
	socketPath string
	baseURL    string
	httpClient *http.Client
}

type HealthResponse struct {
	Status            string `json:"status"`
	Service           string `json:"service"`
	Version           string `json:"version"`
	ActiveConnections int64  `json:"active_connections,omitempty"`
	TotalConnections  int64  `json:"total_connections,omitempty"`
	ActiveUpgrades    int64  `json:"active_upgrades,omitempty"`
	TotalUpgrades     int64  `json:"total_upgrades,omitempty"`
	TotalRequests      int64 `json:"total_requests,omitempty"`
	TotalRequestErrors int64 `json:"total_request_errors,omitempty"`
	UpstreamUnavailableTotal int64 `json:"upstream_unavailable_total,omitempty"`
	UpstreamHandshakeFailedTotal int64 `json:"upstream_handshake_failed_total,omitempty"`
	UpstreamRequestFailedTotal int64 `json:"upstream_request_failed_total,omitempty"`
	UpgradeErrorsTotal int64 `json:"upgrade_errors_total,omitempty"`
	RelayBytesDownstreamToUpstream int64 `json:"relay_bytes_downstream_to_upstream,omitempty"`
	RelayBytesUpstreamToDownstream int64 `json:"relay_bytes_upstream_to_downstream,omitempty"`
	RelayFramesDownstreamToUpstream int64 `json:"relay_frames_downstream_to_upstream,omitempty"`
	RelayFramesUpstreamToDownstream int64 `json:"relay_frames_upstream_to_downstream,omitempty"`
	RelayCloseFramesTotal int64 `json:"relay_close_frames_total,omitempty"`
	RelayPingFramesTotal int64 `json:"relay_ping_frames_total,omitempty"`
	RelayPongFramesTotal int64 `json:"relay_pong_frames_total,omitempty"`
}

func NewClient(cfg config.RustSidecarConfig) (*Client, error) {
	socketPath := strings.TrimSpace(cfg.SocketPath)
	if socketPath == "" {
		return nil, fmt.Errorf("rust sidecar socket path is required")
	}
	timeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
		DisableCompression: false,
	}

	return &Client{
		socketPath: socketPath,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Transport: transport, Timeout: timeout},
	}, nil
}

func (c *Client) SocketPath() string {
	if c == nil {
		return ""
	}
	return c.socketPath
}

func (c *Client) EndpointURL(relativePath string) string {
	if c == nil {
		return ""
	}
	cleanPath := "/" + strings.TrimPrefix(path.Clean("/"+relativePath), "/")
	return c.baseURL + cleanPath
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("rust sidecar client is not initialized")
	}
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	cloned := req.Clone(req.Context())
	if cloned.URL == nil {
		return nil, fmt.Errorf("request url is nil")
	}
	if cloned.URL.Scheme == "" {
		cloned.URL.Scheme = "http"
	}
	if cloned.URL.Host == "" {
		cloned.URL.Host = "rust-sidecar"
	}
	return c.httpClient.Do(cloned)
}

func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("rust sidecar client is nil")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.EndpointURL("/healthz"), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected rust sidecar status: %d", resp.StatusCode)
	}
	var payload HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
