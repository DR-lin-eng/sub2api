package service

import (
	"crypto/tls"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	openaiwsv2 "github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2"
	coderws "github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const openAIWSMessageReadLimitBytes int64 = 16 * 1024 * 1024
const (
	openAIWSProxyTransportMaxIdleConns        = 128
	openAIWSProxyTransportMaxIdleConnsPerHost = 64
	openAIWSProxyTransportIdleConnTimeout     = 90 * time.Second
	openAIWSProxyClientCacheMaxEntries        = 256
	openAIWSProxyClientCacheIdleTTL           = 15 * time.Minute
)

const openAIWSHTTPVersionHeader = "x-sub2api-ws-http-version"

type OpenAIWSTransportMetricsSnapshot struct {
	ProxyClientCacheHits   int64   `json:"proxy_client_cache_hits"`
	ProxyClientCacheMisses int64   `json:"proxy_client_cache_misses"`
	TransportReuseRatio    float64 `json:"transport_reuse_ratio"`
	HTTP1DialTotal         int64   `json:"http1_dial_total"`
	HTTP2DialTotal         int64   `json:"http2_dial_total"`
	FallbackToHTTP1Total   int64   `json:"fallback_to_http1_total"`
}

// openAIWSClientConn 抽象 WS 客户端连接，便于替换底层实现。
type openAIWSClientConn interface {
	WriteJSON(ctx context.Context, value any) error
	ReadMessage(ctx context.Context) ([]byte, error)
	Ping(ctx context.Context) error
	Close() error
}

// openAIWSClientDialer 抽象 WS 建连器。
type openAIWSClientDialer interface {
	Dial(ctx context.Context, wsURL string, headers http.Header, proxyURL string) (openAIWSClientConn, int, http.Header, error)
}

type openAIWSTransportMetricsDialer interface {
	SnapshotTransportMetrics() OpenAIWSTransportMetricsSnapshot
}

type openAIWSDialHTTPVersion string

const (
	openAIWSDialHTTPVersionAuto openAIWSDialHTTPVersion = "auto"
	openAIWSDialHTTPVersionH1   openAIWSDialHTTPVersion = "1.1"
	openAIWSDialHTTPVersionH2   openAIWSDialHTTPVersion = "2"
)

func newDefaultOpenAIWSClientDialer(cfg *config.Config) openAIWSClientDialer {
	return &coderOpenAIWSClientDialer{
		cfg:          cfg,
		proxyClients: make(map[string]*openAIWSProxyClientEntry),
	}
}

type coderOpenAIWSClientDialer struct {
	cfg          *config.Config
	proxyMu      sync.Mutex
	proxyClients map[string]*openAIWSProxyClientEntry
	proxyHits    atomic.Int64
	proxyMisses  atomic.Int64
	http1Dials   atomic.Int64
	http2Dials   atomic.Int64
	fallbackH1   atomic.Int64
}

type openAIWSProxyClientEntry struct {
	client           *http.Client
	lastUsedUnixNano int64
}

func (d *coderOpenAIWSClientDialer) dialHTTPVersion() openAIWSDialHTTPVersion {
	if d != nil && d.cfg != nil {
		switch strings.TrimSpace(strings.ToLower(d.cfg.Gateway.OpenAIWS.DialHTTPVersion)) {
		case string(openAIWSDialHTTPVersionH1):
			return openAIWSDialHTTPVersionH1
		case string(openAIWSDialHTTPVersionH2):
			return openAIWSDialHTTPVersionH2
		}
	}
	return openAIWSDialHTTPVersionAuto
}

func (d *coderOpenAIWSClientDialer) Dial(
	ctx context.Context,
	wsURL string,
	headers http.Header,
	proxyURL string,
) (openAIWSClientConn, int, http.Header, error) {
	targetURL := strings.TrimSpace(wsURL)
	if targetURL == "" {
		return nil, 0, nil, errors.New("ws url is empty")
	}
	versions := []openAIWSDialHTTPVersion{d.dialHTTPVersion()}
	if versions[0] == openAIWSDialHTTPVersionAuto {
		versions = []openAIWSDialHTTPVersion{openAIWSDialHTTPVersionH2, openAIWSDialHTTPVersionH1}
	}
	var lastStatus int
	var lastRespHeaders http.Header
	var lastErr error
	for idx, version := range versions {
		opts := &coderws.DialOptions{
			HTTPHeader:      cloneHeader(headers),
			CompressionMode: coderws.CompressionContextTakeover,
		}
		client, err := d.httpClientForWS(strings.TrimSpace(proxyURL), version)
		if err != nil {
			return nil, 0, nil, err
		}
		if client != nil {
			opts.HTTPClient = client
		}

		conn, resp, err := coderws.Dial(ctx, targetURL, opts)
		if err == nil {
			// coder/websocket 默认单消息读取上限为 32KB，Codex WS 事件（如 rate_limits/大 delta）
			// 可能超过该阈值，需显式提高上限，避免本地 read_fail(message too big)。
			conn.SetReadLimit(openAIWSMessageReadLimitBytes)
			respHeaders := http.Header(nil)
			if resp != nil {
				respHeaders = cloneHeader(resp.Header)
			}
			httpVersion := openAIWSHTTPVersionFromResponse(resp, version)
			annotateOpenAIWSHTTPVersionHeader(respHeaders, httpVersion)
			d.recordOpenAIWSDialHTTPVersion(httpVersion)
			if idx > 0 && version == openAIWSDialHTTPVersionH1 {
				d.fallbackH1.Add(1)
			}
			return &coderOpenAIWSClientConn{conn: conn}, 0, respHeaders, nil
		}
		lastStatus = 0
		lastRespHeaders = nil
		if resp != nil {
			lastStatus = resp.StatusCode
			lastRespHeaders = cloneHeader(resp.Header)
		}
		httpVersion := openAIWSHTTPVersionFromResponse(resp, version)
		annotateOpenAIWSHTTPVersionHeader(lastRespHeaders, httpVersion)
		lastErr = err
		if version != openAIWSDialHTTPVersionH2 || idx == len(versions)-1 || !shouldRetryOpenAIWSDialWithHTTP11(lastStatus, lastRespHeaders, err) {
			break
		}
	}
	return nil, lastStatus, lastRespHeaders, lastErr
}

func (d *coderOpenAIWSClientDialer) httpClientForWS(proxy string, version openAIWSDialHTTPVersion) (*http.Client, error) {
	if strings.TrimSpace(proxy) == "" {
		if version == openAIWSDialHTTPVersionH1 {
			return buildOpenAIWSHTTPClient(nil, version), nil
		}
		return nil, nil
	}
	return d.proxyHTTPClient(proxy, version)
}

func (d *coderOpenAIWSClientDialer) proxyHTTPClient(proxy string, version openAIWSDialHTTPVersion) (*http.Client, error) {
	if d == nil {
		return nil, errors.New("openai ws dialer is nil")
	}
	normalizedProxy := strings.TrimSpace(proxy)
	if normalizedProxy == "" {
		return nil, errors.New("proxy url is empty")
	}
	parsedProxyURL, err := url.Parse(normalizedProxy)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy url: %w", err)
	}
	now := time.Now().UnixNano()
	cacheKey := normalizedProxy + "|" + string(version)

	d.proxyMu.Lock()
	defer d.proxyMu.Unlock()
	if entry, ok := d.proxyClients[cacheKey]; ok && entry != nil && entry.client != nil {
		entry.lastUsedUnixNano = now
		d.proxyHits.Add(1)
		return entry.client, nil
	}
	d.cleanupProxyClientsLocked(now)
	client := buildOpenAIWSHTTPClient(parsedProxyURL, version)
	d.proxyClients[cacheKey] = &openAIWSProxyClientEntry{
		client:           client,
		lastUsedUnixNano: now,
	}
	d.ensureProxyClientCapacityLocked()
	d.proxyMisses.Add(1)
	return client, nil
}

