package service

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	defaultOpenAIStreamingConnectQuickFail   = 5 * time.Second
	defaultOpenAIStreamingHeaderQuickFail    = 12 * time.Second
	defaultOpenAIStreamingIdleTimeout        = 120 * time.Second
	defaultOpenAIStreamingLargeBodyThreshold = 64 * 1024
	defaultOpenAIStreamingXLargeThreshold    = 256 * 1024
	defaultOpenAIStreamingHugeThreshold      = 1024 * 1024
	defaultOpenAIHTTPFlushBatchSize          = 8
	defaultOpenAIHTTPFlushInterval           = 25 * time.Millisecond
	defaultOpenAIProxyBreakerCooldown        = 5 * time.Minute
	defaultOpenAIAccountBreakerCooldown      = 2 * time.Minute
	defaultOpenAIRuntimeSyncBatch            = time.Second
	defaultOpenAITempUnscheduleWriteGap      = 5 * time.Second
)

type OpenAIStreamingPhaseBudget struct {
	ConnectBudget    time.Duration `json:"connect_budget"`
	HeaderBudget     time.Duration `json:"header_budget"`
	StreamIdleBudget time.Duration `json:"stream_idle_budget"`
	LargeBodyBytes   int           `json:"large_body_bytes"`
	XLargeBodyBytes  int           `json:"xlarge_body_bytes"`
	HugeBodyBytes    int           `json:"huge_body_bytes"`
}

type OpenAIStreamRelayMetricsSnapshot struct {
	IncompleteCloseTotal         int64 `json:"incomplete_close_total"`
	ClientWriteBlockedTotal      int64 `json:"client_write_blocked_total"`
	FinalFlushFailTotal          int64 `json:"final_flush_fail_total"`
	FirstTokenAfterHeaderMsTotal int64 `json:"first_token_after_header_ms_total"`
	FirstTokenAfterHeaderCount   int64 `json:"first_token_after_header_count"`
	StreamClosedAfterContentTotal int64 `json:"stream_closed_after_content_total"`
}

type openAIStreamRelayMetrics struct {
	incompleteClose         atomic.Int64
	clientWriteBlocked      atomic.Int64
	finalFlushFail          atomic.Int64
	firstTokenAfterHeaderMs atomic.Int64
	firstTokenAfterHeaderCt atomic.Int64
	streamClosedAfterContent atomic.Int64
}

func (m *openAIStreamRelayMetrics) snapshot() OpenAIStreamRelayMetricsSnapshot {
	if m == nil {
		return OpenAIStreamRelayMetricsSnapshot{}
	}
	return OpenAIStreamRelayMetricsSnapshot{
		IncompleteCloseTotal:          m.incompleteClose.Load(),
		ClientWriteBlockedTotal:       m.clientWriteBlocked.Load(),
		FinalFlushFailTotal:           m.finalFlushFail.Load(),
		FirstTokenAfterHeaderMsTotal:  m.firstTokenAfterHeaderMs.Load(),
		FirstTokenAfterHeaderCount:    m.firstTokenAfterHeaderCt.Load(),
		StreamClosedAfterContentTotal: m.streamClosedAfterContent.Load(),
	}
}

func (m *openAIStreamRelayMetrics) recordIncompleteClose(afterContent bool) {
	if m == nil {
		return
	}
	m.incompleteClose.Add(1)
	if afterContent {
		m.streamClosedAfterContent.Add(1)
	}
}

func (m *openAIStreamRelayMetrics) recordClientWriteBlocked() {
	if m != nil {
		m.clientWriteBlocked.Add(1)
	}
}

func (m *openAIStreamRelayMetrics) recordFinalFlushFail() {
	if m != nil {
		m.finalFlushFail.Add(1)
	}
}

func (m *openAIStreamRelayMetrics) recordFirstTokenAfterHeader(ms int64) {
	if m == nil || ms < 0 {
		return
	}
	m.firstTokenAfterHeaderMs.Add(ms)
	m.firstTokenAfterHeaderCt.Add(1)
}

