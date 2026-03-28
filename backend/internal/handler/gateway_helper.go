package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// claudeCodeValidator is a singleton validator for Claude Code client detection
var claudeCodeValidator = service.NewClaudeCodeValidator()

const claudeCodeParsedRequestContextKey = "claude_code_parsed_request"

// SetClaudeCodeClientContext 检查请求是否来自 Claude Code 客户端，并设置到 context 中
// 返回更新后的 context
func SetClaudeCodeClientContext(c *gin.Context, body []byte, parsedReq *service.ParsedRequest) {
	SetClaudeCodeClientContextContext(gatewayctx.FromGin(c), body, parsedReq)
}

func SetClaudeCodeClientContextContext(c gatewayctx.GatewayContext, body []byte, parsedReq *service.ParsedRequest) {
	if c == nil || c.Request() == nil {
		return
	}
	if parsedReq != nil {
		c.SetValue(claudeCodeParsedRequestContextKey, parsedReq)
	}

	ua := c.HeaderValue("User-Agent")
	// Fast path：非 Claude CLI UA 直接判定 false，避免热路径二次 JSON 反序列化。
	if !claudeCodeValidator.ValidateUserAgent(ua) {
		ctx := service.SetClaudeCodeClient(c.Request().Context(), false)
		c.SetRequest(c.Request().WithContext(ctx))
		return
	}

	isClaudeCode := false
	if !strings.Contains(c.Path(), "messages") {
		// 与 Validate 行为一致：非 messages 路径 UA 命中即可视为 Claude Code 客户端。
		isClaudeCode = true
	} else {
		// 仅在确认为 Claude CLI 且 messages 路径时再做 body 解析。
		bodyMap := claudeCodeBodyMapFromParsedRequest(parsedReq)
		if bodyMap == nil {
			bodyMap = claudeCodeBodyMapFromGatewayContextCache(c)
		}
		if bodyMap == nil && len(body) > 0 {
			_ = json.Unmarshal(body, &bodyMap)
		}
		isClaudeCode = claudeCodeValidator.Validate(c.Request(), bodyMap)
	}

	// 更新 request context
	ctx := service.SetClaudeCodeClient(c.Request().Context(), isClaudeCode)

	// 仅在确认为 Claude Code 客户端时提取版本号写入 context
	if isClaudeCode {
		if version := claudeCodeValidator.ExtractVersion(ua); version != "" {
			ctx = service.SetClaudeCodeVersion(ctx, version)
		}
	}

	c.SetRequest(c.Request().WithContext(ctx))
}

func claudeCodeBodyMapFromParsedRequest(parsedReq *service.ParsedRequest) map[string]any {
	if parsedReq == nil {
		return nil
	}
	bodyMap := map[string]any{
		"model": parsedReq.Model,
	}
	if parsedReq.System != nil || parsedReq.HasSystem {
		bodyMap["system"] = parsedReq.System
	}
	if parsedReq.MetadataUserID != "" {
		bodyMap["metadata"] = map[string]any{"user_id": parsedReq.MetadataUserID}
	}
	return bodyMap
}

func claudeCodeBodyMapFromContextCache(c *gin.Context) map[string]any {
	return claudeCodeBodyMapFromGatewayContextCache(gatewayctx.FromGin(c))
}

func claudeCodeBodyMapFromGatewayContextCache(c gatewayctx.GatewayContext) map[string]any {
	if c == nil {
		return nil
	}
	if cached, ok := c.Value(service.OpenAIParsedRequestBodyKey); ok {
		if bodyMap, ok := cached.(map[string]any); ok {
			return bodyMap
		}
	}
	if cached, ok := c.Value(claudeCodeParsedRequestContextKey); ok {
		switch v := cached.(type) {
		case *service.ParsedRequest:
			return claudeCodeBodyMapFromParsedRequest(v)
		case service.ParsedRequest:
			return claudeCodeBodyMapFromParsedRequest(&v)
		}
	}
	return nil
}

func buildGatewaySessionContext(c *gin.Context, apiKeyID int64) *service.SessionContext {
	return buildGatewaySessionContextContext(gatewayctx.FromGin(c), apiKeyID)
}