func annotateOpenAIWSHTTPVersionHeader(headers http.Header, version string) {
	if headers == nil {
		return
	}
	version = strings.TrimSpace(strings.ToLower(version))
	if version == "" {
		return
	}
	headers.Set(openAIWSHTTPVersionHeader, version)
}

func openAIWSHTTPVersionFromResponse(resp *http.Response, fallback openAIWSDialHTTPVersion) string {
	if resp != nil {
		switch resp.ProtoMajor {
		case 1:
			return "h1"
		case 2:
			return "h2"
		}
	}
	switch fallback {
	case openAIWSDialHTTPVersionH1:
		return "h1"
	case openAIWSDialHTTPVersionH2:
		return "h2"
	default:
		return ""
	}
}

func (d *coderOpenAIWSClientDialer) recordOpenAIWSDialHTTPVersion(version string) {
	switch strings.TrimSpace(strings.ToLower(version)) {
	case "h1":
		d.http1Dials.Add(1)
	case "h2":
		d.http2Dials.Add(1)
	}
}

func buildOpenAIWSHTTPClient(proxyURL *url.URL, version openAIWSDialHTTPVersion) *http.Client {
	transport := &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
		MaxIdleConns:        openAIWSProxyTransportMaxIdleConns,
		MaxIdleConnsPerHost: openAIWSProxyTransportMaxIdleConnsPerHost,
		IdleConnTimeout:     openAIWSProxyTransportIdleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	switch version {
	case openAIWSDialHTTPVersionH1:
		transport.ForceAttemptHTTP2 = false
		transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	default:
		transport.ForceAttemptHTTP2 = true
	}
	return &http.Client{Transport: transport}
}

