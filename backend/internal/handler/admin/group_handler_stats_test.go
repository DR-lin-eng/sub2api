package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type groupStatsUsageRepoStub struct {
	service.UsageLogRepository
	stats []usagestats.GroupStat
	err   error
}

func (s *groupStatsUsageRepoStub) GetGroupStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, requestType *int16, stream *bool, billingType *int8) ([]usagestats.GroupStat, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.stats, nil
}

func TestGroupHandler_GetStats_ReturnsRealCountsAndUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.apiKeys = []service.APIKey{
		{ID: 1, GroupID: ptrInt64ForGroupStats(2), Status: service.StatusActive},
		{ID: 2, GroupID: ptrInt64ForGroupStats(2), Status: "inactive"},
		{ID: 3, GroupID: ptrInt64ForGroupStats(2), Status: service.StatusActive},
	}

	dashboardSvc := service.NewDashboardService(&groupStatsUsageRepoStub{
		stats: []usagestats.GroupStat{
			{GroupID: 2, GroupName: "group", Requests: 11, Cost: 3.5},
		},
	}, nil, nil, nil)

	handler := NewGroupHandler(adminSvc, dashboardSvc, nil)
	router := gin.New()
	router.GET("/api/v1/admin/groups/:id/stats", handler.GetStats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/2/stats", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if gjson.Get(rec.Body.String(), "code").Int() != 0 {
		t.Fatalf("expected success envelope, got body=%s", rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.total_api_keys").Int(); got != 3 {
		t.Fatalf("expected total_api_keys=3, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.active_api_keys").Int(); got != 2 {
		t.Fatalf("expected active_api_keys=2, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.total_requests").Int(); got != 11 {
		t.Fatalf("expected total_requests=11, got=%d body=%s", got, rec.Body.String())
	}
	if got := gjson.Get(rec.Body.String(), "data.total_cost").Float(); got != 3.5 {
		t.Fatalf("expected total_cost=3.5, got=%f body=%s", got, rec.Body.String())
	}
}

func ptrInt64ForGroupStats(v int64) *int64 { return &v }
