package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func ExecutableSoraClientRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil || h.SoraClient == nil {
		return nil
	}
	return []gatewayctx.RouteDef{
		{Method: http.MethodPost, Path: "/api/v1/sora/generate", Handler: h.SoraClient.GenerateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		{Method: http.MethodGet, Path: "/api/v1/sora/generations", Handler: h.SoraClient.ListGenerationsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		{Method: http.MethodGet, Path: "/api/v1/sora/generations/:id", Handler: h.SoraClient.GetGenerationGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		{Method: http.MethodDelete, Path: "/api/v1/sora/generations/:id", Handler: h.SoraClient.DeleteGenerationGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		{Method: http.MethodPost, Path: "/api/v1/sora/generations/:id/cancel", Handler: h.SoraClient.CancelGenerationGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		{Method: http.MethodPost, Path: "/api/v1/sora/generations/:id/save", Handler: h.SoraClient.SaveToStorageGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		{Method: http.MethodGet, Path: "/api/v1/sora/quota", Handler: h.SoraClient.GetQuotaGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		{Method: http.MethodGet, Path: "/api/v1/sora/models", Handler: h.SoraClient.GetModelsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		{Method: http.MethodGet, Path: "/api/v1/sora/storage-status", Handler: h.SoraClient.GetStorageStatusGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
	}
}

// RegisterSoraClientRoutes 注册 Sora 客户端 API 路由（需要用户认证）。
func RegisterSoraClientRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
) {
	if h.SoraClient == nil {
		return
	}

	authenticated := v1.Group("/sora")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	{
		authenticated.POST("/generate", h.SoraClient.Generate)
		authenticated.GET("/generations", h.SoraClient.ListGenerations)
		authenticated.GET("/generations/:id", h.SoraClient.GetGeneration)
		authenticated.DELETE("/generations/:id", h.SoraClient.DeleteGeneration)
		authenticated.POST("/generations/:id/cancel", h.SoraClient.CancelGeneration)
		authenticated.POST("/generations/:id/save", h.SoraClient.SaveToStorage)
		authenticated.GET("/quota", h.SoraClient.GetQuota)
		authenticated.GET("/models", h.SoraClient.GetModels)
		authenticated.GET("/storage-status", h.SoraClient.GetStorageStatus)
	}
}
