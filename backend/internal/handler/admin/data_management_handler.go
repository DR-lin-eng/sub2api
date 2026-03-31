package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type DataManagementHandler struct {
	dataManagementService dataManagementService
}

func NewDataManagementHandler(dataManagementService *service.DataManagementService) *DataManagementHandler {
	return &DataManagementHandler{dataManagementService: dataManagementService}
}

type dataManagementService interface {
	GetConfig(ctx context.Context) (service.DataManagementConfig, error)
	UpdateConfig(ctx context.Context, cfg service.DataManagementConfig) (service.DataManagementConfig, error)
	ValidateS3(ctx context.Context, cfg service.DataManagementS3Config) (service.DataManagementTestS3Result, error)
	CreateBackupJob(ctx context.Context, input service.DataManagementCreateBackupJobInput) (service.DataManagementBackupJob, error)
	ListSourceProfiles(ctx context.Context, sourceType string) ([]service.DataManagementSourceProfile, error)
	CreateSourceProfile(ctx context.Context, input service.DataManagementCreateSourceProfileInput) (service.DataManagementSourceProfile, error)
	UpdateSourceProfile(ctx context.Context, input service.DataManagementUpdateSourceProfileInput) (service.DataManagementSourceProfile, error)
	DeleteSourceProfile(ctx context.Context, sourceType, profileID string) error
	SetActiveSourceProfile(ctx context.Context, sourceType, profileID string) (service.DataManagementSourceProfile, error)
	ListS3Profiles(ctx context.Context) ([]service.DataManagementS3Profile, error)
	CreateS3Profile(ctx context.Context, input service.DataManagementCreateS3ProfileInput) (service.DataManagementS3Profile, error)
	UpdateS3Profile(ctx context.Context, input service.DataManagementUpdateS3ProfileInput) (service.DataManagementS3Profile, error)
	DeleteS3Profile(ctx context.Context, profileID string) error
	SetActiveS3Profile(ctx context.Context, profileID string) (service.DataManagementS3Profile, error)
	ListBackupJobs(ctx context.Context, input service.DataManagementListBackupJobsInput) (service.DataManagementListBackupJobsResult, error)
	GetBackupJob(ctx context.Context, jobID string) (service.DataManagementBackupJob, error)
	EnsureAgentEnabled(ctx context.Context) error
	GetAgentHealth(ctx context.Context) service.DataManagementAgentHealth
}

type TestS3ConnectionRequest struct {
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region" binding:"required"`
	Bucket          string `json:"bucket" binding:"required"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Prefix          string `json:"prefix"`
	ForcePathStyle  bool   `json:"force_path_style"`
	UseSSL          bool   `json:"use_ssl"`
}

type CreateBackupJobRequest struct {
	BackupType     string `json:"backup_type" binding:"required,oneof=postgres redis full"`
	UploadToS3     bool   `json:"upload_to_s3"`
	S3ProfileID    string `json:"s3_profile_id"`
	PostgresID     string `json:"postgres_profile_id"`
	RedisID        string `json:"redis_profile_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type CreateSourceProfileRequest struct {
	ProfileID string                             `json:"profile_id" binding:"required"`
	Name      string                             `json:"name" binding:"required"`
	Config    service.DataManagementSourceConfig `json:"config" binding:"required"`
	SetActive bool                               `json:"set_active"`
}

type UpdateSourceProfileRequest struct {
	Name   string                             `json:"name" binding:"required"`
	Config service.DataManagementSourceConfig `json:"config" binding:"required"`
}

type CreateS3ProfileRequest struct {
	ProfileID       string `json:"profile_id" binding:"required"`
	Name            string `json:"name" binding:"required"`
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Prefix          string `json:"prefix"`
	ForcePathStyle  bool   `json:"force_path_style"`
	UseSSL          bool   `json:"use_ssl"`
	SetActive       bool   `json:"set_active"`
}