type ProxyCircuitState struct {
	ID        int64      `json:"id"`
	Until     *time.Time `json:"until,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	Failures  int        `json:"failures"`
	AccountID int64      `json:"account_id,omitempty"`
}

type OpenAICircuitRuntimeSnapshot struct {
	OpenProxyCount   int                `json:"open_proxy_count"`
	OpenAccountCount int                `json:"open_account_count"`
	Proxies          []ProxyCircuitState `json:"proxies,omitempty"`
	Accounts         []ProxyCircuitState `json:"accounts,omitempty"`
}

type openAICircuitEntry struct {
	Failures  int
	Until     time.Time
	Reason    string
	AccountID int64
}

type openAICircuitBreaker struct {
	threshold int
	cooldown  time.Duration
	states    sync.Map
}

func newOpenAICircuitBreaker(threshold int, cooldown time.Duration) *openAICircuitBreaker {
	if threshold <= 0 {
		threshold = 1
	}
	if cooldown <= 0 {
		cooldown = time.Second
	}
	return &openAICircuitBreaker{
		threshold: threshold,
		cooldown:  cooldown,
	}
}

func (b *openAICircuitBreaker) isOpen(id int64, now time.Time) bool {
	if b == nil || id <= 0 {
		return false
	}
	value, ok := b.states.Load(id)
	if !ok {
		return false
	}
	entry, _ := value.(openAICircuitEntry)
	if entry.Until.IsZero() || now.After(entry.Until) {
		if entry.Until.IsZero() || now.Sub(entry.Until) > b.cooldown {
			b.states.Delete(id)
		}
		return false
	}
	return true
}

func (b *openAICircuitBreaker) reset(id int64) {
	if b != nil && id > 0 {
		b.states.Delete(id)
	}
}

func (b *openAICircuitBreaker) trip(id int64, reason string, cooldown time.Duration, accountID int64) {
	if b == nil || id <= 0 {
		return
	}
	if cooldown <= 0 {
		cooldown = b.cooldown
	}
	b.states.Store(id, openAICircuitEntry{
		Failures:  b.threshold,
		Until:     time.Now().Add(cooldown),
		Reason:    strings.TrimSpace(reason),
		AccountID: accountID,
	})
}

func (b *openAICircuitBreaker) recordFailure(id int64, reason string, cooldown time.Duration, accountID int64, immediate bool) bool {
	if b == nil || id <= 0 {
		return false
	}
	now := time.Now()
	value, _ := b.states.Load(id)
	entry, _ := value.(openAICircuitEntry)
	if immediate {
		entry.Failures = b.threshold
	} else {
		entry.Failures++
	}
	entry.AccountID = accountID
	entry.Reason = strings.TrimSpace(reason)
	if cooldown <= 0 {
		cooldown = b.cooldown
	}
	if entry.Failures >= b.threshold {
		entry.Until = now.Add(cooldown)
		b.states.Store(id, entry)
		return true
	}
	b.states.Store(id, entry)
	return false
}

func (b *openAICircuitBreaker) snapshot(limit int) []ProxyCircuitState {
	if b == nil {
		return nil
	}
	now := time.Now()
	items := make([]ProxyCircuitState, 0)
	b.states.Range(func(key, value any) bool {
		id, _ := key.(int64)
		entry, _ := value.(openAICircuitEntry)
		if id <= 0 || entry.Until.IsZero() || !now.Before(entry.Until) {
			return true
		}
		until := entry.Until
		items = append(items, ProxyCircuitState{
			ID:        id,
			Until:     &until,
			Reason:    entry.Reason,
			Failures:  entry.Failures,
			AccountID: entry.AccountID,
		})
		return true
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func openAITempUnscheduleWriteThrottle(cfg *config.Config) *accountWriteThrottle {
	gap := defaultOpenAITempUnscheduleWriteGap
	if cfg != nil && cfg.Gateway.Scheduling.RuntimeSyncBatchMS > 0 {
		candidate := time.Duration(cfg.Gateway.Scheduling.RuntimeSyncBatchMS*5) * time.Millisecond
		if candidate > gap {
			gap = candidate
		}
	}
	return newAccountWriteThrottle(gap)
}

func resolveOpenAIProxyBreakerThreshold(cfg *config.Config) int {
	if cfg != nil && cfg.Gateway.OpenAI.ProxyCircuitBreaker.FailureThreshold > 0 {
		return cfg.Gateway.OpenAI.ProxyCircuitBreaker.FailureThreshold
	}
	return 2
}

func resolveOpenAIProxyBreakerCooldown(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.Gateway.OpenAI.ProxyCircuitBreaker.CooldownMS > 0 {
		return time.Duration(cfg.Gateway.OpenAI.ProxyCircuitBreaker.CooldownMS) * time.Millisecond
	}
	return defaultOpenAIProxyBreakerCooldown
}

func resolveOpenAIAccountBreakerCooldown(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.Gateway.OpenAI.AccountCircuitBreaker.CooldownMS > 0 {
		return time.Duration(cfg.Gateway.OpenAI.AccountCircuitBreaker.CooldownMS) * time.Millisecond
	}
	return defaultOpenAIAccountBreakerCooldown
}

func (s *OpenAIGatewayService) openAIStreamingConfig() config.GatewayOpenAIStreamingConfig {
	if s != nil && s.cfg != nil {
		return s.cfg.Gateway.OpenAI.Streaming
	}
	return config.GatewayOpenAIStreamingConfig{}
}

func (s *OpenAIGatewayService) openAIStreamingPhaseBudget() OpenAIStreamingPhaseBudget {
	cfg := s.openAIStreamingConfig()
	budget := OpenAIStreamingPhaseBudget{
		ConnectBudget:    defaultOpenAIStreamingConnectQuickFail,
		HeaderBudget:     defaultOpenAIStreamingHeaderQuickFail,
		StreamIdleBudget: defaultOpenAIStreamingIdleTimeout,
		LargeBodyBytes:   defaultOpenAIStreamingLargeBodyThreshold,
		XLargeBodyBytes:  defaultOpenAIStreamingXLargeThreshold,
		HugeBodyBytes:    defaultOpenAIStreamingHugeThreshold,
	}
	if cfg.ConnectQuickFailMS > 0 {
		budget.ConnectBudget = time.Duration(cfg.ConnectQuickFailMS) * time.Millisecond
	}
	if cfg.HeaderQuickFailMS > 0 {
		budget.HeaderBudget = time.Duration(cfg.HeaderQuickFailMS) * time.Millisecond
	}
	if cfg.StreamIdleTimeoutMS > 0 {
		budget.StreamIdleBudget = time.Duration(cfg.StreamIdleTimeoutMS) * time.Millisecond
	}
	if cfg.LargeBodyThresholdBytes > 0 {
		budget.LargeBodyBytes = cfg.LargeBodyThresholdBytes
	}
	if cfg.XLargeBodyThresholdBytes > 0 {
		budget.XLargeBodyBytes = cfg.XLargeBodyThresholdBytes
	}
	if cfg.HugeBodyThresholdBytes > 0 {
		budget.HugeBodyBytes = cfg.HugeBodyThresholdBytes
	}
	return budget
}

func (s *OpenAIGatewayService) applyOpenAITransportOverride(req *http.Request, body []byte, reqStream bool) *http.Request {
	if req == nil {
		return req
	}
	if !reqStream {
		return req
	}
	budget := s.openAIStreamingPhaseBudget()
	override := UpstreamTransportOverride{
		DialTimeout: budget.ConnectBudget,
	}
	bodySize := len(body)
	switch {
	case bodySize >= budget.HugeBodyBytes && budget.HugeBodyBytes > 0:
		override.ResponseHeaderTimeout = 0
	case bodySize >= budget.XLargeBodyBytes && budget.XLargeBodyBytes > 0:
		override.ResponseHeaderTimeout = budget.HeaderBudget + 15*time.Second
	case bodySize >= budget.LargeBodyBytes && budget.LargeBodyBytes > 0:
		override.ResponseHeaderTimeout = budget.HeaderBudget + 5*time.Second
	default:
		override.ResponseHeaderTimeout = budget.HeaderBudget
	}
	ctx := WithUpstreamTransportOverride(req.Context(), override)
	return req.WithContext(ctx)
}

func (s *OpenAIGatewayService) openAIStreamIdleTimeout() time.Duration {
	budget := s.openAIStreamingPhaseBudget()
	if budget.StreamIdleBudget > 0 {
		return budget.StreamIdleBudget
	}
	if s != nil && s.cfg != nil && s.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		return time.Duration(s.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	return 0
}

func (s *OpenAIGatewayService) openAIHTTPFlushBatchSize() int {
	cfg := s.openAIStreamingConfig()
	if cfg.HTTPStreamFlushBatchSize > 0 {
		return cfg.HTTPStreamFlushBatchSize
	}
	return defaultOpenAIHTTPFlushBatchSize
}

func (s *OpenAIGatewayService) openAIHTTPFlushInterval() time.Duration {
	cfg := s.openAIStreamingConfig()
	if cfg.HTTPStreamFlushIntervalMS > 0 {
		return time.Duration(cfg.HTTPStreamFlushIntervalMS) * time.Millisecond
	}
	return defaultOpenAIHTTPFlushInterval
}

func (s *OpenAIGatewayService) queueOpenAIRuntimeStateSync(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	if s.schedulerSnapshot == nil {
		return
	}
	if s.runtimeSyncWake == nil {
		s.syncAccountRuntimeStateToSchedulerCache(context.Background(), accountID)
		return
	}
	s.runtimeSyncMu.Lock()
	if s.runtimeSyncPending == nil {
		s.runtimeSyncPending = make(map[int64]struct{})
	}
	s.runtimeSyncPending[accountID] = struct{}{}
	s.runtimeSyncMu.Unlock()
	select {
	case s.runtimeSyncWake <- struct{}{}:
	default:
	}
}

func (s *OpenAIGatewayService) flushQueuedOpenAIRuntimeStateSync(ctx context.Context) {
	if s == nil || s.schedulerSnapshot == nil {
		return
	}
	s.runtimeSyncMu.Lock()
	pending := s.runtimeSyncPending
	s.runtimeSyncPending = make(map[int64]struct{}, len(pending))
	s.runtimeSyncMu.Unlock()
	for accountID := range pending {
		s.syncAccountRuntimeStateToSchedulerCache(ctx, accountID)
	}
}

func (s *OpenAIGatewayService) startOpenAIRuntimeSyncWorker() {
	if s == nil || s.schedulerSnapshot == nil || s.runtimeSyncWake == nil {
		return
	}
	interval := defaultOpenAIRuntimeSyncBatch
	if s.cfg != nil && s.cfg.Gateway.Scheduling.RuntimeSyncBatchMS > 0 {
		interval = time.Duration(s.cfg.Gateway.Scheduling.RuntimeSyncBatchMS) * time.Millisecond
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.runtimeSyncStop:
				s.flushQueuedOpenAIRuntimeStateSync(context.Background())
				return
			case <-s.runtimeSyncWake:
			case <-ticker.C:
			}
			s.flushQueuedOpenAIRuntimeStateSync(context.Background())
		}
	}()
}

func (s *OpenAIGatewayService) openAIProxyBreakerCooldown() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAI.ProxyCircuitBreaker.CooldownMS > 0 {
		return time.Duration(s.cfg.Gateway.OpenAI.ProxyCircuitBreaker.CooldownMS) * time.Millisecond
	}
	return defaultOpenAIProxyBreakerCooldown
}

func (s *OpenAIGatewayService) openAIAccountBreakerCooldown() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAI.AccountCircuitBreaker.CooldownMS > 0 {
		return time.Duration(s.cfg.Gateway.OpenAI.AccountCircuitBreaker.CooldownMS) * time.Millisecond
	}
	return defaultOpenAIAccountBreakerCooldown
}

func (s *OpenAIGatewayService) snapshotOpenAICircuitRuntime(limit int) OpenAICircuitRuntimeSnapshot {
	snapshot := OpenAICircuitRuntimeSnapshot{}
	if s == nil {
		return snapshot
	}
	if s.proxyCircuit != nil {
		snapshot.Proxies = s.proxyCircuit.snapshot(limit)
		snapshot.OpenProxyCount = len(snapshot.Proxies)
	}
	if s.accountCircuit != nil {
		snapshot.Accounts = s.accountCircuit.snapshot(limit)
		snapshot.OpenAccountCount = len(snapshot.Accounts)
	}
	return snapshot
}

func (s *OpenAIGatewayService) isOpenAICircuitBlocked(account *Account) bool {
	if s == nil || account == nil {
		return false
	}
	now := time.Now()
	if s.accountCircuit != nil && s.accountCircuit.isOpen(account.ID, now) {
		return true
	}
	if account.ProxyID != nil && *account.ProxyID > 0 && s.proxyCircuit != nil && s.proxyCircuit.isOpen(*account.ProxyID, now) {
		return true
	}
	return false
}

func (s *OpenAIGatewayService) recordOpenAISuccessCircuitState(account *Account) {
	if s == nil || account == nil {
		return
	}
	if s.accountCircuit != nil {
		s.accountCircuit.reset(account.ID)
	}
}

func (s *OpenAIGatewayService) registerOpenAIRuntimeFailure(account *Account, failoverErr *UpstreamFailoverError) {
	if s == nil || account == nil || failoverErr == nil {
		return
	}
	reason := strings.ToLower(strings.TrimSpace(failoverErr.TempUnscheduleReason))
	bodyText := strings.ToLower(strings.TrimSpace(string(failoverErr.ResponseBody)))
	if failoverErr.StatusCode == http.StatusUnauthorized || strings.Contains(bodyText, "token_invalidated") {
		if s.accountCircuit != nil {
			s.accountCircuit.recordFailure(account.ID, "token_invalidated", maxDuration(failoverErr.TempUnscheduleFor, 20*time.Minute), account.ID, true)
		}
		return
	}
	if failoverErr.FailedProxyID > 0 || strings.Contains(reason, "proxy") || strings.Contains(reason, "network") || strings.Contains(bodyText, "connection refused") || strings.Contains(bodyText, "connection reset") || strings.Contains(bodyText, "socks connect") || strings.Contains(bodyText, "eof") {
		if failoverErr.FailedProxyID > 0 && s.proxyCircuit != nil {
			s.proxyCircuit.recordFailure(failoverErr.FailedProxyID, "proxy/network failure", s.openAIProxyBreakerCooldown(), account.ID, false)
		}
		if s.accountCircuit != nil {
			s.accountCircuit.recordFailure(account.ID, "proxy/network failure", maxDuration(failoverErr.TempUnscheduleFor, s.openAIAccountBreakerCooldown()), account.ID, true)
		}
		return
	}
	if strings.Contains(bodyText, "context deadline exceeded") || strings.Contains(reason, "timeout") {
		if s.accountCircuit != nil {
			s.accountCircuit.recordFailure(account.ID, "header timeout", maxDuration(failoverErr.TempUnscheduleFor, s.openAIAccountBreakerCooldown()), account.ID, true)
		}
	}
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
