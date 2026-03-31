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

type BackupHandler struct {
	backupService *service.BackupService
	userService   *service.UserService
}

func NewBackupHandler(backupService *service.BackupService, userService *service.UserService) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
		userService:   userService,
	}
}

// ─── S3 配置 ───

func (h *BackupHandler) GetS3Config(c *gin.Context) {
	h.GetS3ConfigGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) GetS3ConfigGateway(c gatewayctx.GatewayContext) {
	cfg, err := h.backupService.GetS3Config(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

func (h *BackupHandler) UpdateS3Config(c *gin.Context) {
	h.UpdateS3ConfigGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) UpdateS3ConfigGateway(c gatewayctx.GatewayContext) {
	var req service.BackupS3Config
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	cfg, err := h.backupService.UpdateS3Config(c.Request().Context(), req)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

func (h *BackupHandler) TestS3Connection(c *gin.Context) {
	h.TestS3ConnectionGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) TestS3ConnectionGateway(c gatewayctx.GatewayContext) {
	var req service.BackupS3Config
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	err := h.backupService.TestS3Connection(c.Request().Context(), req)
	if err != nil {
		response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"ok": true, "message": "connection successful"})
}

// ─── 定时备份 ───

func (h *BackupHandler) GetSchedule(c *gin.Context) {
	h.GetScheduleGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) GetScheduleGateway(c gatewayctx.GatewayContext) {
	cfg, err := h.backupService.GetSchedule(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

func (h *BackupHandler) UpdateSchedule(c *gin.Context) {
	h.UpdateScheduleGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) UpdateScheduleGateway(c gatewayctx.GatewayContext) {
	var req service.BackupScheduleConfig
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	cfg, err := h.backupService.UpdateSchedule(c.Request().Context(), req)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

// ─── 备份操作 ───

type CreateBackupRequest struct {
	ExpireDays *int `json:"expire_days"` // nil=使用默认值14，0=永不过期
}

func (h *BackupHandler) CreateBackup(c *gin.Context) {
	h.CreateBackupGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) CreateBackupGateway(c gatewayctx.GatewayContext) {
	var req CreateBackupRequest
	if c.Request() != nil && c.Request().ContentLength > 0 {
		_ = json.NewDecoder(c.Request().Body).Decode(&req)
	}

	expireDays := 14 // 默认14天过期
	if req.ExpireDays != nil {
		expireDays = *req.ExpireDays
	}

	record, err := h.backupService.StartBackup(c.Request().Context(), "manual", expireDays)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.AcceptedContext(gatewayJSONResponder{ctx: c}, record)
}

func (h *BackupHandler) ListBackups(c *gin.Context) {
	h.ListBackupsGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) ListBackupsGateway(c gatewayctx.GatewayContext) {
	records, err := h.backupService.ListBackups(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	if records == nil {
		records = []service.BackupRecord{}
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"items": records})
}

func (h *BackupHandler) GetBackup(c *gin.Context) {
	h.GetBackupGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) GetBackupGateway(c gatewayctx.GatewayContext) {
	backupID := c.PathParam("id")
	if backupID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "backup ID is required")
		return
	}
	record, err := h.backupService.GetBackupRecord(c.Request().Context(), backupID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, record)
}

func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	h.DeleteBackupGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) DeleteBackupGateway(c gatewayctx.GatewayContext) {
	backupID := c.PathParam("id")
	if backupID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "backup ID is required")
		return
	}
	if err := h.backupService.DeleteBackup(c.Request().Context(), backupID); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"deleted": true})
}

func (h *BackupHandler) GetDownloadURL(c *gin.Context) {
	h.GetDownloadURLGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) GetDownloadURLGateway(c gatewayctx.GatewayContext) {
	backupID := c.PathParam("id")
	if backupID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "backup ID is required")
		return
	}
	url, err := h.backupService.GetBackupDownloadURL(c.Request().Context(), backupID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"url": url})
}

// ─── 恢复操作（需要重新输入管理员密码） ───

type RestoreBackupRequest struct {
	Password string `json:"password" binding:"required"`
}

func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	h.RestoreBackupGateway(gatewayctx.FromGin(c))
}

func (h *BackupHandler) RestoreBackupGateway(c gatewayctx.GatewayContext) {
	backupID := c.PathParam("id")
	if backupID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "backup ID is required")
		return
	}

	var req RestoreBackupRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "password is required for restore operation")
		return
	}

	// 从上下文获取当前管理员用户 ID
	sub, ok := middleware.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "unauthorized")
		return
	}

	// 获取管理员用户并验证密码
	user, err := h.userService.GetByID(c.Request().Context(), sub.UserID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	if !user.CheckPassword(req.Password) {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "incorrect admin password")
		return
	}

	record, err := h.backupService.StartRestore(c.Request().Context(), backupID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.AcceptedContext(gatewayJSONResponder{ctx: c}, record)
}