func buildGatewaySessionContextContext(c gatewayctx.GatewayContext, apiKeyID int64) *service.SessionContext {
	if c == nil {
		return &service.SessionContext{APIKeyID: apiKeyID}
	}
	return &service.SessionContext{
		ClientIP:             strings.TrimSpace(c.ClientIP()),
		UserAgent:            c.HeaderValue("User-Agent"),
		APIKeyID:             apiKeyID,
		StableSessionID:      strings.TrimSpace(c.HeaderValue("session_id")),
		StableConversationID: strings.TrimSpace(c.HeaderValue("conversation_id")),
	}
}

// 并发槽位等待相关常量
//
// 性能优化说明：
// 原实现使用固定间隔（100ms）轮询并发槽位，存在以下问题：
// 1. 高并发时频繁轮询增加 Redis 压力
// 2. 固定间隔可能导致多个请求同时重试（惊群效应）
//
// 新实现使用指数退避 + 抖动算法：
// 1. 初始退避 100ms，每次乘以 1.5，最大 2s
// 2. 添加 ±20% 的随机抖动，分散重试时间点
// 3. 减少 Redis 压力，避免惊群效应
const (
	// maxConcurrencyWait 等待并发槽位的最大时间
	maxConcurrencyWait = 30 * time.Second
	// defaultPingInterval 流式响应等待时发送 ping 的默认间隔
	defaultPingInterval = 10 * time.Second
	// initialBackoff 初始退避时间
	initialBackoff = 100 * time.Millisecond
	// backoffMultiplier 退避时间乘数（指数退避）
	backoffMultiplier = 1.5
	// maxBackoff 最大退避时间
	maxBackoff = 2 * time.Second
	// fairWaitQueuePollBackoff 非队首 ticket 的轮询间隔下限，避免所有等待请求频繁抢 Redis
	fairWaitQueuePollBackoff = 250 * time.Millisecond
)

// SSEPingFormat defines the format of SSE ping events for different platforms
type SSEPingFormat string

const (
	// SSEPingFormatClaude is the Claude/Anthropic SSE ping format
	SSEPingFormatClaude SSEPingFormat = "data: {\"type\": \"ping\"}\n\n"
	// SSEPingFormatNone indicates no ping should be sent (e.g., OpenAI has no ping spec)
	SSEPingFormatNone SSEPingFormat = ""
	// SSEPingFormatComment is an SSE comment ping for OpenAI/Codex CLI clients
	SSEPingFormatComment SSEPingFormat = ":\n\n"
)

// ConcurrencyError represents a concurrency limit error with context
type ConcurrencyError struct {
	SlotType  string
	IsTimeout bool
}

func (e *ConcurrencyError) Error() string {
	if e.IsTimeout {
		return fmt.Sprintf("timeout waiting for %s concurrency slot", e.SlotType)
	}
	return fmt.Sprintf("%s concurrency limit reached", e.SlotType)
}

// ConcurrencyHelper provides common concurrency slot management for gateway handlers
type ConcurrencyHelper struct {
	concurrencyService *service.ConcurrencyService
	pingFormat         SSEPingFormat
	pingInterval       time.Duration
}

// NewConcurrencyHelper creates a new ConcurrencyHelper
func NewConcurrencyHelper(concurrencyService *service.ConcurrencyService, pingFormat SSEPingFormat, pingInterval time.Duration) *ConcurrencyHelper {
	if pingInterval <= 0 {
		pingInterval = defaultPingInterval
	}
	return &ConcurrencyHelper{
		concurrencyService: concurrencyService,
		pingFormat:         pingFormat,
		pingInterval:       pingInterval,
	}
}

