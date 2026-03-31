package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ExportData exports proxy-only data for migration.
func (h *ProxyHandler) ExportData(c *gin.Context) {
	h.ExportDataGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) ExportDataGateway(c gatewayctx.GatewayContext) {
	ctx := c.Request().Context()

	selectedIDs, err := parseProxyIDsRequest(c.Request())
	if err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, 400, err.Error())
		return
	}

	var proxies []service.Proxy
	if len(selectedIDs) > 0 {
		proxies, err = h.getProxiesByIDs(ctx, selectedIDs)
		if err != nil {
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}
	} else {
		protocol := c.QueryValue("protocol")
		status := c.QueryValue("status")
		search := strings.TrimSpace(c.QueryValue("search"))
		if len(search) > 100 {
			search = search[:100]
		}

		proxies, err = h.listProxiesFiltered(ctx, protocol, status, search)
		if err != nil {
			response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
			return
		}
	}

	dataProxies := make([]DataProxy, 0, len(proxies))
	for i := range proxies {
		p := proxies[i]
		key := buildProxyKey(p.Protocol, p.Host, p.Port, p.Username, p.Password)
		dataProxies = append(dataProxies, DataProxy{
			ProxyKey: key,
			Name:     p.Name,
			Protocol: p.Protocol,
			Host:     p.Host,
			Port:     p.Port,
			Username: p.Username,
			Password: p.Password,
			Status:   p.Status,
		})
	}

	payload := DataPayload{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Proxies:    dataProxies,
		Accounts:   []DataAccount{},
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, payload)
}

// ImportData imports proxy-only data for migration.
func (h *ProxyHandler) ImportData(c *gin.Context) {
	h.ImportDataGateway(gatewayctx.FromGin(c))
}

func (h *ProxyHandler) ImportDataGateway(c gatewayctx.GatewayContext) {
	type ProxyImportRequest struct {
		Data DataPayload `json:"data"`
	}

	var req ProxyImportRequest
	contentType := ""
	if req0 := c.Request(); req0 != nil {
		contentType = req0.Header.Get("Content-Type")
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/form-data") {
		payload, err := parseImportPayloadFromMultipartRequest(c.Request())
		if err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, 400, err.Error())
			return
		}
		req.Data = payload
	} else {
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			response.ErrorContext(gatewayJSONResponder{ctx: c}, 400, "Invalid request: "+err.Error())
			return
		}
	}

	if err := validateDataHeader(req.Data); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, 400, err.Error())
		return
	}

	ctx := c.Request().Context()
	result := DataImportResult{}

	existingProxies, err := h.listProxiesFiltered(ctx, "", "", "")
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	proxyByKey := make(map[string]service.Proxy, len(existingProxies))
	for i := range existingProxies {
		p := existingProxies[i]
		key := buildProxyKey(p.Protocol, p.Host, p.Port, p.Username, p.Password)
		proxyByKey[key] = p
	}

	latencyProbeIDs := make([]int64, 0, len(req.Data.Proxies))
	for i := range req.Data.Proxies {
		item := req.Data.Proxies[i]
		key := item.ProxyKey
		if key == "" {
			key = buildProxyKey(item.Protocol, item.Host, item.Port, item.Username, item.Password)
		}

		if err := validateDataProxy(item); err != nil {
			result.ProxyFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:     "proxy",
				Name:     item.Name,
				ProxyKey: key,
				Message:  err.Error(),
			})
			continue
		}

		normalizedStatus := normalizeProxyStatus(item.Status)
		if existing, ok := proxyByKey[key]; ok {
			result.ProxyReused++
			if normalizedStatus != "" && normalizedStatus != existing.Status {
				if _, err := h.adminService.UpdateProxy(ctx, existing.ID, &service.UpdateProxyInput{Status: normalizedStatus}); err != nil {
					result.Errors = append(result.Errors, DataImportError{
						Kind:     "proxy",
						Name:     item.Name,
						ProxyKey: key,
						Message:  "update status failed: " + err.Error(),
					})
				}
			}
			latencyProbeIDs = append(latencyProbeIDs, existing.ID)
			continue
		}

		created, err := h.adminService.CreateProxy(ctx, &service.CreateProxyInput{
			Name:     defaultProxyName(item.Name),
			Protocol: item.Protocol,
			Host:     item.Host,
			Port:     item.Port,
			Username: item.Username,
			Password: item.Password,
		})
		if err != nil {
			result.ProxyFailed++
			result.Errors = append(result.Errors, DataImportError{
				Kind:     "proxy",
				Name:     item.Name,
				ProxyKey: key,
				Message:  err.Error(),
			})
			continue
		}
		result.ProxyCreated++
		proxyByKey[key] = *created

		if normalizedStatus != "" && normalizedStatus != created.Status {
			if _, err := h.adminService.UpdateProxy(ctx, created.ID, &service.UpdateProxyInput{Status: normalizedStatus}); err != nil {
				result.Errors = append(result.Errors, DataImportError{
					Kind:     "proxy",
					Name:     item.Name,
					ProxyKey: key,
					Message:  "update status failed: " + err.Error(),
				})
			}
		}
		// CreateProxy already triggers a latency probe, avoid double probing here.
	}

	if len(latencyProbeIDs) > 0 {
		ids := append([]int64(nil), latencyProbeIDs...)
		go func() {
			for _, id := range ids {
				_, _ = h.adminService.TestProxy(context.Background(), id)
			}
		}()
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, result)
}

func (h *ProxyHandler) getProxiesByIDs(ctx context.Context, ids []int64) ([]service.Proxy, error) {
	if len(ids) == 0 {
		return []service.Proxy{}, nil
	}
	return h.adminService.GetProxiesByIDs(ctx, ids)
}

func parseProxyIDs(c *gin.Context) ([]int64, error) {
	values := c.QueryArray("ids")
	if len(values) == 0 {
		raw := strings.TrimSpace(c.Query("ids"))
		if raw != "" {
			values = []string{raw}
		}
	}
	if len(values) == 0 {
		return nil, nil
	}

	ids := make([]int64, 0, len(values))
	for _, item := range values {
		for _, part := range strings.Split(item, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("invalid proxy id: %s", part)
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func parseProxyIDsRequest(req *http.Request) ([]int64, error) {
	if req == nil || req.URL == nil {
		return nil, nil
	}
	values := req.URL.Query()["ids"]
	if len(values) == 0 {
		raw := strings.TrimSpace(req.URL.Query().Get("ids"))
		if raw != "" {
			values = []string{raw}
		}
	}
	if len(values) == 0 {
		return nil, nil
	}
	ids := make([]int64, 0, len(values))
	for _, item := range values {
		for _, part := range strings.Split(item, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("invalid proxy id: %s", part)
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (h *ProxyHandler) listProxiesFiltered(ctx context.Context, protocol, status, search string) ([]service.Proxy, error) {
	page := 1
	pageSize := pagination.PaginationParams{Page: 1, PageSize: dataPageCap}.Limit()
	var out []service.Proxy
	for {
		items, total, err := h.adminService.ListProxies(ctx, page, pageSize, protocol, status, search)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(out) >= int(total) || len(items) == 0 {
			break
		}
		page++
	}
	return out, nil
}