func shouldRetryOpenAIWSDialWithHTTP11(status int, headers http.Header, err error) bool {
	switch status {
	case http.StatusUpgradeRequired, http.StatusBadGateway, http.StatusBadRequest, http.StatusServiceUnavailable:
		return true
	}
	class := classifyOpenAIWSDialError(err)
	switch class {
	case "handshake_not_finished", "bad_handshake":
		return true
	}
	server := strings.ToLower(strings.TrimSpace(headers.Get("server")))
	via := strings.ToLower(strings.TrimSpace(headers.Get("via")))
	if strings.Contains(server, "cloudflare") || via != "" || strings.TrimSpace(headers.Get("cf-ray")) != "" {
		return true
	}
	return false
}

func (d *coderOpenAIWSClientDialer) cleanupProxyClientsLocked(nowUnixNano int64) {
	if d == nil || len(d.proxyClients) == 0 {
		return
	}
	idleTTL := openAIWSProxyClientCacheIdleTTL
	if idleTTL <= 0 {
		return
	}
	now := time.Unix(0, nowUnixNano)
	for key, entry := range d.proxyClients {
		if entry == nil || entry.client == nil {
			delete(d.proxyClients, key)
			continue
		}
		lastUsed := time.Unix(0, entry.lastUsedUnixNano)
		if now.Sub(lastUsed) > idleTTL {
			closeOpenAIWSProxyClient(entry.client)
			delete(d.proxyClients, key)
		}
	}
}

func (d *coderOpenAIWSClientDialer) ensureProxyClientCapacityLocked() {
	if d == nil {
		return
	}
	maxEntries := openAIWSProxyClientCacheMaxEntries
	if maxEntries <= 0 {
		return
	}
	for len(d.proxyClients) > maxEntries {
		var oldestKey string
		var oldestLastUsed int64
		hasOldest := false
		for key, entry := range d.proxyClients {
			lastUsed := int64(0)
			if entry != nil {
				lastUsed = entry.lastUsedUnixNano
			}
			if !hasOldest || lastUsed < oldestLastUsed {
				hasOldest = true
				oldestKey = key
				oldestLastUsed = lastUsed
			}
		}
		if !hasOldest {
			return
		}
		if entry := d.proxyClients[oldestKey]; entry != nil {
			closeOpenAIWSProxyClient(entry.client)
		}
		delete(d.proxyClients, oldestKey)
	}
}

func closeOpenAIWSProxyClient(client *http.Client) {
	if client == nil || client.Transport == nil {
		return
	}
	if transport, ok := client.Transport.(*http.Transport); ok && transport != nil {
		transport.CloseIdleConnections()
	}
}

func (d *coderOpenAIWSClientDialer) SnapshotTransportMetrics() OpenAIWSTransportMetricsSnapshot {
	if d == nil {
		return OpenAIWSTransportMetricsSnapshot{}
	}
	hits := d.proxyHits.Load()
	misses := d.proxyMisses.Load()
	total := hits + misses
	reuseRatio := 0.0
	if total > 0 {
		reuseRatio = float64(hits) / float64(total)
	}
	return OpenAIWSTransportMetricsSnapshot{
		ProxyClientCacheHits:   hits,
		ProxyClientCacheMisses: misses,
		TransportReuseRatio:    reuseRatio,
		HTTP1DialTotal:         d.http1Dials.Load(),
		HTTP2DialTotal:         d.http2Dials.Load(),
		FallbackToHTTP1Total:   d.fallbackH1.Load(),
	}
}

type coderOpenAIWSClientConn struct {
	conn *coderws.Conn
}

var _ openaiwsv2.FrameConn = (*coderOpenAIWSClientConn)(nil)

func (c *coderOpenAIWSClientConn) WriteJSON(ctx context.Context, value any) error {
	if c == nil || c.conn == nil {
		return errOpenAIWSConnClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return wsjson.Write(ctx, c.conn, value)
}

func (c *coderOpenAIWSClientConn) ReadMessage(ctx context.Context) ([]byte, error) {
	if c == nil || c.conn == nil {
		return nil, errOpenAIWSConnClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}

	msgType, payload, err := c.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	switch msgType {
	case coderws.MessageText, coderws.MessageBinary:
		return payload, nil
	default:
		return nil, errOpenAIWSConnClosed
	}
}

func (c *coderOpenAIWSClientConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	if c == nil || c.conn == nil {
		return coderws.MessageText, nil, errOpenAIWSConnClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	msgType, payload, err := c.conn.Read(ctx)
	if err != nil {
		return coderws.MessageText, nil, err
	}
	return msgType, payload, nil
}

func (c *coderOpenAIWSClientConn) WriteFrame(ctx context.Context, msgType coderws.MessageType, payload []byte) error {
	if c == nil || c.conn == nil {
		return errOpenAIWSConnClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.conn.Write(ctx, msgType, payload)
}

func (c *coderOpenAIWSClientConn) Ping(ctx context.Context) error {
	if c == nil || c.conn == nil {
		return errOpenAIWSConnClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.conn.Ping(ctx)
}

func (c *coderOpenAIWSClientConn) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	// Close 为幂等，忽略重复关闭错误。
	_ = c.conn.Close(coderws.StatusNormalClosure, "")
	_ = c.conn.CloseNow()
	return nil
}