type UpdateS3ProfileRequest struct {
	Name            string `json:"name" binding:"required"`
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Prefix          string `json:"prefix"`
	ForcePathStyle  bool   `json:"force_path_style"`
	UseSSL          bool   `json:"use_ssl"`
}

func (h *DataManagementHandler) GetAgentHealth(c *gin.Context) {
	h.GetAgentHealthGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) GetAgentHealthGateway(c gatewayctx.GatewayContext) {
	health := h.getAgentHealthGateway(c)
	payload := map[string]any{
		"enabled":     health.Enabled,
		"reason":      health.Reason,
		"socket_path": health.SocketPath,
	}
	if health.Agent != nil {
		payload["agent"] = map[string]any{
			"status":         health.Agent.Status,
			"version":        health.Agent.Version,
			"uptime_seconds": health.Agent.UptimeSeconds,
		}
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, payload)
}

func (h *DataManagementHandler) GetConfig(c *gin.Context) {
	h.GetConfigGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) GetConfigGateway(c gatewayctx.GatewayContext) {
	if !h.requireAgentEnabledGateway(c) {
		return
	}
	cfg, err := h.dataManagementService.GetConfig(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

func (h *DataManagementHandler) UpdateConfig(c *gin.Context) {
	h.UpdateConfigGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) UpdateConfigGateway(c gatewayctx.GatewayContext) {
	var req service.DataManagementConfig
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}
	cfg, err := h.dataManagementService.UpdateConfig(c.Request().Context(), req)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, cfg)
}

func (h *DataManagementHandler) TestS3(c *gin.Context) {
	h.TestS3Gateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) TestS3Gateway(c gatewayctx.GatewayContext) {
	var req TestS3ConnectionRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}
	result, err := h.dataManagementService.ValidateS3(c.Request().Context(), service.DataManagementS3Config{
		Enabled:         true,
		Endpoint:        req.Endpoint,
		Region:          req.Region,
		Bucket:          req.Bucket,
		AccessKeyID:     req.AccessKeyID,
		SecretAccessKey: req.SecretAccessKey,
		Prefix:          req.Prefix,
		ForcePathStyle:  req.ForcePathStyle,
		UseSSL:          req.UseSSL,
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"ok": result.OK, "message": result.Message})
}

func (h *DataManagementHandler) CreateBackupJob(c *gin.Context) {
	h.CreateBackupJobGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) CreateBackupJobGateway(c gatewayctx.GatewayContext) {
	var req CreateBackupJobRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	req.IdempotencyKey = normalizeBackupIdempotencyKey(c.HeaderValue("X-Idempotency-Key"), req.IdempotencyKey)
	if !h.requireAgentEnabledGateway(c) {
		return
	}

	triggeredBy := "admin:unknown"
	if subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c); ok {
		triggeredBy = "admin:" + strconv.FormatInt(subject.UserID, 10)
	}
	job, err := h.dataManagementService.CreateBackupJob(c.Request().Context(), service.DataManagementCreateBackupJobInput{
		BackupType:     req.BackupType,
		UploadToS3:     req.UploadToS3,
		S3ProfileID:    req.S3ProfileID,
		PostgresID:     req.PostgresID,
		RedisID:        req.RedisID,
		TriggeredBy:    triggeredBy,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"job_id": job.JobID, "status": job.Status})
}

func (h *DataManagementHandler) ListSourceProfiles(c *gin.Context) {
	h.ListSourceProfilesGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) ListSourceProfilesGateway(c gatewayctx.GatewayContext) {
	sourceType := strings.TrimSpace(c.PathParam("source_type"))
	if sourceType == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid source_type")
		return
	}
	if sourceType != "postgres" && sourceType != "redis" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "source_type must be postgres or redis")
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}
	items, err := h.dataManagementService.ListSourceProfiles(c.Request().Context(), sourceType)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"items": items})
}

