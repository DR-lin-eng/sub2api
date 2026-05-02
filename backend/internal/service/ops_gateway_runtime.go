package service

import (
	"context"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/runtimeobs"
	ffi "github.com/Wei-Shaw/sub2api/internal/rustbridge/ffi"
	rustsidecar "github.com/Wei-Shaw/sub2api/internal/rustbridge/sidecar"
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
	Metrics                  GatewayAccountSchedulerMetricsSnapshot `json:"metrics"`
	Transport                HTTPUpstreamMetricsSnapshot            `json:"transport"`
	ActiveSchedulingAccounts int                                    `json:"active_scheduling_accounts"`
	PoolAccountsTotal        int                                    `json:"pool_accounts_total"`
	Items                    []*GatewaySchedulerRuntimeEntry        `json:"items"`
	Timestamp                time.Time                              `json:"timestamp"`
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
	GnetHTTP1            runtimeobs.GnetHTTP1Snapshot     `json:"gnet_http1"`
	RustFFI              ffi.MetricsSnapshot              `json:"rust_ffi"`
	RustSidecar          *RustSidecarRuntimeResponse      `json:"rust_sidecar,omitempty"`
	Circuits             OpenAICircuitRuntimeSnapshot     `json:"circuits"`
	Timestamp            time.Time                        `json:"timestamp"`
}

type RustSidecarRuntimeResponse struct {
	Enabled                         bool   `json:"enabled"`
	Available                       bool   `json:"available"`
	Status                          string `json:"status,omitempty"`
	Service                         string `json:"service,omitempty"`
	Version                         string `json:"version,omitempty"`
	ActiveConnections               int64  `json:"active_connections,omitempty"`
	TotalConnections                int64  `json:"total_connections,omitempty"`
	ActiveUpgrades                  int64  `json:"active_upgrades,omitempty"`
	TotalUpgrades                   int64  `json:"total_upgrades,omitempty"`
	TotalRequests                   int64  `json:"total_requests,omitempty"`
	TotalRequestErrors              int64  `json:"total_request_errors,omitempty"`
	UpstreamUnavailableTotal        int64  `json:"upstream_unavailable_total,omitempty"`
	UpstreamHandshakeFailedTotal    int64  `json:"upstream_handshake_failed_total,omitempty"`
	UpstreamRequestFailedTotal      int64  `json:"upstream_request_failed_total,omitempty"`
	UpgradeErrorsTotal              int64  `json:"upgrade_errors_total,omitempty"`
	RelayBytesDownstreamToUpstream  int64  `json:"relay_bytes_downstream_to_upstream,omitempty"`
	RelayBytesUpstreamToDownstream  int64  `json:"relay_bytes_upstream_to_downstream,omitempty"`
	RelayFramesDownstreamToUpstream int64  `json:"relay_frames_downstream_to_upstream,omitempty"`
	RelayFramesUpstreamToDownstream int64  `json:"relay_frames_upstream_to_downstream,omitempty"`
	RelayCloseFramesTotal           int64  `json:"relay_close_frames_total,omitempty"`
	RelayPingFramesTotal            int64  `json:"relay_ping_frames_total,omitempty"`
	RelayPongFramesTotal            int64  `json:"relay_pong_frames_total,omitempty"`
	Error                           string `json:"error,omitempty"`
}

func summarizeGatewaySchedulerPool(accounts []Account, loadMap map[int64]*AccountLoadInfo) (int, int) {
	poolAccountsTotal := 0
	activeSchedulingAccounts := 0
	for _, acc := range accounts {
		if acc.ID <= 0 {
			continue
		}
		poolAccountsTotal++
		loadInfo := loadMap[acc.ID]
		if loadInfo != nil && (loadInfo.CurrentConcurrency > 0 || loadInfo.WaitingCount > 0) {
			activeSchedulingAccounts++
		}
	}
	return poolAccountsTotal, activeSchedulingAccounts
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

	accounts, err := s.listAllAccountsForOpsCached(ctx, platformFilter)
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
	poolAccountsTotal, activeSchedulingAccounts := summarizeGatewaySchedulerPool(accounts, loadMap)
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
		Metrics:                  s.gatewayService.SnapshotGatewayAccountSchedulerMetrics(),
		Transport:                transport,
		ActiveSchedulingAccounts: activeSchedulingAccounts,
		PoolAccountsTotal:        poolAccountsTotal,
		Items:                    items,
		Timestamp:                time.Now().UTC(),
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
		GnetHTTP1:   runtimeobs.SnapshotGnetHTTP1(),
		RustFFI:     snapshot.RustFFI,
		RustSidecar: s.getRustSidecarRuntime(ctx),
		Relay:       snapshot.Relay,
		Circuits:    snapshot.Circuits,
		Timestamp:   time.Now().UTC(),
	}, nil
}

