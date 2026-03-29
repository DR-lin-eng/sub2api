package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserMsgQueueHelper 用户消息串行队列 Handler 层辅助
// 复用 ConcurrencyHelper 的退避 + SSE ping 模式
type UserMsgQueueHelper struct {
	queueService *service.UserMessageQueueService
	pingFormat   SSEPingFormat
	pingInterval time.Duration
}

// NewUserMsgQueueHelper 创建用户消息串行队列辅助
func NewUserMsgQueueHelper(
	queueService *service.UserMessageQueueService,
	pingFormat SSEPingFormat,
	pingInterval time.Duration,
) *UserMsgQueueHelper {
	if pingInterval <= 0 {
		pingInterval = defaultPingInterval
	}
	return &UserMsgQueueHelper{
		queueService: queueService,
		pingFormat:   pingFormat,
		pingInterval: pingInterval,
	}
}

// AcquireWithWait 等待获取串行锁，流式请求期间发送 SSE ping
// 返回的 releaseFunc 内部使用 sync.Once，确保只执行一次释放
func (h *UserMsgQueueHelper) AcquireWithWait(
	c *gin.Context,
	accountID int64,
	baseRPM int,
	isStream bool,
	streamStarted *bool,
	timeout time.Duration,
	reqLog *zap.Logger,
) (releaseFunc func(), err error) {
	return h.AcquireWithWaitContext(gatewayctx.FromGin(c), accountID, baseRPM, isStream, streamStarted, timeout, reqLog)
}

// AcquireWithWaitContext 等待获取串行锁，流式请求期间发送 SSE ping。
func (h *UserMsgQueueHelper) AcquireWithWaitContext(
	c gatewayctx.GatewayContext,
	accountID int64,
	baseRPM int,
	isStream bool,
	streamStarted *bool,
	timeout time.Duration,
	reqLog *zap.Logger,
) (releaseFunc func(), err error) {
	if c == nil {
		return nil, fmt.Errorf("gateway context is nil")
	}

	ctx, cancel := context.WithTimeout(c.Context(), timeout)
	defer cancel()

	// 先尝试立即获取
	result, err := h.queueService.TryAcquire(ctx, accountID)
	if err != nil {
		return nil, err // fail-open 已在 service 层处理
	}

	if result.Acquired {
		// 获取成功，执行 RPM 自适应延迟
		if err := h.queueService.EnforceDelay(ctx, accountID, baseRPM); err != nil {
			if ctx.Err() != nil {
				// 延迟期间 context 取消，释放锁
				bgCtx, bgCancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = h.queueService.Release(bgCtx, accountID, result.RequestID)
				bgCancel()
				return nil, ctx.Err()
			}
		}
		reqLog.Debug("gateway.umq_lock_acquired", zap.Int64("account_id", accountID))
		return h.makeReleaseFunc(accountID, result.RequestID, reqLog), nil
	}

	// 需要等待：指数退避轮询
	return h.waitForLockWithPingContext(c, ctx, accountID, baseRPM, isStream, streamStarted, reqLog)
}

// waitForLockWithPing 等待获取锁，流式请求期间发送 SSE ping
func (h *UserMsgQueueHelper) waitForLockWithPing(
	c *gin.Context,
	ctx context.Context,
	accountID int64,
	baseRPM int,
	isStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), error) {
	return h.waitForLockWithPingContext(gatewayctx.FromGin(c), ctx, accountID, baseRPM, isStream, streamStarted, reqLog)
}

func (h *UserMsgQueueHelper) waitForLockWithPingContext(
	c gatewayctx.GatewayContext,
	ctx context.Context,
	accountID int64,
	baseRPM int,
	isStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), error) {
	if c == nil {
		return nil, fmt.Errorf("gateway context is nil")
	}

	needPing := isStream && h.pingFormat != ""
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
			return nil, fmt.Errorf("umq wait timeout for account %d", accountID)

		case <-pingCh:
			if err := writeConcurrencyPing(c, h.pingFormat, streamStarted); err != nil {
				return nil, err
			}

		case <-timer.C:
			result, err := h.queueService.TryAcquire(ctx, accountID)
			if err != nil {
				return nil, err
			}
			if result.Acquired {
				// 获取成功，执行 RPM 自适应延迟
				if delayErr := h.queueService.EnforceDelay(ctx, accountID, baseRPM); delayErr != nil {
					if ctx.Err() != nil {
						bgCtx, bgCancel := context.WithTimeout(context.Background(), 5*time.Second)
						_ = h.queueService.Release(bgCtx, accountID, result.RequestID)
						bgCancel()
						return nil, ctx.Err()
					}
				}
				reqLog.Debug("gateway.umq_lock_acquired", zap.Int64("account_id", accountID))
				return h.makeReleaseFunc(accountID, result.RequestID, reqLog), nil
			}
			backoff = nextBackoff(backoff)
			timer.Reset(backoff)
		}
	}
}

// makeReleaseFunc 创建锁释放函数（使用 sync.Once 确保只执行一次）
func (h *UserMsgQueueHelper) makeReleaseFunc(accountID int64, requestID string, reqLog *zap.Logger) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			bgCtx, bgCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer bgCancel()
			if err := h.queueService.Release(bgCtx, accountID, requestID); err != nil {
				reqLog.Warn("gateway.umq_release_failed",
					zap.Int64("account_id", accountID),
					zap.Error(err),
				)
			} else {
				reqLog.Debug("gateway.umq_lock_released", zap.Int64("account_id", accountID))
			}
		})
	}
}

// ThrottleWithPing 软性限速模式：施加 RPM 自适应延迟，流式期间发送 SSE ping
// 不获取串行锁，不阻塞并发。返回后即可转发请求。
func (h *UserMsgQueueHelper) ThrottleWithPing(
	c *gin.Context,
	accountID int64,
	baseRPM int,
	isStream bool,
	streamStarted *bool,
	timeout time.Duration,
	reqLog *zap.Logger,
) error {
	return h.ThrottleWithPingContext(gatewayctx.FromGin(c), accountID, baseRPM, isStream, streamStarted, timeout, reqLog)
}

// ThrottleWithPingContext 软性限速模式：施加 RPM 自适应延迟，流式期间发送 SSE ping。
func (h *UserMsgQueueHelper) ThrottleWithPingContext(
	c gatewayctx.GatewayContext,
	accountID int64,
	baseRPM int,
	isStream bool,
	streamStarted *bool,
	timeout time.Duration,
	reqLog *zap.Logger,
) error {
	if c == nil {
		return fmt.Errorf("gateway context is nil")
	}

	ctx, cancel := context.WithTimeout(c.Context(), timeout)
	defer cancel()

	delay := h.queueService.CalculateRPMAwareDelay(ctx, accountID, baseRPM)
	if delay <= 0 {
		return nil
	}

	reqLog.Debug("gateway.umq_throttle_delay",
		zap.Int64("account_id", accountID),
		zap.Duration("delay", delay),
	)

	needPing := isStream && h.pingFormat != ""
	var pingCh <-chan time.Time
	if needPing {
		pingTicker := time.NewTicker(h.pingInterval)
		defer pingTicker.Stop()
		pingCh = pingTicker.C
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pingCh:
			if err := writeConcurrencyPing(c, h.pingFormat, streamStarted); err != nil {
				return err
			}
		case <-timer.C:
			return nil
		}
	}
}
