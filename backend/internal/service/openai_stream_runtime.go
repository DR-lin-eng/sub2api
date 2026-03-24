package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
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

type openAIHealthPrefetchJob struct {
	AccountID      int64
	RequestedModel string
}

type openAIHealthPrefetchState struct {
	inFlight atomic.Bool
	lastAt   atomic.Int64
}

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

func (s *OpenAIGatewayService) openAIHealthPrefetchConfig() config.GatewayOpenAIHealthPrefetchConfig {
	if s != nil && s.cfg != nil {
		return s.cfg.Gateway.OpenAI.HealthPrefetch
	}
	return config.GatewayOpenAIHealthPrefetchConfig{}
}

func (s *OpenAIGatewayService) openAIHealthPrefetchEnabled() bool {
	cfg := s.openAIHealthPrefetchConfig()
	return cfg.Enabled && cfg.TopN > 0 && cfg.WorkerCount > 0 && cfg.QueueSize > 0
}

func (s *OpenAIGatewayService) openAIHealthPrefetchTopN() int {
	cfg := s.openAIHealthPrefetchConfig()
	if cfg.TopN <= 0 {
		return 0
	}
	if cfg.TopN < 3 {
		return 3
	}
	if cfg.TopN > 10 {
		return 10
	}
	return cfg.TopN
}

func (s *OpenAIGatewayService) openAIHealthPrefetchCooldown() time.Duration {
	cfg := s.openAIHealthPrefetchConfig()
	if cfg.CooldownSeconds > 0 {
		return time.Duration(cfg.CooldownSeconds) * time.Second
	}
	return time.Minute
}

func (s *OpenAIGatewayService) openAIHealthPrefetchTimeout() time.Duration {
	cfg := s.openAIHealthPrefetchConfig()
	if cfg.TimeoutMS > 0 {
		return time.Duration(cfg.TimeoutMS) * time.Millisecond
	}
	return 3500 * time.Millisecond
}

func (s *OpenAIGatewayService) startOpenAIHealthPrefetchWorker() {
	if s == nil || !s.openAIHealthPrefetchEnabled() {
		return
	}
	cfg := s.openAIHealthPrefetchConfig()
	s.healthPrefetchCh = make(chan openAIHealthPrefetchJob, cfg.QueueSize)
	workerCount := cfg.WorkerCount
	if workerCount <= 0 {
		workerCount = 1
	}
	for i := 0; i < workerCount; i++ {
		s.healthPrefetchWG.Add(1)
		go func() {
			defer s.healthPrefetchWG.Done()
			for {
				select {
				case <-s.healthPrefetchStop:
					return
				case job, ok := <-s.healthPrefetchCh:
					if !ok {
						return
					}
					s.runOpenAIHealthPrefetchJob(job)
				}
			}
		}()
	}
}

func (s *OpenAIGatewayService) enqueueOpenAIHealthPrefetch(accountID int64, requestedModel string) {
	if s == nil || !s.openAIHealthPrefetchEnabled() || accountID <= 0 || s.healthPrefetchCh == nil {
		return
	}
	key := openAIHealthPrefetchKey(accountID, requestedModel)
	state := s.loadOrCreateOpenAIHealthPrefetchState(key)
	now := time.Now()
	last := time.Unix(0, state.lastAt.Load())
	if !last.IsZero() && now.Sub(last) < s.openAIHealthPrefetchCooldown() {
		return
	}
	if !state.inFlight.CompareAndSwap(false, true) {
		return
	}
	state.lastAt.Store(now.UnixNano())
	job := openAIHealthPrefetchJob{AccountID: accountID, RequestedModel: requestedModel}
	select {
	case s.healthPrefetchCh <- job:
	default:
		state.inFlight.Store(false)
	}
}

func openAIHealthPrefetchKey(accountID int64, requestedModel string) string {
	return strings.TrimSpace(requestedModel) + ":" + strconv.FormatInt(accountID, 10)
}