func (h *DataManagementHandler) CreateSourceProfile(c *gin.Context) {
	h.CreateSourceProfileGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) CreateSourceProfileGateway(c gatewayctx.GatewayContext) {
	sourceType := strings.TrimSpace(c.PathParam("source_type"))
	if sourceType != "postgres" && sourceType != "redis" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "source_type must be postgres or redis")
		return
	}

	var req CreateSourceProfileRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}
	profile, err := h.dataManagementService.CreateSourceProfile(c.Request().Context(), service.DataManagementCreateSourceProfileInput{
		SourceType: sourceType,
		ProfileID:  req.ProfileID,
		Name:       req.Name,
		Config:     req.Config,
		SetActive:  req.SetActive,
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, profile)
}

func (h *DataManagementHandler) UpdateSourceProfile(c *gin.Context) {
	h.UpdateSourceProfileGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) UpdateSourceProfileGateway(c gatewayctx.GatewayContext) {
	sourceType := strings.TrimSpace(c.PathParam("source_type"))
	if sourceType != "postgres" && sourceType != "redis" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "source_type must be postgres or redis")
		return
	}
	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid profile_id")
		return
	}

	var req UpdateSourceProfileRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}
	profile, err := h.dataManagementService.UpdateSourceProfile(c.Request().Context(), service.DataManagementUpdateSourceProfileInput{
		SourceType: sourceType,
		ProfileID:  profileID,
		Name:       req.Name,
		Config:     req.Config,
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, profile)
}

func (h *DataManagementHandler) DeleteSourceProfile(c *gin.Context) {
	h.DeleteSourceProfileGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) DeleteSourceProfileGateway(c gatewayctx.GatewayContext) {
	sourceType := strings.TrimSpace(c.PathParam("source_type"))
	if sourceType != "postgres" && sourceType != "redis" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "source_type must be postgres or redis")
		return
	}
	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid profile_id")
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}
	if err := h.dataManagementService.DeleteSourceProfile(c.Request().Context(), sourceType, profileID); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"deleted": true})
}

func (h *DataManagementHandler) SetActiveSourceProfile(c *gin.Context) {
	h.SetActiveSourceProfileGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) SetActiveSourceProfileGateway(c gatewayctx.GatewayContext) {
	sourceType := strings.TrimSpace(c.PathParam("source_type"))
	if sourceType != "postgres" && sourceType != "redis" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "source_type must be postgres or redis")
		return
	}
	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid profile_id")
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}
	profile, err := h.dataManagementService.SetActiveSourceProfile(c.Request().Context(), sourceType, profileID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, profile)
}

func (h *DataManagementHandler) ListS3Profiles(c *gin.Context) {
	h.ListS3ProfilesGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) ListS3ProfilesGateway(c gatewayctx.GatewayContext) {
	if !h.requireAgentEnabledGateway(c) {
		return
	}

	items, err := h.dataManagementService.ListS3Profiles(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"items": items})
}

func (h *DataManagementHandler) CreateS3Profile(c *gin.Context) {
	h.CreateS3ProfileGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) CreateS3ProfileGateway(c gatewayctx.GatewayContext) {
	var req CreateS3ProfileRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}

	profile, err := h.dataManagementService.CreateS3Profile(c.Request().Context(), service.DataManagementCreateS3ProfileInput{
		ProfileID: req.ProfileID,
		Name:      req.Name,
		SetActive: req.SetActive,
		S3: service.DataManagementS3Config{
			Enabled:         req.Enabled,
			Endpoint:        req.Endpoint,
			Region:          req.Region,
			Bucket:          req.Bucket,
			AccessKeyID:     req.AccessKeyID,
			SecretAccessKey: req.SecretAccessKey,
			Prefix:          req.Prefix,
			ForcePathStyle:  req.ForcePathStyle,
			UseSSL:          req.UseSSL,
		},
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, profile)
}

