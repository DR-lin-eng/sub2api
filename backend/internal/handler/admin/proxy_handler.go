package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ProxyHandler handles admin proxy management
type ProxyHandler struct {
	adminService service.AdminService
}

// NewProxyHandler creates a new admin proxy handler
func NewProxyHandler(adminService service.AdminService) *ProxyHandler {
	return &ProxyHandler{
		adminService: adminService,
	}
}

// CreateProxyRequest represents create proxy request
type CreateProxyRequest struct {
	Name     string `json:"name" binding:"required"`
	Protocol string `json:"protocol" binding:"required,oneof=http https socks4 socks5 socks5h"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required,min=1,max=65535"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// UpdateProxyRequest represents update proxy request
type UpdateProxyRequest struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol" binding:"omitempty,oneof=http https socks4 socks5 socks5h"`
	Host     string `json:"host"`
	Port     int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username string `json:"username"`
	Password string `json:"password"`
	Status   string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// List handles listing all proxies with pagination
// GET /api/v1/admin/proxies
func (h *ProxyHandler) List(c *gin.Context) {
	h.ListGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) ListGateway(c gatewayctx.GatewayContext) {
	page, pageSize := response.ParsePaginationValues(c)
	protocol := c.QueryValue("protocol")
	status := c.QueryValue("status")
	search := c.QueryValue("search")
	// 标准化和验证 search 参数
	search = strings.TrimSpace(search)
	if len(search) > 100 {
		search = search[:100]
	}

	proxies, total, err := h.adminService.ListProxiesWithAccountCount(c.Request().Context(), page, pageSize, protocol, status, search)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	out := make([]dto.AdminProxyWithAccountCount, 0, len(proxies))
	for i := range proxies {
		out = append(out, *dto.ProxyWithAccountCountFromServiceAdmin(&proxies[i]))
	}
	response.PaginatedContext(gatewayJSONResponder{ctx: c}, out, total, page, pageSize)
}

// GetAll handles getting all active proxies without pagination
// GET /api/v1/admin/proxies/all
// Optional query param: with_count=true to include account count per proxy
func (h *ProxyHandler) GetAll(c *gin.Context) {
	h.GetAllGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) GetAllGateway(c gatewayctx.GatewayContext) {
	withCount := c.QueryValue("with_count") == "true"

	if withCount {
		proxies, err := h.adminService.GetAllProxiesWithAccountCount(c.Request().Context())
		if err != nil {
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}
		out := make([]dto.AdminProxyWithAccountCount, 0, len(proxies))
		for i := range proxies {
			out = append(out, *dto.ProxyWithAccountCountFromServiceAdmin(&proxies[i]))
		}
		response.SuccessContext(gatewayJSONResponder{ctx: c}, out)
		return
	}

	proxies, err := h.adminService.GetAllProxies(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	out := make([]dto.AdminProxy, 0, len(proxies))
	for i := range proxies {
		out = append(out, *dto.ProxyFromServiceAdmin(&proxies[i]))
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, out)
}

// GetByID handles getting a proxy by ID
// GET /api/v1/admin/proxies/:id
func (h *ProxyHandler) GetByID(c *gin.Context) {
	h.GetByIDGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) GetByIDGateway(c gatewayctx.GatewayContext) {
	proxyID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid proxy ID")
		return
	}

	proxy, err := h.adminService.GetProxy(c.Request().Context(), proxyID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.ProxyFromServiceAdmin(proxy))
}

// Create handles creating a new proxy
// POST /api/v1/admin/proxies
func (h *ProxyHandler) Create(c *gin.Context) {
	h.CreateGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) CreateGateway(c gatewayctx.GatewayContext) {
	var req CreateProxyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	executeAdminIdempotentGatewayJSON(c, "admin.proxies.create", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		proxy, err := h.adminService.CreateProxy(ctx, &service.CreateProxyInput{
			Name:     strings.TrimSpace(req.Name),
			Protocol: strings.TrimSpace(req.Protocol),
			Host:     strings.TrimSpace(req.Host),
			Port:     req.Port,
			Username: strings.TrimSpace(req.Username),
			Password: strings.TrimSpace(req.Password),
		})
		if err != nil {
			return nil, err
		}
		return dto.ProxyFromServiceAdmin(proxy), nil
	})
}

// Update handles updating a proxy
// PUT /api/v1/admin/proxies/:id
func (h *ProxyHandler) Update(c *gin.Context) {
	h.UpdateGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) UpdateGateway(c gatewayctx.GatewayContext) {
	proxyID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid proxy ID")
		return
	}

	var req UpdateProxyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	proxy, err := h.adminService.UpdateProxy(c.Request().Context(), proxyID, &service.UpdateProxyInput{
		Name:     strings.TrimSpace(req.Name),
		Protocol: strings.TrimSpace(req.Protocol),
		Host:     strings.TrimSpace(req.Host),
		Port:     req.Port,
		Username: strings.TrimSpace(req.Username),
		Password: strings.TrimSpace(req.Password),
		Status:   strings.TrimSpace(req.Status),
	})
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.ProxyFromServiceAdmin(proxy))
}