func (s *OpenAIGatewayService) loadOrCreateOpenAIHealthPrefetchState(key string) *openAIHealthPrefetchState {
	if value, ok := s.healthPrefetchState.Load(key); ok {
		if state, ok := value.(*openAIHealthPrefetchState); ok && state != nil {
			return state
		}
	}
	state := &openAIHealthPrefetchState{}
	actual, _ := s.healthPrefetchState.LoadOrStore(key, state)
	existing, _ := actual.(*openAIHealthPrefetchState)
	if existing != nil {
		return existing
	}
	return state
}

func (s *OpenAIGatewayService) runOpenAIHealthPrefetchJob(job openAIHealthPrefetchJob) {
	key := openAIHealthPrefetchKey(job.AccountID, job.RequestedModel)
	defer func() {
		if value, ok := s.healthPrefetchState.Load(key); ok {
			if state, ok := value.(*openAIHealthPrefetchState); ok && state != nil {
				state.inFlight.Store(false)
			}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), s.openAIHealthPrefetchTimeout())
	defer cancel()
	s.probeOpenAIAccountHealth(ctx, job.AccountID, job.RequestedModel)
}

func (s *OpenAIGatewayService) prefetchOpenAIAccountHealthCandidates(candidates []openAIAccountCandidateScore, requestedModel string) {
	if s == nil || !s.openAIHealthPrefetchEnabled() || len(candidates) == 0 {
		return
	}
	limit := s.openAIHealthPrefetchTopN()
	if limit <= 0 {
		return
	}
	if limit > len(candidates) {
		limit = len(candidates)
	}
	for i := 0; i < limit; i++ {
		account := candidates[i].account
		if account == nil || account.ID <= 0 {
			continue
		}
		s.enqueueOpenAIHealthPrefetch(account.ID, requestedModel)
	}
}

func createOpenAIHealthPrefetchPayload(modelID string, isOAuth bool) []byte {
	payload := map[string]any{
		"model": modelID,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": "hi",
					},
				},
			},
		},
		"stream":            false,
		"max_output_tokens": 1,
		"instructions":      openai.DefaultInstructions,
	}
	if isOAuth {
		payload["store"] = false
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func (s *OpenAIGatewayService) probeOpenAIAccountHealth(ctx context.Context, accountID int64, requestedModel string) {
	if s == nil || s.accountRepo == nil || s.httpUpstream == nil || accountID <= 0 {
		return
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil || !account.IsOpenAI() || !account.IsSchedulable() {
		return
	}
	modelID := strings.TrimSpace(requestedModel)
	if modelID == "" {
		modelID = openai.DefaultTestModel
	}
	if account.Type == AccountTypeAPIKey {
		if mapping := account.GetModelMapping(); len(mapping) > 0 {
			if mappedModel, exists := mapping[modelID]; exists {
				modelID = mappedModel
			}
		}
	}

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return
	}

	apiURL := chatgptCodexURL
	isOAuth := account.IsOAuth()
	if !isOAuth {
		baseURL := account.GetOpenAIBaseURL()
		if strings.TrimSpace(baseURL) == "" {
			baseURL = "https://api.openai.com"
		}
		apiURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/") + "/v1/responses"
	}
	payload := createOpenAIHealthPrefetchPayload(modelID, isOAuth)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if isOAuth {
		req.Host = "chatgpt.com"
		req.Header.Set("accept", "application/json")
		if chatgptAccountID := account.GetChatGPTAccountID(); strings.TrimSpace(chatgptAccountID) != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
	if err != nil {
		failoverErr := newProxyRequestFailoverError(account, proxyURL, err)
		s.RegisterOpenAIRuntimeFailure(account, failoverErr)
		s.TempUnscheduleRetryableError(ctx, account.ID, failoverErr)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		failoverErr := buildOpenAIUpstreamFailoverError(account, resp.StatusCode, upstreamMsg, respBody)
		if s.rateLimitService != nil {
			s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		}
		s.RegisterOpenAIRuntimeFailure(account, failoverErr)
		s.TempUnscheduleRetryableError(ctx, account.ID, failoverErr)
		s.queueOpenAIRuntimeStateSync(account.ID)
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	s.MarkOpenAIAccountHealthy(account)
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