func writeConcurrencyPing(ctx gatewayctx.GatewayContext, pingFormat SSEPingFormat, streamStarted *bool) error {
	if ctx == nil {
		return fmt.Errorf("gateway context is nil")
	}
	if streamStarted == nil {
		return fmt.Errorf("streamStarted is nil")
	}
	if pingFormat == "" {
		return nil
	}

	if !*streamStarted {
		ctx.SetHeader("Content-Type", "text/event-stream")
		ctx.SetHeader("Cache-Control", "no-cache, no-transform")
		ctx.SetHeader("Connection", "keep-alive")
		ctx.SetHeader("X-Accel-Buffering", "no")
		ctx.Header().Del("Content-Encoding")
		ctx.Header().Del("Content-Length")
		ctx.Header().Del("Transfer-Encoding")
		*streamStarted = true
	}

	if pingFormat == SSEPingFormatComment {
		return ctx.WriteSSEComment("")
	}
	if _, err := ctx.WriteBytes(http.StatusOK, []byte(string(pingFormat))); err != nil {
		return err
	}
	return ctx.Flush()
}

// wrapReleaseOnDone ensures release runs at most once and still triggers on context cancellation.
// 用于避免客户端断开或上游超时导致的并发槽位泄漏。
// 优化：基于 context.AfterFunc 注册回调，避免每请求额外守护 goroutine。
func wrapReleaseOnDone(ctx context.Context, releaseFunc func()) func() {
	if releaseFunc == nil {
		return nil
	}
	var once sync.Once
	var stop func() bool

	release := func() {
		once.Do(func() {
			if stop != nil {
				_ = stop()
			}
			releaseFunc()
		})
	}

	stop = context.AfterFunc(ctx, release)

	return release
}

// IncrementWaitCount increments the wait count for a user
func (h *ConcurrencyHelper) IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error) {
	return h.concurrencyService.IncrementWaitCount(ctx, userID, maxWait)
}

// DecrementWaitCount decrements the wait count for a user
func (h *ConcurrencyHelper) DecrementWaitCount(ctx context.Context, userID int64) {
	h.concurrencyService.DecrementWaitCount(ctx, userID)
}

// IncrementAccountWaitCount increments the wait count for an account
func (h *ConcurrencyHelper) IncrementAccountWaitCount(ctx context.Context, accountID int64, maxWait int) (bool, error) {
	return h.concurrencyService.IncrementAccountWaitCount(ctx, accountID, maxWait)
}

// DecrementAccountWaitCount decrements the wait count for an account
func (h *ConcurrencyHelper) DecrementAccountWaitCount(ctx context.Context, accountID int64) {
	h.concurrencyService.DecrementAccountWaitCount(ctx, accountID)
}

// TryAcquireUserSlot 尝试立即获取用户并发槽位。
// 返回值: (releaseFunc, acquired, error)
func (h *ConcurrencyHelper) TryAcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int) (func(), bool, error) {
	result, err := h.concurrencyService.AcquireUserSlot(ctx, userID, maxConcurrency)
	if err != nil {
		return nil, false, err
	}
	if !result.Acquired {
		return nil, false, nil
	}
	return result.ReleaseFunc, true, nil
}

// TryAcquireAccountSlot 尝试立即获取账号并发槽位。
// 返回值: (releaseFunc, acquired, error)
func (h *ConcurrencyHelper) TryAcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int) (func(), bool, error) {
	result, err := h.concurrencyService.AcquireAccountSlot(ctx, accountID, maxConcurrency)
	if err != nil {
		return nil, false, err
	}
	if !result.Acquired {
		return nil, false, nil
	}
	return result.ReleaseFunc, true, nil
}

// AcquireUserSlotWithWait acquires a user concurrency slot, waiting if necessary.
// For streaming requests, sends ping events during the wait.
// streamStarted is updated if streaming response has begun.
func (h *ConcurrencyHelper) AcquireUserSlotWithWait(c *gin.Context, userID int64, maxConcurrency int, isStream bool, streamStarted *bool) (func(), error) {
	return h.AcquireUserSlotWithWaitContext(gatewayctx.FromGin(c), userID, maxConcurrency, isStream, streamStarted)
}

// AcquireAccountSlotWithWait acquires an account concurrency slot, waiting if necessary.
// For streaming requests, sends ping events during the wait.
// streamStarted is updated if streaming response has begun.
func (h *ConcurrencyHelper) AcquireAccountSlotWithWait(c *gin.Context, accountID int64, maxConcurrency int, isStream bool, streamStarted *bool) (func(), error) {
	return h.AcquireAccountSlotWithWaitContext(gatewayctx.FromGin(c), accountID, maxConcurrency, isStream, streamStarted)
}

