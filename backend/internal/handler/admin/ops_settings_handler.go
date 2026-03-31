package admin

import (
	"encoding/json"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// GetEmailNotificationConfig returns Ops email notification config (DB-backed).
// GET /api/v1/admin/ops/email-notification/config
func (h *OpsHandler) GetEmailNotificationConfig(c *gin.Context) {
	h.GetEmailNotificationConfigGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) GetEmailNotificationConfigGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	cfg, err := h.opsService.GetEmailNotificationConfig(c.Request().Context())
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get email notification config")
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

// UpdateEmailNotificationConfig updates Ops email notification config (DB-backed).
// PUT /api/v1/admin/ops/email-notification/config
func (h *OpsHandler) UpdateEmailNotificationConfig(c *gin.Context) {
	h.UpdateEmailNotificationConfigGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) UpdateEmailNotificationConfigGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	var req service.OpsEmailNotificationConfigUpdateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request body")
		return
	}

	updated, err := h.opsService.UpdateEmailNotificationConfig(c.Request().Context(), &req)
	if err != nil {
		// Most failures here are validation errors from request payload; treat as 400.
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, updated)
}

// GetAlertRuntimeSettings returns Ops alert evaluator runtime settings (DB-backed).
// GET /api/v1/admin/ops/runtime/alert
func (h *OpsHandler) GetAlertRuntimeSettings(c *gin.Context) {
	h.GetAlertRuntimeSettingsGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) GetAlertRuntimeSettingsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	cfg, err := h.opsService.GetOpsAlertRuntimeSettings(c.Request().Context())
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get alert runtime settings")
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

// UpdateAlertRuntimeSettings updates Ops alert evaluator runtime settings (DB-backed).
// PUT /api/v1/admin/ops/runtime/alert
func (h *OpsHandler) UpdateAlertRuntimeSettings(c *gin.Context) {
	h.UpdateAlertRuntimeSettingsGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) UpdateAlertRuntimeSettingsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	var req service.OpsAlertRuntimeSettings
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request body")
		return
	}

	updated, err := h.opsService.UpdateOpsAlertRuntimeSettings(c.Request().Context(), &req)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, updated)
}

// GetRuntimeLogConfig returns runtime log config (DB-backed).
// GET /api/v1/admin/ops/runtime/logging
func (h *OpsHandler) GetRuntimeLogConfig(c *gin.Context) {
	h.GetRuntimeLogConfigGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) GetRuntimeLogConfigGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	cfg, err := h.opsService.GetRuntimeLogConfig(c.Request().Context())
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get runtime log config")
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

// UpdateRuntimeLogConfig updates runtime log config and applies changes immediately.
// PUT /api/v1/admin/ops/runtime/logging
func (h *OpsHandler) UpdateRuntimeLogConfig(c *gin.Context) {
	h.UpdateRuntimeLogConfigGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) UpdateRuntimeLogConfigGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	var req service.OpsRuntimeLogConfig
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request body")
		return
	}

	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Unauthorized")
		return
	}

	updated, err := h.opsService.UpdateRuntimeLogConfig(c.Request().Context(), &req, subject.UserID)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, updated)
}

// ResetRuntimeLogConfig removes runtime override and falls back to env/yaml baseline.
// POST /api/v1/admin/ops/runtime/logging/reset
func (h *OpsHandler) ResetRuntimeLogConfig(c *gin.Context) {
	h.ResetRuntimeLogConfigGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) ResetRuntimeLogConfigGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	subject, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "Unauthorized")
		return
	}

	updated, err := h.opsService.ResetRuntimeLogConfig(c.Request().Context(), subject.UserID)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, updated)
}

// GetAdvancedSettings returns Ops advanced settings (DB-backed).
// GET /api/v1/admin/ops/advanced-settings
func (h *OpsHandler) GetAdvancedSettings(c *gin.Context) {
	h.GetAdvancedSettingsGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) GetAdvancedSettingsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	cfg, err := h.opsService.GetOpsAdvancedSettings(c.Request().Context())
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get advanced settings")
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

// UpdateAdvancedSettings updates Ops advanced settings (DB-backed).
// PUT /api/v1/admin/ops/advanced-settings
func (h *OpsHandler) UpdateAdvancedSettings(c *gin.Context) {
	h.UpdateAdvancedSettingsGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) UpdateAdvancedSettingsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	var req service.OpsAdvancedSettings
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request body")
		return
	}

	updated, err := h.opsService.UpdateOpsAdvancedSettings(c.Request().Context(), &req)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, updated)
}

// GetMetricThresholds returns Ops metric thresholds (DB-backed).
// GET /api/v1/admin/ops/settings/metric-thresholds
func (h *OpsHandler) GetMetricThresholds(c *gin.Context) {
	h.GetMetricThresholdsGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) GetMetricThresholdsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	cfg, err := h.opsService.GetMetricThresholds(c.Request().Context())
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Failed to get metric thresholds")
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

// UpdateMetricThresholds updates Ops metric thresholds (DB-backed).
// PUT /api/v1/admin/ops/settings/metric-thresholds
func (h *OpsHandler) UpdateMetricThresholds(c *gin.Context) {
	h.UpdateMetricThresholdsGateway(gatewayctx.FromGin(c))
}

func (h *OpsHandler) UpdateMetricThresholdsGateway(c gatewayctx.GatewayContext) {
	if h.opsService == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	var req service.OpsMetricThresholds
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request body")
		return
	}

	updated, err := h.opsService.UpdateMetricThresholds(c.Request().Context(), &req)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, updated)
}
