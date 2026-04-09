package handler

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type claudeBootstrapResponse struct {
	ClientData             map[string]any                   `json:"client_data"`
	AdditionalModelOptions []claudeBootstrapAdditionalModel `json:"additional_model_options"`
}

type claudeBootstrapAdditionalModel struct {
	Model       string `json:"model"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type claudeMetricsEnabledResponse struct {
	MetricsLoggingEnabled bool `json:"metrics_logging_enabled"`
}

type claudePolicyLimitsResponse struct {
	Restrictions map[string]claudePolicyRestriction `json:"restrictions"`
}

type claudePolicyRestriction struct {
	Allowed bool `json:"allowed"`
}

type claudeManagedSettingsResponse struct {
	UUID     string         `json:"uuid"`
	Checksum string         `json:"checksum"`
	Settings map[string]any `json:"settings"`
}

type claudeUserSettingsEnvelope struct {
	UserID       string                    `json:"userId"`
	Version      int                       `json:"version"`
	LastModified string                    `json:"lastModified"`
	Checksum     string                    `json:"checksum"`
	Content      claudeUserSettingsContent `json:"content"`
}

type claudeUserSettingsContent struct {
	Entries map[string]string `json:"entries"`
}

type claudeUpdateUserSettingsRequest struct {
	Entries map[string]string `json:"entries"`
}

var defaultClaudeBootstrapResponse = claudeBootstrapResponse{
	ClientData:             map[string]any{},
	AdditionalModelOptions: []claudeBootstrapAdditionalModel{},
}

var defaultClaudeMetricsEnabledResponse = claudeMetricsEnabledResponse{
	MetricsLoggingEnabled: false,
}

var defaultClaudePolicyLimitsResponse = claudePolicyLimitsResponse{
	Restrictions: map[string]claudePolicyRestriction{},
}

var defaultClaudeManagedSettingsResponse = claudeManagedSettingsResponse{
	UUID:     "sub2api-local-managed-settings",
	Checksum: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	Settings: map[string]any{},
}

var claudeManagedSettingsETag = buildStaticETagFromString("claude-managed-settings:none")
var claudePolicyLimitsETag = buildStaticETagFromJSON(defaultClaudePolicyLimitsResponse)

var claudeUserSettingsStore sync.Map

func (h *GatewayHandler) ClaudeBootstrap(c *gin.Context) {
	h.ClaudeBootstrapGateway(gatewayctx.FromGin(c))
}

func (h *GatewayHandler) ClaudeBootstrapGateway(c gatewayctx.GatewayContext) {
	if !supportsClaudeTelemetrySimulation(c) {
		h.errorResponseGateway(c, http.StatusNotFound, "not_found_error", "Bootstrap is not supported for this platform")
		return
	}
	c.SetHeader("Cache-Control", "private, max-age=3600")
	c.WriteJSON(http.StatusOK, defaultClaudeBootstrapResponse)
}

func (h *GatewayHandler) ClaudeMetricsEnabled(c *gin.Context) {
	h.ClaudeMetricsEnabledGateway(gatewayctx.FromGin(c))
}

func (h *GatewayHandler) ClaudeMetricsEnabledGateway(c gatewayctx.GatewayContext) {
	if !supportsClaudeTelemetrySimulation(c) {
		h.errorResponseGateway(c, http.StatusNotFound, "not_found_error", "Metrics telemetry is not supported for this platform")
		return
	}
	c.SetHeader("Cache-Control", "private, max-age=86400")
	c.WriteJSON(http.StatusOK, defaultClaudeMetricsEnabledResponse)
}

func (h *GatewayHandler) ClaudeManagedSettings(c *gin.Context) {
	h.ClaudeManagedSettingsGateway(gatewayctx.FromGin(c))
}

func (h *GatewayHandler) ClaudeManagedSettingsGateway(c gatewayctx.GatewayContext) {
	if !supportsClaudeTelemetrySimulation(c) {
		h.errorResponseGateway(c, http.StatusNotFound, "not_found_error", "Managed settings are not supported for this platform")
		return
	}
	c.SetHeader("Cache-Control", "private, max-age=3600")
	if applyStaticETagHeaders(c, claudeManagedSettingsETag) {
		return
	}
	_, _ = c.WriteBytes(http.StatusNoContent, nil)
}

func (h *GatewayHandler) ClaudePolicyLimits(c *gin.Context) {
	h.ClaudePolicyLimitsGateway(gatewayctx.FromGin(c))
}

func (h *GatewayHandler) ClaudePolicyLimitsGateway(c gatewayctx.GatewayContext) {
	if !supportsClaudeTelemetrySimulation(c) {
		h.errorResponseGateway(c, http.StatusNotFound, "not_found_error", "Policy limits are not supported for this platform")
		return
	}
	c.SetHeader("Cache-Control", "private, max-age=3600")
	if applyStaticETagHeaders(c, claudePolicyLimitsETag) {
		return
	}
	c.WriteJSON(http.StatusOK, defaultClaudePolicyLimitsResponse)
}

func (h *GatewayHandler) ClaudeUserSettings(c *gin.Context) {
	h.ClaudeUserSettingsGateway(gatewayctx.FromGin(c))
}

func (h *GatewayHandler) ClaudeUserSettingsGateway(c gatewayctx.GatewayContext) {
	if !supportsClaudeTelemetrySimulation(c) {
		h.errorResponseGateway(c, http.StatusNotFound, "not_found_error", "User settings sync is not supported for this platform")
		return
	}
	userKey := claudeUserSettingsStoreKey(c)
	if userKey == "" {
		h.errorResponseGateway(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	if value, ok := claudeUserSettingsStore.Load(userKey); ok {
		if state, ok := value.(claudeUserSettingsEnvelope); ok {
			c.SetHeader("Cache-Control", "private, max-age=300")
			c.WriteJSON(http.StatusOK, state)
			return
		}
	}
	c.SetStatus(http.StatusNotFound)
}

func (h *GatewayHandler) ClaudeUpdateUserSettings(c *gin.Context) {
	h.ClaudeUpdateUserSettingsGateway(gatewayctx.FromGin(c))
}

func (h *GatewayHandler) ClaudeUpdateUserSettingsGateway(c gatewayctx.GatewayContext) {
	if !supportsClaudeTelemetrySimulation(c) {
		h.errorResponseGateway(c, http.StatusNotFound, "not_found_error", "User settings sync is not supported for this platform")
		return
	}
	userKey := claudeUserSettingsStoreKey(c)
	if userKey == "" {
		h.errorResponseGateway(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	var req claudeUpdateUserSettingsRequest
	if err := c.BindJSON(&req); err != nil {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "Invalid user settings payload")
		return
	}
	if req.Entries == nil {
		req.Entries = map[string]string{}
	}

	version := 1
	if current, ok := claudeUserSettingsStore.Load(userKey); ok {
		if state, ok := current.(claudeUserSettingsEnvelope); ok && state.Version > 0 {
			version = state.Version + 1
		}
	}
	lastModified := time.Now().UTC().Format(time.RFC3339)
	state := claudeUserSettingsEnvelope{
		UserID:       userKey,
		Version:      version,
		LastModified: lastModified,
		Checksum:     buildUserSettingsChecksum(req.Entries),
		Content: claudeUserSettingsContent{
			Entries: cloneStringMap(req.Entries),
		},
	}
	claudeUserSettingsStore.Store(userKey, state)
	c.SetHeader("Cache-Control", "private, max-age=300")
	c.WriteJSON(http.StatusOK, map[string]any{
		"checksum":     state.Checksum,
		"lastModified": state.LastModified,
	})
}

func supportsClaudeTelemetrySimulation(c gatewayctx.GatewayContext) bool {
	switch resolveGatewayPlatform(c) {
	case service.PlatformAnthropic, service.PlatformAntigravity:
		return true
	default:
		return false
	}
}

func resolveGatewayPlatform(c gatewayctx.GatewayContext) string {
	if forcedPlatform, ok := middleware2.GetForcePlatformFromGatewayContext(c); ok && forcedPlatform != "" {
		return forcedPlatform
	}
	apiKey, ok := middleware2.GetAPIKeyFromGatewayContext(c)
	if !ok || apiKey == nil || apiKey.Group == nil {
		return ""
	}
	return apiKey.Group.Platform
}

func claudeUserSettingsStoreKey(c gatewayctx.GatewayContext) string {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok || subject.UserID <= 0 {
		return ""
	}
	return strconv.FormatInt(subject.UserID, 10)
}

func applyStaticETagHeaders(c gatewayctx.GatewayContext, etag string) bool {
	if c == nil || strings.TrimSpace(etag) == "" {
		return false
	}
	c.SetHeader("ETag", etag)
	c.SetHeader("Vary", "If-None-Match")
	if ifNoneMatchMatched(c.HeaderValue("If-None-Match"), etag) {
		_, _ = c.WriteBytes(http.StatusNotModified, nil)
		return true
	}
	return false
}

func ifNoneMatchMatched(ifNoneMatch, etag string) bool {
	if strings.TrimSpace(ifNoneMatch) == "" || strings.TrimSpace(etag) == "" {
		return false
	}
	for _, token := range strings.Split(ifNoneMatch, ",") {
		candidate := strings.TrimSpace(token)
		if candidate == "*" || candidate == etag {
			return true
		}
		if strings.HasPrefix(candidate, "W/") && strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

func buildStaticETagFromString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func buildStaticETagFromJSON(payload any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		return buildStaticETagFromString(fmt.Sprintf("%v", payload))
	}
	return buildStaticETagFromString(string(raw))
}

func buildUserSettingsChecksum(entries map[string]string) string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(entries[key])
		builder.WriteByte('\n')
	}
	sum := md5.Sum([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}