// waitForSlotWithPing waits for a concurrency slot, sending ping events for streaming requests.
// streamStarted pointer is updated when streaming begins (for proper error handling by caller).
func (h *ConcurrencyHelper) waitForSlotWithPing(c *gin.Context, slotType string, id int64, maxConcurrency int, isStream bool, streamStarted *bool) (func(), error) {
	return h.waitForSlotWithPingTimeoutContext(gatewayctx.FromGin(c), slotType, id, maxConcurrency, maxConcurrencyWait, isStream, streamStarted, false)
}

// waitForSlotWithPingTimeout waits for a concurrency slot with a custom timeout.
func (h *ConcurrencyHelper) waitForSlotWithPingTimeout(c *gin.Context, slotType string, id int64, maxConcurrency int, timeout time.Duration, isStream bool, streamStarted *bool, tryImmediate bool) (func(), error) {
	return h.waitForSlotWithPingTimeoutContext(gatewayctx.FromGin(c), slotType, id, maxConcurrency, timeout, isStream, streamStarted, tryImmediate)
}

func (h *ConcurrencyHelper) AcquireUserSlotWithWaitContext(ctx gatewayctx.GatewayContext, userID int64, maxConcurrency int, isStream bool, streamStarted *bool) (func(), error) {
	if ctx == nil {
		return nil, fmt.Errorf("gateway context is nil")
	}

	releaseFunc, acquired, err := h.TryAcquireUserSlot(ctx.Context(), userID, maxConcurrency)
	if err != nil {
		return nil, err
	}
	if acquired {
		return releaseFunc, nil
	}
	return h.waitForSlotWithPingContext(ctx, "user", userID, maxConcurrency, isStream, streamStarted)
}

func (h *ConcurrencyHelper) AcquireAccountSlotWithWaitContext(ctx gatewayctx.GatewayContext, accountID int64, maxConcurrency int, isStream bool, streamStarted *bool) (func(), error) {
	if ctx == nil {
		return nil, fmt.Errorf("gateway context is nil")
	}

	releaseFunc, acquired, err := h.TryAcquireAccountSlot(ctx.Context(), accountID, maxConcurrency)
	if err != nil {
		return nil, err
	}
	if acquired {
		return releaseFunc, nil
	}
	return h.waitForSlotWithPingContext(ctx, "account", accountID, maxConcurrency, isStream, streamStarted)
}

func (h *ConcurrencyHelper) waitForSlotWithPingContext(ctx gatewayctx.GatewayContext, slotType string, id int64, maxConcurrency int, isStream bool, streamStarted *bool) (func(), error) {
	return h.waitForSlotWithPingTimeoutContext(ctx, slotType, id, maxConcurrency, maxConcurrencyWait, isStream, streamStarted, false)
}

