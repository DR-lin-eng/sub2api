package service

import (
	"context"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type GatewaySchedulerRuntimeEntry struct {
	AccountID          int64      `json:"account_id"`
	Platform           string     `json:"platform"`
	Priority           int        `json:"priority"`
	CurrentConcurrency int        `json:"current_concurrency"`
	WaitingCount       int        `json:"waiting_count"`
	LoadRate           int        `json:"load_rate"`
	ReadyTier          int        `json:"ready_tier"`
	TTFTEWMAMs         *float64   `json:"ttft_ewma_ms,omitempty"`
	ErrorRateEWMA      float64    `json:"error_rate_ewma"`
	StickyHits         int64      `json:"sticky_hits"`
	LoadBalanceHits    int64      `json:"load_balance_hits"`
	Switches           int64      `json:"switches"`
	LastSelectedAt     *time.Time `json:"last_selected_at,omitempty"`
}

type GatewaySchedulerRuntimeResponse struct {
	Metrics   GatewayAccountSchedulerMetricsSnapshot `json:"metrics"`
	Transport HTTPUpstreamMetricsSnapshot            `json:"transport"`
	Items     []*GatewaySchedulerRuntimeEntry        `json:"items"`
	Timestamp time.Time                              `json:"timestamp"`
}

type OpenAIWSRuntimeResponse struct {
	AcquireTotal         int64                            `json:"acquire_total"`
	ReuseTotal           int64                            `json:"reuse_total"`
	CreateTotal          int64                            `json:"create_total"`
	QueueWaitMsTotal     int64                            `json:"queue_wait_ms_total"`
	ConnPickMsTotal      int64                            `json:"conn_pick_ms_total"`
	ScaleUpTotal         int64                            `json:"scale_up_total"`
	ScaleDownTotal       int64                            `json:"scale_down_total"`
	PrewarmSuccessTotal  int64                            `json:"prewarm_success_total"`
	PrewarmFallbackTotal int64                            `json:"prewarm_fallback_total"`
	FallbackReasonCounts map[string]int64                 `json:"fallback_reason_counts"`
	Retry                OpenAIWSRetryMetricsSnapshot     `json:"retry"`
	Transport            OpenAIWSTransportMetricsSnapshot `json:"transport"`
	Passthrough          map[string]int64                 `json:"passthrough,omitempty"`
	Relay                OpenAIStreamRelayMetricsSnapshot `json:"relay"`
	Circuits             OpenAICircuitRuntimeSnapshot     `json:"circuits"`
	Timestamp            time.Time                        `json:"timestamp"`
}

func (s *OpsService) GetGatewaySchedulerRuntime(
	ctx context.Context,
	platformFilter string,
	groupIDFilter *int64,
	limit int,
) (*GatewaySchedulerRuntimeResponse, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	if s.gatewayService == nil {
		return nil, infraerrors.ServiceUnavailable("GATEWAY_SERVICE_UNAVAILABLE", "Gateway service not available")
	}

	accounts, err := s.listAllAccountsForOps(ctx, platformFilter)
	if err != nil {
		return nil, err
	}
	if groupIDFilter != nil && *groupIDFilter > 0 {
		filtered := make([]Account, 0, len(accounts))
		for _, acc := range accounts {
			for _, grp := range acc.Groups {
				if grp != nil && grp.ID == *groupIDFilter {
					filtered = append(filtered, acc)
					break
				}
			}
		}
		accounts = filtered
	}

	loadMap := s.getAccountsLoadMapBestEffort(ctx, accounts)
	runtimeStats := s.gatewayService.SnapshotGatewayAccountSchedulerRuntime()
	runtimeByID := make(map[int64]GatewayAccountSchedulerRuntimeSnapshot, len(runtimeStats))
	for _, item := range runtimeStats {
		runtimeByID[item.AccountID] = item
	}

	items := make([]*GatewaySchedulerRuntimeEntry, 0, len(accounts))
	for _, acc := range accounts {
		if acc.ID <= 0 {
			continue
		}
		loadInfo := loadMap[acc.ID]
		if loadInfo == nil {
			loadInfo = &AccountLoadInfo{AccountID: acc.ID}
		}
		readyTier, _, _ := gatewayAccountReadyTier(&acc, loadInfo)
		runtime := runtimeByID[acc.ID]
		items = append(items, &GatewaySchedulerRuntimeEntry{
			AccountID:          acc.ID,
			Platform:           acc.Platform,
			Priority:           acc.Priority,
			CurrentConcurrency: loadInfo.CurrentConcurrency,
			WaitingCount:       loadInfo.WaitingCount,
			LoadRate:           loadInfo.LoadRate,
			ReadyTier:          readyTier,
			TTFTEWMAMs:         runtime.TTFTEWMAMs,
			ErrorRateEWMA:      runtime.ErrorRateEWMA,
			StickyHits:         runtime.StickyHits,
			LoadBalanceHits:    runtime.LoadBalanceHits,
			Switches:           runtime.Switches,
			LastSelectedAt:     runtime.LastSelectedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].LoadRate != items[j].LoadRate {
			return items[i].LoadRate > items[j].LoadRate
		}
		if items[i].WaitingCount != items[j].WaitingCount {
			return items[i].WaitingCount > items[j].WaitingCount
		}
		return items[i].AccountID < items[j].AccountID
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	transport := HTTPUpstreamMetricsSnapshot{}
	if provider, ok := s.gatewayService.httpUpstream.(HTTPUpstreamMetricsProvider); ok {
		transport = provider.SnapshotMetrics()
	}

	return &GatewaySchedulerRuntimeResponse{
		Metrics:   s.gatewayService.SnapshotGatewayAccountSchedulerMetrics(),
		Transport: transport,
		Items:     items,
		Timestamp: time.Now().UTC(),
	}, nil
}

func (s *OpsService) GetOpenAIWSRuntime(
	ctx context.Context,
	platformFilter string,
) (*OpenAIWSRuntimeResponse, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	if s.openAIGatewayService == nil {
		return nil, infraerrors.ServiceUnavailable("OPENAI_GATEWAY_SERVICE_UNAVAILABLE", "OpenAI gateway service not available")
	}
	if platform := strings.TrimSpace(platformFilter); platform != "" && !strings.EqualFold(platform, PlatformOpenAI) {
		return &OpenAIWSRuntimeResponse{
			FallbackReasonCounts: map[string]int64{},
			Passthrough:          map[string]int64{},
			Timestamp:            time.Now().UTC(),
		}, nil
	}

	snapshot := s.openAIGatewayService.SnapshotOpenAIWSPerformanceMetrics()
	return &OpenAIWSRuntimeResponse{
		AcquireTotal:         snapshot.Pool.AcquireTotal,
		ReuseTotal:           snapshot.Pool.AcquireReuseTotal,
		CreateTotal:          snapshot.Pool.AcquireCreateTotal,
		QueueWaitMsTotal:     snapshot.Pool.AcquireQueueWaitMsTotal,
		ConnPickMsTotal:      snapshot.Pool.ConnPickMsTotal,
		ScaleUpTotal:         snapshot.Pool.ScaleUpTotal,
		ScaleDownTotal:       snapshot.Pool.ScaleDownTotal,
		PrewarmSuccessTotal:  snapshot.Retry.PrewarmSuccessTotal,
		PrewarmFallbackTotal: snapshot.Retry.PrewarmFallbackTotal,
		FallbackReasonCounts: snapshot.Retry.FallbackReasonCounts,
		Retry:                snapshot.Retry,
		Transport:            snapshot.Transport,
		Passthrough: map[string]int64{
			"semantic_mutation_total":           snapshot.Passthrough.SemanticMutationTotal,
			"usage_parse_failure_total":         snapshot.Passthrough.UsageParseFailureTotal,
			"incomplete_close_total":            snapshot.Passthrough.IncompleteCloseTotal,
			"stream_closed_after_content_total": snapshot.Passthrough.StreamClosedAfterContentTotal,
		},
		Relay:     snapshot.Relay,
		Circuits:  snapshot.Circuits,
		Timestamp: time.Now().UTC(),
	}, nil
}