func (h *DataManagementHandler) UpdateS3Profile(c *gin.Context) {
	h.UpdateS3ProfileGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) UpdateS3ProfileGateway(c gatewayctx.GatewayContext) {
	var req UpdateS3ProfileRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid profile_id")
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}

	profile, err := h.dataManagementService.UpdateS3Profile(c.Request().Context(), service.DataManagementUpdateS3ProfileInput{
		ProfileID: profileID,
		Name:      req.Name,
		S3: service.DataManagementS3Config{
			Enabled:         req.Enabled,
			Endpoint:        req.Endpoint,
			Region:          req.Region,
			Bucket:          req.Bucket,
			AccessKeyID:     req.AccessKeyID,
			SecretAccessKey: req.SecretAccessKey,
			Prefix:          req.Prefix,
			ForcePathStyle:  req.ForcePathStyle,
			UseSSL:          req.UseSSL,
		},
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, profile)
}

func (h *DataManagementHandler) DeleteS3Profile(c *gin.Context) {
	h.DeleteS3ProfileGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) DeleteS3ProfileGateway(c gatewayctx.GatewayContext) {
	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid profile_id")
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}
	if err := h.dataManagementService.DeleteS3Profile(c.Request().Context(), profileID); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"deleted": true})
}

func (h *DataManagementHandler) SetActiveS3Profile(c *gin.Context) {
	h.SetActiveS3ProfileGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) SetActiveS3ProfileGateway(c gatewayctx.GatewayContext) {
	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid profile_id")
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}
	profile, err := h.dataManagementService.SetActiveS3Profile(c.Request().Context(), profileID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, profile)
}

func (h *DataManagementHandler) ListBackupJobs(c *gin.Context) {
	h.ListBackupJobsGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) ListBackupJobsGateway(c gatewayctx.GatewayContext) {
	if !h.requireAgentEnabledGateway(c) {
		return
	}

	pageSize := int32(20)
	if raw := strings.TrimSpace(c.QueryValue("page_size")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid page_size")
			return
		}
		pageSize = int32(v)
	}

	result, err := h.dataManagementService.ListBackupJobs(c.Request().Context(), service.DataManagementListBackupJobsInput{
		PageSize:   pageSize,
		PageToken:  c.QueryValue("page_token"),
		Status:     c.QueryValue("status"),
		BackupType: c.QueryValue("backup_type"),
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

func (h *DataManagementHandler) GetBackupJob(c *gin.Context) {
	h.GetBackupJobGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) GetBackupJobGateway(c gatewayctx.GatewayContext) {
	jobID := strings.TrimSpace(c.PathParam("job_id"))
	if jobID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid backup job ID")
		return
	}

	if !h.requireAgentEnabledGateway(c) {
		return
	}
	job, err := h.dataManagementService.GetBackupJob(c.Request().Context(), jobID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, job)
}

func (h *DataManagementHandler) requireAgentEnabled(c *gin.Context) bool {
	return h.requireAgentEnabledGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) requireAgentEnabledGateway(c gatewayctx.GatewayContext) bool {
	if h.dataManagementService == nil {
		err := infraerrors.ServiceUnavailable(
			service.DataManagementAgentUnavailableReason,
			"data management agent service is not configured",
		).WithMetadata(map[string]string{"socket_path": service.DefaultDataManagementAgentSocketPath})
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return false
	}

	if err := h.dataManagementService.EnsureAgentEnabled(c.Request().Context()); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return false
	}

	return true
}

func (h *DataManagementHandler) getAgentHealth(c *gin.Context) service.DataManagementAgentHealth {
	return h.getAgentHealthGateway(gatewayctx.FromGin(c))
}

func (h *DataManagementHandler) getAgentHealthGateway(c gatewayctx.GatewayContext) service.DataManagementAgentHealth {
	if h.dataManagementService == nil {
		return service.DataManagementAgentHealth{
			Enabled:    false,
			Reason:     service.DataManagementAgentUnavailableReason,
			SocketPath: service.DefaultDataManagementAgentSocketPath,
		}
	}
	return h.dataManagementService.GetAgentHealth(c.Request().Context())
}

func normalizeBackupIdempotencyKey(headerValue, bodyValue string) string {
	headerKey := strings.TrimSpace(headerValue)
	if headerKey != "" {
		return headerKey
	}
	return strings.TrimSpace(bodyValue)
}