func (h *ConcurrencyHelper) waitForSlotWithPingTimeoutContext(gctx gatewayctx.GatewayContext, slotType string, id int64, maxConcurrency int, timeout time.Duration, isStream bool, streamStarted *bool, tryImmediate bool) (func(), error) {
	if gctx == nil {
		return nil, fmt.Errorf("gateway context is nil")
	}
	ctx, cancel := context.WithTimeout(gctx.Context(), timeout)
	defer cancel()

	acquireSlot := func() (*service.AcquireResult, error) {
		if slotType == "user" {
			return h.concurrencyService.AcquireUserSlot(ctx, id, maxConcurrency)
		}
		return h.concurrencyService.AcquireAccountSlot(ctx, id, maxConcurrency)
	}

	if tryImmediate {
		result, err := acquireSlot()
		if err != nil {
			return nil, err
		}
		if result.Acquired {
			return result.ReleaseFunc, nil
		}
	}

	fairWaitEnabled := h.concurrencyService != nil && h.concurrencyService.FairWaitQueueEnabled()
	ticketID := ""
	if fairWaitEnabled {
		ticketID = service.NewConcurrencyTicketID()
		var ticketErr error
		switch slotType {
		case "user":
			ticketErr = h.concurrencyService.EnqueueUserWaitTicket(ctx, id, ticketID)
		default:
			ticketErr = h.concurrencyService.EnqueueAccountWaitTicket(ctx, id, ticketID)
		}
		if ticketErr != nil {
			ticketID = ""
		} else {
			defer func() {
				switch slotType {
				case "user":
					h.concurrencyService.RemoveUserWaitTicket(ctx, id, ticketID)
				default:
					h.concurrencyService.RemoveAccountWaitTicket(ctx, id, ticketID)
				}
			}()
		}
	}

	// Determine if ping is needed (streaming + ping format defined)
	needPing := isStream && h.pingFormat != ""

	// Only create ping ticker if ping is needed
	var pingCh <-chan time.Time
	if needPing {
		pingTicker := time.NewTicker(h.pingInterval)
		defer pingTicker.Stop()
		pingCh = pingTicker.C
	}

	backoff := initialBackoff
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, &ConcurrencyError{
				SlotType:  slotType,
				IsTimeout: true,
			}

		case <-pingCh:
			// Send ping to keep connection alive
			if err := writeConcurrencyPing(gctx, h.pingFormat, streamStarted); err != nil {
				return nil, err
			}

		case <-timer.C:
			if ticketID != "" {
				var (
					isTurn bool
					err    error
				)
				switch slotType {
				case "user":
					isTurn, err = h.concurrencyService.IsUserWaitTicketTurn(ctx, id, ticketID)
				default:
					isTurn, err = h.concurrencyService.IsAccountWaitTicketTurn(ctx, id, ticketID)
				}
				if err != nil {
					return nil, err
				}
				if !isTurn {
					if backoff < fairWaitQueuePollBackoff {
						backoff = fairWaitQueuePollBackoff
					}
					timer.Reset(backoff)
					continue
				}
			}

			// Try to acquire slot
			result, err := acquireSlot()
			if err != nil {
				return nil, err
			}

			if result.Acquired {
				if ticketID != "" {
					switch slotType {
					case "user":
						h.concurrencyService.RemoveUserWaitTicket(ctx, id, ticketID)
					default:
						h.concurrencyService.RemoveAccountWaitTicket(ctx, id, ticketID)
					}
					ticketID = ""
				}
				return result.ReleaseFunc, nil
			}
			backoff = nextBackoff(backoff)
			if ticketID != "" && backoff < fairWaitQueuePollBackoff {
				backoff = fairWaitQueuePollBackoff
			}
			timer.Reset(backoff)
		}
	}
}

// AcquireAccountSlotWithWaitTimeout acquires an account slot with a custom timeout (keeps SSE ping).
func (h *ConcurrencyHelper) AcquireAccountSlotWithWaitTimeout(c *gin.Context, accountID int64, maxConcurrency int, timeout time.Duration, isStream bool, streamStarted *bool) (func(), error) {
	return h.AcquireAccountSlotWithWaitTimeoutContext(gatewayctx.FromGin(c), accountID, maxConcurrency, timeout, isStream, streamStarted)
}

func (h *ConcurrencyHelper) AcquireAccountSlotWithWaitTimeoutContext(ctx gatewayctx.GatewayContext, accountID int64, maxConcurrency int, timeout time.Duration, isStream bool, streamStarted *bool) (func(), error) {
	return h.waitForSlotWithPingTimeoutContext(ctx, "account", accountID, maxConcurrency, timeout, isStream, streamStarted, true)
}

// nextBackoff 计算下一次退避时间
// 性能优化：使用指数退避 + 随机抖动，避免惊群效应
// current: 当前退避时间
// 返回值：下一次退避时间（100ms ~ 2s 之间）
func nextBackoff(current time.Duration) time.Duration {
	// 指数退避：当前时间 * 1.5
	next := time.Duration(float64(current) * backoffMultiplier)
	if next > maxBackoff {
		next = maxBackoff
	}
	// 添加 ±20% 的随机抖动（jitter 范围 0.8 ~ 1.2）
	// 抖动可以分散多个请求的重试时间点，避免同时冲击 Redis
	jitter := 0.8 + rand.Float64()*0.4
	jittered := time.Duration(float64(next) * jitter)
	if jittered < initialBackoff {
		return initialBackoff
	}
	if jittered > maxBackoff {
		return maxBackoff
	}
	return jittered
}