// Delete handles deleting a proxy
// DELETE /api/v1/admin/proxies/:id
func (h *ProxyHandler) Delete(c *gin.Context) {
	h.DeleteGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) DeleteGateway(c gatewayctx.GatewayContext) {
	proxyID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid proxy ID")
		return
	}

	err = h.adminService.DeleteProxy(c.Request().Context(), proxyID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"message": "Proxy deleted successfully"})
}

// BatchDelete handles batch deleting proxies
// POST /api/v1/admin/proxies/batch-delete
func (h *ProxyHandler) BatchDelete(c *gin.Context) {
	h.BatchDeleteGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) BatchDeleteGateway(c gatewayctx.GatewayContext) {
	type BatchDeleteRequest struct {
		IDs []int64 `json:"ids" binding:"required,min=1"`
	}

	var req BatchDeleteRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	result, err := h.adminService.BatchDeleteProxies(c.Request().Context(), req.IDs)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

// Test handles testing proxy connectivity
// POST /api/v1/admin/proxies/:id/test
func (h *ProxyHandler) Test(c *gin.Context) {
	h.TestGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) TestGateway(c gatewayctx.GatewayContext) {
	proxyID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid proxy ID")
		return
	}

	result, err := h.adminService.TestProxy(c.Request().Context(), proxyID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

// CheckQuality handles checking proxy quality across common AI targets.
// POST /api/v1/admin/proxies/:id/quality-check
func (h *ProxyHandler) CheckQuality(c *gin.Context) {
	h.CheckQualityGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) CheckQualityGateway(c gatewayctx.GatewayContext) {
	proxyID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid proxy ID")
		return
	}

	result, err := h.adminService.CheckProxyQuality(c.Request().Context(), proxyID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

// GetStats handles getting proxy statistics
// GET /api/v1/admin/proxies/:id/stats
func (h *ProxyHandler) GetStats(c *gin.Context) {
	h.GetStatsGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) GetStatsGateway(c gatewayctx.GatewayContext) {
	proxyID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid proxy ID")
		return
	}

	// Return mock data for now
	_ = proxyID
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"total_accounts":  0,
		"active_accounts": 0,
		"total_requests":  0,
		"success_rate":    100.0,
		"average_latency": 0,
	})
}

// GetProxyAccounts handles getting accounts using a proxy
// GET /api/v1/admin/proxies/:id/accounts
func (h *ProxyHandler) GetProxyAccounts(c *gin.Context) {
	h.GetProxyAccountsGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) GetProxyAccountsGateway(c gatewayctx.GatewayContext) {
	proxyID, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid proxy ID")
		return
	}

	accounts, err := h.adminService.GetProxyAccounts(c.Request().Context(), proxyID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	out := make([]dto.ProxyAccountSummary, 0, len(accounts))
	for i := range accounts {
		out = append(out, *dto.ProxyAccountSummaryFromService(&accounts[i]))
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, out)
}

// BatchCreateProxyItem represents a single proxy in batch create request
type BatchCreateProxyItem struct {
	Protocol string `json:"protocol" binding:"required,oneof=http https socks4 socks5 socks5h"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required,min=1,max=65535"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// BatchCreateRequest represents batch create proxies request
type BatchCreateRequest struct {
	Proxies []BatchCreateProxyItem `json:"proxies" binding:"required,min=1"`
}

// BatchCreate handles batch creating proxies
// POST /api/v1/admin/proxies/batch
func (h *ProxyHandler) BatchCreate(c *gin.Context) {
	h.BatchCreateGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) BatchCreateGateway(c gatewayctx.GatewayContext) {
	var req BatchCreateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	created := 0
	skipped := 0

	for _, item := range req.Proxies {
		// Trim all string fields
		host := strings.TrimSpace(item.Host)
		protocol := strings.TrimSpace(item.Protocol)
		username := strings.TrimSpace(item.Username)
		password := strings.TrimSpace(item.Password)

		// Check for duplicates (same host, port, username, password)
		exists, err := h.adminService.CheckProxyExists(c.Request().Context(), host, item.Port, username, password)
		if err != nil {
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}

		if exists {
			skipped++
			continue
		}

		// Create proxy with default name
		_, err = h.adminService.CreateProxy(c.Request().Context(), &service.CreateProxyInput{
			Name:     "default",
			Protocol: protocol,
			Host:     host,
			Port:     item.Port,
			Username: username,
			Password: password,
		})
		if err != nil {
			// If creation fails due to duplicate, count as skipped
			skipped++
			continue
		}

		created++
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{
		"created": created,
		"skipped": skipped,
	})
}
