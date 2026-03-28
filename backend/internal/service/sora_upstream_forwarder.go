package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

// forwardToUpstream 将请求 HTTP 透传到上游 Sora 服务（用于 apikey 类型账号）。
// 上游地址为 account.GetBaseURL() + "/sora/v1/chat/completions"，
// 使用 account.GetCredential("api_key") 作为 Bearer Token。
// 支持流式和非流式响应的直接透传。
func (s *SoraGatewayService) forwardToUpstream(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	clientStream bool,
	startTime time.Time,
) (*ForwardResult, error) {
	return s.forwardToUpstreamContext(ctx, gatewayctx.FromGin(c), account, body, clientStream, startTime)
}

func (s *SoraGatewayService) forwardToUpstreamContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	clientStream bool,
	startTime time.Time,
) (*ForwardResult, error) {
	apiKey := account.GetCredential("api_key")
	if apiKey == "" {
		s.writeSoraErrorContext(c, http.StatusBadGateway, "upstream_error", "Sora apikey account missing api_key credential", clientStream)
		return nil, fmt.Errorf("sora apikey account %d missing api_key", account.ID)
	}

	baseURL := account.GetBaseURL()
	if baseURL == "" {
		s.writeSoraErrorContext(c, http.StatusBadGateway, "upstream_error", "Sora apikey account missing base_url", clientStream)
		return nil, fmt.Errorf("sora apikey account %d missing base_url", account.ID)
	}
	// 校验 scheme 合法性（仅允许 http/https）
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		s.writeSoraErrorContext(c, http.StatusBadGateway, "upstream_error", "Sora apikey base_url must start with http:// or https://", clientStream)
		return nil, fmt.Errorf("sora apikey account %d invalid base_url scheme: %s", account.ID, baseURL)
	}
	upstreamURL := strings.TrimRight(baseURL, "/") + "/sora/v1/chat/completions"

	// 构建上游请求
	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		s.writeSoraErrorContext(c, http.StatusInternalServerError, "api_error", "Failed to create upstream request", clientStream)
		return nil, fmt.Errorf("create upstream request: %w", err)
	}

	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Authorization", "Bearer "+apiKey)

	// 透传客户端的部分请求头
	for _, header := range []string{"Accept", "Accept-Encoding"} {
		if v := strings.TrimSpace(c.HeaderValue(header)); v != "" {
			upstreamReq.Header.Set(header, v)
		}
	}

	logger.LegacyPrintf("service.sora", "[ForwardUpstream] account=%d url=%s", account.ID, upstreamURL)

	// 获取代理 URL
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// 发送请求
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		s.writeSoraErrorContext(c, http.StatusBadGateway, "upstream_error", "Failed to connect to upstream Sora service", clientStream)
		return nil, &UpstreamFailoverError{
			StatusCode: http.StatusBadGateway,
		}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// 错误响应处理
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

		if s.shouldFailoverUpstreamError(resp.StatusCode) {
			return nil, &UpstreamFailoverError{
				StatusCode:      resp.StatusCode,
				ResponseBody:    respBody,
				ResponseHeaders: resp.Header.Clone(),
			}
		}

		// 非转移错误，直接透传给客户端
		c.SetStatus(resp.StatusCode)
		for key, values := range resp.Header {
			for _, v := range values {
				c.Header().Add(key, v)
			}
		}
		if _, err := c.WriteBytes(0, respBody); err != nil {
			return nil, fmt.Errorf("write upstream error response: %w", err)
		}
		return nil, fmt.Errorf("upstream error: %d", resp.StatusCode)
	}

	// 成功响应 — 直接透传
	c.SetStatus(resp.StatusCode)
	for key, values := range resp.Header {
		lower := strings.ToLower(key)
		// 透传内容相关头部
		if lower == "content-type" || lower == "transfer-encoding" ||
			lower == "cache-control" || lower == "x-request-id" {
			for _, v := range values {
				c.Header().Add(key, v)
			}
		}
	}

	// 流式复制响应体
	if clientStream {
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				if _, err := c.WriteBytes(0, buf[:n]); err != nil {
					return nil, fmt.Errorf("stream upstream response write: %w", err)
				}
				if err := c.Flush(); err != nil {
					return nil, fmt.Errorf("stream upstream response flush: %w", err)
				}
			}
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				return nil, fmt.Errorf("stream upstream response read: %w", readErr)
			}
		}
	} else {
		buf := make([]byte, 32*1024)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				if _, err := c.WriteBytes(0, buf[:n]); err != nil {
					return nil, fmt.Errorf("copy upstream response: %w", err)
				}
			}
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				return nil, fmt.Errorf("copy upstream response: %w", readErr)
			}
		}
	}

	duration := time.Since(startTime)
	return &ForwardResult{
		RequestID: resp.Header.Get("x-request-id"),
		Model:     "", // 由调用方填充
		Stream:    clientStream,
		Duration:  duration,
	}, nil
}
