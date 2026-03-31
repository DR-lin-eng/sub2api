package admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/sysutil"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (h *SystemHandler) GetVersionGateway(c gatewayctx.GatewayContext) {
	if c == nil {
		return
	}
	info, _ := h.updateSvc.CheckUpdate(c.Request().Context(), false)
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"version": info.CurrentVersion,
	})
}

func (h *SystemHandler) CheckUpdatesGateway(c gatewayctx.GatewayContext) {
	if c == nil {
		return
	}
	force := strings.EqualFold(strings.TrimSpace(c.QueryValue("force")), "true")
	info, err := h.updateSvc.CheckUpdate(c.Request().Context(), force)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, info)
}

func (h *SystemHandler) PerformUpdateGateway(c gatewayctx.GatewayContext) {
	if c == nil {
		return
	}
	operationID := buildSystemOperationIDGateway(c, "update")
	payload := map[string]any{"operation_id": operationID}
	executeAdminIdempotentGatewayJSON(c, "admin.system.update", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		var releaseReason string
		succeeded := false
		defer func() {
			release(releaseReason, succeeded)
		}()

		if err := h.updateSvc.PerformUpdate(ctx); err != nil {
			releaseReason = "SYSTEM_UPDATE_FAILED"
			return nil, err
		}
		succeeded = true

		return map[string]any{
			"message":      "Update completed. Please restart the service.",
			"need_restart": true,
			"operation_id": lock.OperationID(),
		}, nil
	})
}

func (h *SystemHandler) RollbackGateway(c gatewayctx.GatewayContext) {
	if c == nil {
		return
	}
	operationID := buildSystemOperationIDGateway(c, "rollback")
	payload := map[string]any{"operation_id": operationID}
	executeAdminIdempotentGatewayJSON(c, "admin.system.rollback", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		var releaseReason string
		succeeded := false
		defer func() {
			release(releaseReason, succeeded)
		}()

		if err := h.updateSvc.Rollback(); err != nil {
			releaseReason = "SYSTEM_ROLLBACK_FAILED"
			return nil, err
		}
		succeeded = true

		return map[string]any{
			"message":      "Rollback completed. Please restart the service.",
			"need_restart": true,
			"operation_id": lock.OperationID(),
		}, nil
	})
}

func (h *SystemHandler) RestartServiceGateway(c gatewayctx.GatewayContext) {
	if c == nil {
		return
	}
	operationID := buildSystemOperationIDGateway(c, "restart")
	payload := map[string]any{"operation_id": operationID}
	executeAdminIdempotentGatewayJSON(c, "admin.system.restart", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		succeeded := false
		defer func() {
			release("", succeeded)
		}()

		go func() {
			time.Sleep(500 * time.Millisecond)
			sysutil.RestartServiceAsync()
		}()
		succeeded = true

		return map[string]any{
			"message":      "Service restart initiated",
			"operation_id": lock.OperationID(),
		}, nil
	})
}

func buildSystemOperationIDGateway(c gatewayctx.GatewayContext, operation string) string {
	key := strings.TrimSpace(c.HeaderValue("Idempotency-Key"))
	if key == "" {
		return "sysop-" + operation + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	actorScope := "admin:0"
	if subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c); ok {
		actorScope = "admin:" + strconv.FormatInt(subject.UserID, 10)
	}
	seed := operation + "|" + actorScope + "|" + c.Path() + "|" + key
	hash := service.HashIdempotencyKey(seed)
	if len(hash) > 24 {
		hash = hash[:24]
	}
	return "sysop-" + hash
}