func (s *OpsService) getRustSidecarRuntime(ctx context.Context) *RustSidecarRuntimeResponse {
	resp := &RustSidecarRuntimeResponse{}
	if s == nil || s.cfg == nil {
		return resp
	}
	resp.Enabled = s.cfg.Rust.Sidecar.Enabled
	if !resp.Enabled {
		return resp
	}
	if cached, ok := s.rustSidecarHealthCache.Load().(*opsCachedRustSidecarHealth); ok && cached != nil && time.Now().UnixNano() < cached.ExpiresAt && cached.Value != nil {
		cloned := *cached.Value
		return &cloned
	}
	value, _, _ := s.rustSidecarHealthSF.Do("rust_sidecar_health", func() (any, error) {
		next := &RustSidecarRuntimeResponse{Enabled: true}
		client, err := rustsidecar.NewClient(s.cfg.Rust.Sidecar)
		if err != nil {
			next.Error = err.Error()
			s.rustSidecarHealthCache.Store(&opsCachedRustSidecarHealth{
				Value:     next,
				ExpiresAt: time.Now().Add(opsRustSidecarHealthTTL).UnixNano(),
			})
			return next, nil
		}
		timeout := time.Duration(s.cfg.Rust.Sidecar.HealthcheckTimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 3 * time.Second
		}
		if ctx == nil {
			ctx = context.Background()
		}
		healthCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		health, err := client.Health(healthCtx)
		if err != nil {
			next.Error = err.Error()
			s.rustSidecarHealthCache.Store(&opsCachedRustSidecarHealth{
				Value:     next,
				ExpiresAt: time.Now().Add(opsRustSidecarHealthTTL).UnixNano(),
			})
			return next, nil
		}
		next.Available = true
		next.Status = health.Status
		next.Service = health.Service
		next.Version = health.Version
		next.ActiveConnections = health.ActiveConnections
		next.TotalConnections = health.TotalConnections
		next.ActiveUpgrades = health.ActiveUpgrades
		next.TotalUpgrades = health.TotalUpgrades
		next.TotalRequests = health.TotalRequests
		next.TotalRequestErrors = health.TotalRequestErrors
		next.UpstreamUnavailableTotal = health.UpstreamUnavailableTotal
		next.UpstreamHandshakeFailedTotal = health.UpstreamHandshakeFailedTotal
		next.UpstreamRequestFailedTotal = health.UpstreamRequestFailedTotal
		next.UpgradeErrorsTotal = health.UpgradeErrorsTotal
		next.RelayBytesDownstreamToUpstream = health.RelayBytesDownstreamToUpstream
		next.RelayBytesUpstreamToDownstream = health.RelayBytesUpstreamToDownstream
		next.RelayFramesDownstreamToUpstream = health.RelayFramesDownstreamToUpstream
		next.RelayFramesUpstreamToDownstream = health.RelayFramesUpstreamToDownstream
		next.RelayCloseFramesTotal = health.RelayCloseFramesTotal
		next.RelayPingFramesTotal = health.RelayPingFramesTotal
		next.RelayPongFramesTotal = health.RelayPongFramesTotal
		s.rustSidecarHealthCache.Store(&opsCachedRustSidecarHealth{
			Value:     next,
			ExpiresAt: time.Now().Add(opsRustSidecarHealthTTL).UnixNano(),
		})
		return next, nil
	})
	if cachedResp, ok := value.(*RustSidecarRuntimeResponse); ok && cachedResp != nil {
		cloned := *cachedResp
		return &cloned
	}
	return resp
}
