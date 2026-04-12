package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type dashboardUsageRepoStubForAdminStats struct {
	service.UsageLogRepository
	stats *usagestats.DashboardStats
	err   error
}

func (s *dashboardUsageRepoStubForAdminStats) GetDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.stats, nil
}

func TestProxyHandler_GetStats_ReturnsRealCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	latency := int64(123)
	score := 87
	adminSvc.proxyCounts = []service.ProxyWithAccountCount{{
		Proxy:         service.Proxy{ID: 4, Name: "proxy", Status: service.StatusActive},
		AccountCount:  2,
		LatencyMs:     &latency,
		QualityScore:  &score,
		QualityStatus: "healthy",
	}}

	handler := NewProxyHandler(adminSvc)
	router := gin.New()
	router.GET("/api/v1/admin/proxies/:id/stats", handler.GetStats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/4/stats", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.total_accounts").Int(); got != 1 {
		t.Fatalf("expected total_accounts=1, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.active_accounts").Int(); got != 1 {
		t.Fatalf("expected active_accounts=1, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.total_requests").Int(); got != 12 {
		t.Fatalf("expected total_requests=12, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.average_latency").Int(); got != 123 {
		t.Fatalf("expected average_latency=123, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.success_rate").Float(); got != 87 {
		t.Fatalf("expected success_rate=87, got=%f body=%s", got, rec.Body.String())
	}
}

func TestRedeemHandler_GetStats_ReturnsRealAggregates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.redeems = []service.RedeemCode{
		{ID: 1, Type: service.RedeemTypeBalance, Status: service.StatusUnused, Value: 10},
		{ID: 2, Type: service.RedeemTypeConcurrency, Status: service.StatusUsed, Value: 20},
		{ID: 3, Type: service.RedeemTypeSubscription, Status: service.StatusExpired, Value: 30},
	}

	handler := NewRedeemHandler(adminSvc, nil)
	router := gin.New()
	router.GET("/api/v1/admin/redeem-codes/stats", handler.GetStats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/redeem-codes/stats", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.total_codes").Int(); got != 3 {
		t.Fatalf("expected total_codes=3, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.active_codes").Int(); got != 1 {
		t.Fatalf("expected active_codes=1, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.used_codes").Int(); got != 1 {
		t.Fatalf("expected used_codes=1, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.expired_codes").Int(); got != 1 {
		t.Fatalf("expected expired_codes=1, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.total_value_distributed").Float(); got != 20 {
		t.Fatalf("expected total_value_distributed=20, got=%f body=%s", got, rec.Body.String())
	}
}

func TestDashboardHandler_GetRealtimeMetrics_UsesDashboardStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dashboardSvc := service.NewDashboardService(&dashboardUsageRepoStubForAdminStats{
		stats: &usagestats.DashboardStats{
			Rpm:               42,
			AverageDurationMs: 321,
		},
	}, nil, nil, nil)

	handler := NewDashboardHandler(dashboardSvc, nil)
	router := gin.New()
	router.GET("/api/v1/admin/dashboard/realtime", handler.GetRealtimeMetrics)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/realtime", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.requests_per_minute").Float(); got != 42 {
		t.Fatalf("expected requests_per_minute=42, got=%f body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.average_response_time").Float(); got != 321 {
		t.Fatalf("expected average_response_time=321, got=%f body=%s", got, rec.Body.String())
	}
}
