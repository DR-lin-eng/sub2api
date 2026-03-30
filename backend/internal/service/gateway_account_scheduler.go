package service

import (
	"container/heap"
	"context"
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	gatewayAccountScheduleLayerSessionSticky = "session_hash"
	gatewayAccountScheduleLayerLoadBalance   = "load_balance"
)

type GatewayAccountScheduleRequest struct {
	GroupID               *int64
	SessionHash           string
	StickyAccountID       int64
	RequestedModel        string
	Candidates            []*Account
	PreferOAuth           bool
	FallbackSelectionMode string
}

type GatewayAccountScheduleDecision struct {
	Layer               string  `json:"layer"`
	StickySessionHit    bool    `json:"sticky_session_hit"`
	CandidateCount      int     `json:"candidate_count"`
	TopK                int     `json:"top_k"`
	LatencyMs           int64   `json:"latency_ms"`
	LoadSkew            float64 `json:"load_skew"`
	SelectedAccountID   int64   `json:"selected_account_id"`
	SelectedAccountType string  `json:"selected_account_type"`
}

type GatewayAccountSchedulerMetricsSnapshot struct {
	SelectTotal              int64   `json:"select_total"`
	StickySessionHitTotal    int64   `json:"sticky_session_hit_total"`
	LoadBalanceSelectTotal   int64   `json:"load_balance_select_total"`
	AccountSwitchTotal       int64   `json:"account_switch_total"`
	SchedulerLatencyMsTotal  int64   `json:"scheduler_latency_ms_total"`
	SchedulerLatencyMsAvg    float64 `json:"scheduler_latency_ms_avg"`
	StickyHitRatio           float64 `json:"sticky_hit_ratio"`
	AccountSwitchRate        float64 `json:"account_switch_rate"`
	LoadSkewAvg              float64 `json:"load_skew_avg"`
	RuntimeStatsAccountCount int     `json:"runtime_stats_account_count"`
}

type GatewayAccountSchedulerRuntimeSnapshot struct {
	AccountID       int64      `json:"account_id"`
	ErrorRateEWMA   float64    `json:"error_rate_ewma"`
	TTFTEWMAMs      *float64   `json:"ttft_ewma_ms,omitempty"`
	StickyHits      int64      `json:"sticky_hits"`
	LoadBalanceHits int64      `json:"load_balance_hits"`
	Switches        int64      `json:"switches"`
	LastSelectedAt  *time.Time `json:"last_selected_at,omitempty"`
}

type GatewayAccountScheduler interface {
	Select(ctx context.Context, req GatewayAccountScheduleRequest) (*AccountSelectionResult, GatewayAccountScheduleDecision, error)
	ReportResult(accountID int64, success bool, firstTokenMs *int)
	ReportSwitch()
	SnapshotMetrics() GatewayAccountSchedulerMetricsSnapshot
}

type gatewayAccountSchedulerMetrics struct {
	selectTotal            atomic.Int64
	stickySessionHitTotal  atomic.Int64
	loadBalanceSelectTotal atomic.Int64
	accountSwitchTotal     atomic.Int64
	latencyMsTotal         atomic.Int64
	loadSkewMilliTotal     atomic.Int64
}

func (m *gatewayAccountSchedulerMetrics) recordSelect(decision GatewayAccountScheduleDecision) {
	if m == nil {
		return
	}
	m.selectTotal.Add(1)
	m.latencyMsTotal.Add(decision.LatencyMs)
	m.loadSkewMilliTotal.Add(int64(math.Round(decision.LoadSkew * 1000)))
	if decision.StickySessionHit {
		m.stickySessionHitTotal.Add(1)
	}
	if decision.Layer == gatewayAccountScheduleLayerLoadBalance {
		m.loadBalanceSelectTotal.Add(1)
	}
}

func (m *gatewayAccountSchedulerMetrics) recordSwitch() {
	if m == nil {
		return
	}
	m.accountSwitchTotal.Add(1)
}

type gatewayAccountRuntimeStats struct {
	accounts     sync.Map
	accountCount atomic.Int64
}

type gatewayAccountRuntimeStat struct {
	errorRateEWMABits   atomic.Uint64
	ttftEWMABits        atomic.Uint64
	stickyHits          atomic.Int64
	loadBalanceHits     atomic.Int64
	switches            atomic.Int64
	lastSelectedAtMilli atomic.Int64
}

func newGatewayAccountRuntimeStats() *gatewayAccountRuntimeStats {
	return &gatewayAccountRuntimeStats{}
}

func (s *gatewayAccountRuntimeStats) loadOrCreate(accountID int64) *gatewayAccountRuntimeStat {
	if value, ok := s.accounts.Load(accountID); ok {
		if stat, _ := value.(*gatewayAccountRuntimeStat); stat != nil {
			return stat
		}
	}

	stat := &gatewayAccountRuntimeStat{}
	stat.ttftEWMABits.Store(math.Float64bits(math.NaN()))
	actual, loaded := s.accounts.LoadOrStore(accountID, stat)
	if !loaded {
		s.accountCount.Add(1)
		return stat
	}
	existing, _ := actual.(*gatewayAccountRuntimeStat)
	if existing != nil {
		return existing
	}
	return stat
}

func (s *gatewayAccountRuntimeStats) report(accountID int64, success bool, firstTokenMs *int, alpha float64) {
	if s == nil || accountID <= 0 {
		return
	}
	if alpha <= 0 || alpha > 1 {
		alpha = 0.2
	}
	stat := s.loadOrCreate(accountID)

	errorSample := 1.0
	if success {
		errorSample = 0.0
	}
	updateEWMAAtomic(&stat.errorRateEWMABits, errorSample, alpha)

	if firstTokenMs != nil && *firstTokenMs > 0 {
		ttft := float64(*firstTokenMs)
		ttftBits := math.Float64bits(ttft)
		for {
			oldBits := stat.ttftEWMABits.Load()
			oldValue := math.Float64frombits(oldBits)
			if math.IsNaN(oldValue) {
				if stat.ttftEWMABits.CompareAndSwap(oldBits, ttftBits) {
					break
				}
				continue
			}
			newValue := alpha*ttft + (1-alpha)*oldValue
			if stat.ttftEWMABits.CompareAndSwap(oldBits, math.Float64bits(newValue)) {
				break
			}
		}
	}
}

func (s *gatewayAccountRuntimeStats) recordStickyHit(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	stat := s.loadOrCreate(accountID)
	stat.stickyHits.Add(1)
	stat.lastSelectedAtMilli.Store(time.Now().UTC().UnixMilli())
}

func (s *gatewayAccountRuntimeStats) recordLoadBalanceHit(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	stat := s.loadOrCreate(accountID)
	stat.loadBalanceHits.Add(1)
	stat.lastSelectedAtMilli.Store(time.Now().UTC().UnixMilli())
}

func (s *gatewayAccountRuntimeStats) recordSwitch(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	stat := s.loadOrCreate(accountID)
	stat.switches.Add(1)
}

func (s *gatewayAccountRuntimeStats) snapshot(accountID int64) (errorRate float64, ttft float64, hasTTFT bool) {
	if s == nil || accountID <= 0 {
		return 0, 0, false
	}
	value, ok := s.accounts.Load(accountID)
	if !ok {
		return 0, 0, false
	}
	stat, _ := value.(*gatewayAccountRuntimeStat)
	if stat == nil {
		return 0, 0, false
	}
	errorRate = clamp01(math.Float64frombits(stat.errorRateEWMABits.Load()))
	ttftValue := math.Float64frombits(stat.ttftEWMABits.Load())
	if math.IsNaN(ttftValue) {
		return errorRate, 0, false
	}
	return errorRate, ttftValue, true
}

func (s *gatewayAccountRuntimeStats) snapshotAll() []GatewayAccountSchedulerRuntimeSnapshot {
	if s == nil {
		return nil
	}
	out := make([]GatewayAccountSchedulerRuntimeSnapshot, 0)
	s.accounts.Range(func(key, value any) bool {
		accountID, ok := key.(int64)
		if !ok || accountID <= 0 {
			return true
		}
		stat, _ := value.(*gatewayAccountRuntimeStat)
		if stat == nil {
			return true
		}
		item := GatewayAccountSchedulerRuntimeSnapshot{
			AccountID:       accountID,
			ErrorRateEWMA:   clamp01(math.Float64frombits(stat.errorRateEWMABits.Load())),
			StickyHits:      stat.stickyHits.Load(),
			LoadBalanceHits: stat.loadBalanceHits.Load(),
			Switches:        stat.switches.Load(),
		}
		ttft := math.Float64frombits(stat.ttftEWMABits.Load())
		if !math.IsNaN(ttft) {
			item.TTFTEWMAMs = &ttft
		}
		if milli := stat.lastSelectedAtMilli.Load(); milli > 0 {
			ts := time.UnixMilli(milli).UTC()
			item.LastSelectedAt = &ts
		}
		out = append(out, item)
		return true
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].AccountID < out[j].AccountID
	})
	return out
}

func (s *gatewayAccountRuntimeStats) size() int {
	if s == nil {
		return 0
	}
	return int(s.accountCount.Load())
}

type defaultGatewayAccountScheduler struct {
	service *GatewayService
	metrics gatewayAccountSchedulerMetrics
	stats   *gatewayAccountRuntimeStats
}

func newDefaultGatewayAccountScheduler(service *GatewayService, stats *gatewayAccountRuntimeStats) GatewayAccountScheduler {
	if stats == nil {
		stats = newGatewayAccountRuntimeStats()
	}
	return &defaultGatewayAccountScheduler{
		service: service,
		stats:   stats,
	}
}

type gatewayAccountCandidateScore struct {
	account   *Account
	loadInfo  *AccountLoadInfo
	score     float64
	errorRate float64
	ttft      float64
	hasTTFT   bool
	slotCap   int
	slotFree  int
	readyTier int
}

type gatewayAccountCandidateHeap []gatewayAccountCandidateScore

func (h gatewayAccountCandidateHeap) Len() int { return len(h) }
func (h gatewayAccountCandidateHeap) Less(i, j int) bool {
	return isGatewayAccountCandidateBetter(h[j], h[i])
}
func (h gatewayAccountCandidateHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *gatewayAccountCandidateHeap) Push(x any) {
	candidate, ok := x.(gatewayAccountCandidateScore)
	if !ok {
		panic("gatewayAccountCandidateHeap: invalid element type")
	}
	*h = append(*h, candidate)
}
func (h *gatewayAccountCandidateHeap) Pop() any {
	old := *h
	n := len(old)
	last := old[n-1]
	*h = old[:n-1]
	return last
}

func isGatewayAccountCandidateBetter(left gatewayAccountCandidateScore, right gatewayAccountCandidateScore) bool {
	if left.score != right.score {
		return left.score > right.score
	}
	if left.slotFree != right.slotFree {
		return left.slotFree > right.slotFree
	}
	if left.account.Priority != right.account.Priority {
		return left.account.Priority < right.account.Priority
	}
	if left.loadInfo.CurrentConcurrency != right.loadInfo.CurrentConcurrency {
		return left.loadInfo.CurrentConcurrency < right.loadInfo.CurrentConcurrency
	}
	if left.loadInfo.LoadRate != right.loadInfo.LoadRate {
		return left.loadInfo.LoadRate < right.loadInfo.LoadRate
	}
	if left.loadInfo.WaitingCount != right.loadInfo.WaitingCount {
		return left.loadInfo.WaitingCount < right.loadInfo.WaitingCount
	}
	return left.account.ID < right.account.ID
}

func selectTopKGatewayCandidates(candidates []gatewayAccountCandidateScore, topK int) []gatewayAccountCandidateScore {
	if len(candidates) == 0 {
		return nil
	}
	if topK <= 0 {
		topK = 1
	}
	if topK >= len(candidates) {
		ranked := append([]gatewayAccountCandidateScore(nil), candidates...)
		sort.Slice(ranked, func(i, j int) bool {
			return isGatewayAccountCandidateBetter(ranked[i], ranked[j])
		})
		return ranked
	}

	best := make(gatewayAccountCandidateHeap, 0, topK)
	for _, candidate := range candidates {
		if len(best) < topK {
			heap.Push(&best, candidate)
			continue
		}
		if isGatewayAccountCandidateBetter(candidate, best[0]) {
			best[0] = candidate
			heap.Fix(&best, 0)
		}
	}

	ranked := make([]gatewayAccountCandidateScore, len(best))
	copy(ranked, best)
	sort.Slice(ranked, func(i, j int) bool {
		return isGatewayAccountCandidateBetter(ranked[i], ranked[j])
	})
	return ranked
}

func deriveGatewaySelectionSeed(req GatewayAccountScheduleRequest) uint64 {
	hasher := fnv.New64a()
	writeValue := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		_, _ = hasher.Write([]byte(trimmed))
		_, _ = hasher.Write([]byte{0})
	}
	writeValue(req.SessionHash)
	writeValue(req.RequestedModel)
	if req.GroupID != nil {
		_, _ = hasher.Write([]byte(strconv.FormatInt(*req.GroupID, 10)))
	}

	seed := hasher.Sum64()
	if strings.TrimSpace(req.SessionHash) == "" {
		seed ^= uint64(time.Now().UnixNano())
	}
	if seed == 0 {
		seed = uint64(time.Now().UnixNano()) ^ 0x9e3779b97f4a7c15
	}
	return seed
}

func buildGatewayWeightedSelectionOrder(
	candidates []gatewayAccountCandidateScore,
	req GatewayAccountScheduleRequest,
) []gatewayAccountCandidateScore {
	if len(candidates) <= 1 {
		return append([]gatewayAccountCandidateScore(nil), candidates...)
	}

	pool := append([]gatewayAccountCandidateScore(nil), candidates...)
	weights := make([]float64, len(pool))
	minScore := pool[0].score
	for i := 1; i < len(pool); i++ {
		if pool[i].score < minScore {
			minScore = pool[i].score
		}
	}
	for i := range pool {
		weight := (pool[i].score - minScore) + 1.0
		if math.IsNaN(weight) || math.IsInf(weight, 0) || weight <= 0 {
			weight = 1.0
		}
		weights[i] = weight
	}

	order := make([]gatewayAccountCandidateScore, 0, len(pool))
	rng := newOpenAISelectionRNG(deriveGatewaySelectionSeed(req))
	for len(pool) > 0 {
		total := 0.0
		for _, w := range weights {
			total += w
		}

		selectedIdx := 0
		if total > 0 {
			r := rng.nextFloat64() * total
			acc := 0.0
			for i, w := range weights {
				acc += w
				if r <= acc {
					selectedIdx = i
					break
				}
			}
		} else {
			selectedIdx = int(rng.nextUint64() % uint64(len(pool)))
		}

		order = append(order, pool[selectedIdx])
		pool = append(pool[:selectedIdx], pool[selectedIdx+1:]...)
		weights = append(weights[:selectedIdx], weights[selectedIdx+1:]...)
	}
	return order
}

func buildGatewayTieredSelectionOrder(
	candidates []gatewayAccountCandidateScore,
	topK int,
	req GatewayAccountScheduleRequest,
) []gatewayAccountCandidateScore {
	if len(candidates) <= 1 {
		return append([]gatewayAccountCandidateScore(nil), candidates...)
	}
	if topK <= 0 {
		topK = 1
	}

	tierBuckets := [3][]gatewayAccountCandidateScore{}
	for _, candidate := range candidates {
		tier := candidate.readyTier
		if tier < 0 {
			tier = 0
		}
		if tier > 2 {
			tier = 2
		}
		tierBuckets[tier] = append(tierBuckets[tier], candidate)
	}

	order := make([]gatewayAccountCandidateScore, 0, len(candidates))
	for tier := 0; tier < len(tierBuckets); tier++ {
		if len(tierBuckets[tier]) == 0 {
			continue
		}
		ranked := selectTopKGatewayCandidates(tierBuckets[tier], topK)
		order = append(order, buildGatewayWeightedSelectionOrder(ranked, req)...)
	}
	return order
}

func adaptiveGatewaySelectionTopK(baseTopK int, candidateCount int, loadSkew float64) int {
	return adaptiveSchedulerSelectionTopK(baseTopK, candidateCount, loadSkew)
}

func (s *defaultGatewayAccountScheduler) Select(
	ctx context.Context,
	req GatewayAccountScheduleRequest,
) (*AccountSelectionResult, GatewayAccountScheduleDecision, error) {
	decision := GatewayAccountScheduleDecision{}
	start := time.Now()
	defer func() {
		decision.LatencyMs = time.Since(start).Milliseconds()
		s.metrics.recordSelect(decision)
	}()

	if selection, err := s.selectStickySession(ctx, req); err != nil {
		return nil, decision, err
	} else if selection != nil && selection.Account != nil {
		decision.Layer = gatewayAccountScheduleLayerSessionSticky
		decision.StickySessionHit = true
		decision.SelectedAccountID = selection.Account.ID
		decision.SelectedAccountType = selection.Account.Type
		return selection, decision, nil
	}

	selection, candidateCount, topK, loadSkew, err := s.selectByLoadBalance(ctx, req)
	decision.Layer = gatewayAccountScheduleLayerLoadBalance
	decision.CandidateCount = candidateCount
	decision.TopK = topK
	decision.LoadSkew = loadSkew
	if err != nil {
		return nil, decision, err
	}
	if selection != nil && selection.Account != nil {
		decision.SelectedAccountID = selection.Account.ID
		decision.SelectedAccountType = selection.Account.Type
	}
	return selection, decision, nil
}

func (s *defaultGatewayAccountScheduler) selectStickySession(
	ctx context.Context,
	req GatewayAccountScheduleRequest,
) (*AccountSelectionResult, error) {
	if strings.TrimSpace(req.SessionHash) == "" || req.StickyAccountID <= 0 || len(req.Candidates) == 0 {
		return nil, nil
	}

	var account *Account
	for _, candidate := range req.Candidates {
		if candidate != nil && candidate.ID == req.StickyAccountID {
			account = candidate
			break
		}
	}
	if account == nil {
		if s.service != nil && s.service.cache != nil {
			_ = s.service.cache.DeleteSessionAccountID(ctx, derefGroupID(req.GroupID), req.SessionHash)
		}
		return nil, nil
	}

	result, acquireErr := s.service.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
	if acquireErr == nil && result != nil && result.Acquired {
		if !s.service.checkAndRegisterSession(ctx, account, req.SessionHash) {
			result.ReleaseFunc()
			return nil, nil
		}
		s.stats.recordStickyHit(account.ID)
		if req.SessionHash != "" && s.service.cache != nil {
			_ = s.service.cache.RefreshSessionTTL(ctx, derefGroupID(req.GroupID), req.SessionHash, stickySessionTTL)
		}
		return &AccountSelectionResult{
			Account:     account,
			Acquired:    true,
			ReleaseFunc: result.ReleaseFunc,
		}, nil
	}

	cfg := s.service.schedulingConfig()
	if s.service.concurrencyService != nil {
		if !s.service.checkAndRegisterSession(ctx, account, req.SessionHash) {
			return nil, nil
		}
		s.stats.recordStickyHit(account.ID)
		return &AccountSelectionResult{
			Account: account,
			WaitPlan: &AccountWaitPlan{
				AccountID:      account.ID,
				MaxConcurrency: account.Concurrency,
				Timeout:        cfg.StickySessionWaitTimeout,
				MaxWaiting:     cfg.StickySessionMaxWaiting,
			},
		}, nil
	}
	return nil, nil
}

func (s *defaultGatewayAccountScheduler) selectByLoadBalance(
	ctx context.Context,
	req GatewayAccountScheduleRequest,
) (*AccountSelectionResult, int, int, float64, error) {
	if len(req.Candidates) == 0 {
		return nil, 0, 0, 0, ErrNoAvailableAccounts
	}

	loadReq := make([]AccountWithConcurrency, 0, len(req.Candidates))
	for _, account := range req.Candidates {
		if account == nil {
			continue
		}
		loadReq = append(loadReq, AccountWithConcurrency{
			ID:             account.ID,
			MaxConcurrency: account.EffectiveLoadFactor(),
		})
	}
	if len(loadReq) == 0 {
		return nil, 0, 0, 0, ErrNoAvailableAccounts
	}

	loadMap := map[int64]*AccountLoadInfo{}
	if s.service.concurrencyService != nil {
		if batchLoad, loadErr := s.service.concurrencyService.GetAccountsLoadBatch(ctx, loadReq); loadErr == nil {
			loadMap = batchLoad
		} else {
			if result, ok := s.service.tryAcquireByLegacyOrder(ctx, req.Candidates, req.GroupID, req.SessionHash, req.PreferOAuth); ok {
				s.stats.recordLoadBalanceHit(result.Account.ID)
				return result, len(req.Candidates), 1, 0, nil
			}
		}
	}

	minPriority, maxPriority := req.Candidates[0].Priority, req.Candidates[0].Priority
	maxWaiting := 1
	loadRateSum := 0.0
	loadRateSumSquares := 0.0
	minTTFT, maxTTFT := 0.0, 0.0
	hasTTFTSample := false
	candidates := make([]gatewayAccountCandidateScore, 0, len(req.Candidates))

	for _, account := range req.Candidates {
		if account == nil {
			continue
		}
		loadInfo := loadMap[account.ID]
		if loadInfo == nil {
			loadInfo = &AccountLoadInfo{AccountID: account.ID}
		}
		if account.Priority < minPriority {
			minPriority = account.Priority
		}
		if account.Priority > maxPriority {
			maxPriority = account.Priority
		}
		if loadInfo.WaitingCount > maxWaiting {
			maxWaiting = loadInfo.WaitingCount
		}
		readyTier, slotCap, slotFree := gatewayAccountReadyTier(account, loadInfo)
		errorRate, ttft, hasTTFT := s.stats.snapshot(account.ID)
		if hasTTFT && ttft > 0 {
			if !hasTTFTSample {
				minTTFT, maxTTFT = ttft, ttft
				hasTTFTSample = true
			} else {
				if ttft < minTTFT {
					minTTFT = ttft
				}
				if ttft > maxTTFT {
					maxTTFT = ttft
				}
			}
		}
		loadRate := float64(loadInfo.LoadRate)
		loadRateSum += loadRate
		loadRateSumSquares += loadRate * loadRate
		candidates = append(candidates, gatewayAccountCandidateScore{
			account:   account,
			loadInfo:  loadInfo,
			errorRate: errorRate,
			ttft:      ttft,
			hasTTFT:   hasTTFT,
			slotCap:   slotCap,
			slotFree:  slotFree,
			readyTier: readyTier,
		})
	}
	if len(candidates) == 0 {
		return nil, 0, 0, 0, ErrNoAvailableAccounts
	}

	loadSkew := calcLoadSkewByMoments(loadRateSum, loadRateSumSquares, len(candidates))
	weights := s.service.gatewaySchedulerWeights()
	for i := range candidates {
		item := &candidates[i]
		priorityFactor := 1.0
		if maxPriority > minPriority {
			priorityFactor = 1 - float64(item.account.Priority-minPriority)/float64(maxPriority-minPriority)
		}
		loadFactor := 1 - clamp01(float64(item.loadInfo.LoadRate)/100.0)
		slotFactor := clamp01(float64(item.slotFree) / float64(max(1, item.slotCap)))
		loadFactor = 0.6*loadFactor + 0.4*slotFactor
		queueFactor := 1 - clamp01(float64(item.loadInfo.WaitingCount)/float64(maxWaiting))
		switch item.readyTier {
		case 0:
			queueFactor = 1.0
		case 1:
			if queueFactor < 0.4 {
				queueFactor = 0.4
			}
		default:
			queueFactor *= 0.2
		}
		errorFactor := 1 - clamp01(item.errorRate)
		ttftFactor := 0.5
		if item.hasTTFT && hasTTFTSample && maxTTFT > minTTFT {
			ttftFactor = 1 - clamp01((item.ttft-minTTFT)/(maxTTFT-minTTFT))
		}

		item.score = weights.Priority*priorityFactor +
			weights.Load*loadFactor +
			weights.Queue*queueFactor +
			weights.ErrorRate*errorFactor +
			weights.TTFT*ttftFactor
	}

	topK := adaptiveGatewaySelectionTopK(s.service.gatewaySchedulerTopK(), len(candidates), loadSkew)
	selectionOrder := buildGatewayTieredSelectionOrder(candidates, topK, req)
	for _, candidate := range selectionOrder {
		result, acquireErr := s.service.tryAcquireAccountSlot(ctx, candidate.account.ID, candidate.account.Concurrency)
		if acquireErr != nil {
			return nil, len(candidates), topK, loadSkew, acquireErr
		}
		if result != nil && result.Acquired {
			if !s.service.checkAndRegisterSession(ctx, candidate.account, req.SessionHash) {
				result.ReleaseFunc()
				continue
			}
			if req.SessionHash != "" {
				_ = s.service.BindStickySession(ctx, req.GroupID, req.SessionHash, candidate.account.ID)
			}
			s.stats.recordLoadBalanceHit(candidate.account.ID)
			return &AccountSelectionResult{
				Account:     candidate.account,
				Acquired:    true,
				ReleaseFunc: result.ReleaseFunc,
			}, len(candidates), topK, loadSkew, nil
		}
	}

	cfg := s.service.schedulingConfig()
	fallbackAccounts := make([]*Account, 0, len(selectionOrder))
	for _, candidate := range selectionOrder {
		if candidate.account != nil {
			fallbackAccounts = append(fallbackAccounts, candidate.account)
		}
	}
	if len(fallbackAccounts) == 0 {
		fallbackAccounts = append(fallbackAccounts, req.Candidates...)
	}
	s.service.sortCandidatesForFallback(fallbackAccounts, req.PreferOAuth, req.FallbackSelectionMode)
	for _, acc := range fallbackAccounts {
		if acc == nil {
			continue
		}
		if !s.service.checkAndRegisterSession(ctx, acc, req.SessionHash) {
			continue
		}
		s.stats.recordLoadBalanceHit(acc.ID)
		return &AccountSelectionResult{
			Account: acc,
			WaitPlan: &AccountWaitPlan{
				AccountID:      acc.ID,
				MaxConcurrency: acc.Concurrency,
				Timeout:        cfg.FallbackWaitTimeout,
				MaxWaiting:     cfg.FallbackMaxWaiting,
			},
		}, len(candidates), topK, loadSkew, nil
	}

	return nil, len(candidates), topK, loadSkew, ErrNoAvailableAccounts
}

func (s *defaultGatewayAccountScheduler) ReportResult(accountID int64, success bool, firstTokenMs *int) {
	if s == nil || s.stats == nil {
		return
	}
	s.stats.report(accountID, success, firstTokenMs, s.service.gatewaySchedulerRuntimeStatsAlpha())
}

func (s *defaultGatewayAccountScheduler) ReportSwitch() {
	if s == nil {
		return
	}
	s.metrics.recordSwitch()
}

func (s *defaultGatewayAccountScheduler) SnapshotMetrics() GatewayAccountSchedulerMetricsSnapshot {
	if s == nil {
		return GatewayAccountSchedulerMetricsSnapshot{}
	}

	selectTotal := s.metrics.selectTotal.Load()
	stickyHit := s.metrics.stickySessionHitTotal.Load()
	switchTotal := s.metrics.accountSwitchTotal.Load()
	latencyTotal := s.metrics.latencyMsTotal.Load()
	loadSkewTotal := s.metrics.loadSkewMilliTotal.Load()

	snapshot := GatewayAccountSchedulerMetricsSnapshot{
		SelectTotal:              selectTotal,
		StickySessionHitTotal:    stickyHit,
		LoadBalanceSelectTotal:   s.metrics.loadBalanceSelectTotal.Load(),
		AccountSwitchTotal:       switchTotal,
		SchedulerLatencyMsTotal:  latencyTotal,
		RuntimeStatsAccountCount: s.stats.size(),
	}
	if selectTotal > 0 {
		snapshot.SchedulerLatencyMsAvg = float64(latencyTotal) / float64(selectTotal)
		snapshot.StickyHitRatio = float64(stickyHit) / float64(selectTotal)
		snapshot.AccountSwitchRate = float64(switchTotal) / float64(selectTotal)
		snapshot.LoadSkewAvg = float64(loadSkewTotal) / 1000 / float64(selectTotal)
	}
	return snapshot
}

func (s *GatewayService) getGatewayAccountScheduler() GatewayAccountScheduler {
	if s == nil {
		return nil
	}
	s.gatewaySchedulerOnce.Do(func() {
		if s.gatewayAccountStats == nil {
			s.gatewayAccountStats = newGatewayAccountRuntimeStats()
		}
		if s.gatewayScheduler == nil {
			s.gatewayScheduler = newDefaultGatewayAccountScheduler(s, s.gatewayAccountStats)
		}
	})
	return s.gatewayScheduler
}

func (s *GatewayService) ReportGatewayAccountScheduleResult(accountID int64, success bool, firstTokenMs *int) {
	scheduler := s.getGatewayAccountScheduler()
	if scheduler == nil {
		return
	}
	scheduler.ReportResult(accountID, success, firstTokenMs)
}

func (s *GatewayService) RecordGatewayAccountSwitch(accountID int64) {
	scheduler := s.getGatewayAccountScheduler()
	if scheduler == nil {
		return
	}
	scheduler.ReportSwitch()
	if s.gatewayAccountStats != nil && accountID > 0 {
		s.gatewayAccountStats.recordSwitch(accountID)
	}
}

func (s *GatewayService) recordGatewayStickyHit(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	_ = s.getGatewayAccountScheduler()
	if s.gatewayAccountStats == nil {
		return
	}
	s.gatewayAccountStats.recordStickyHit(accountID)
}

func (s *GatewayService) recordGatewayLoadBalanceHit(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	_ = s.getGatewayAccountScheduler()
	if s.gatewayAccountStats == nil {
		return
	}
	s.gatewayAccountStats.recordLoadBalanceHit(accountID)
}

func (s *GatewayService) SnapshotGatewayAccountSchedulerMetrics() GatewayAccountSchedulerMetricsSnapshot {
	scheduler := s.getGatewayAccountScheduler()
	if scheduler == nil {
		return GatewayAccountSchedulerMetricsSnapshot{}
	}
	return scheduler.SnapshotMetrics()
}

func (s *GatewayService) SnapshotGatewayAccountSchedulerRuntime() []GatewayAccountSchedulerRuntimeSnapshot {
	if s == nil || s.gatewayAccountStats == nil {
		return nil
	}
	return s.gatewayAccountStats.snapshotAll()
}

func (s *GatewayService) gatewaySchedulerTopK() int {
	if s != nil && s.cfg != nil && s.cfg.Gateway.Scheduling.LBTopK > 0 {
		return s.cfg.Gateway.Scheduling.LBTopK
	}
	return 5
}

func (s *GatewayService) gatewaySchedulerRuntimeStatsAlpha() float64 {
	if s != nil && s.cfg != nil {
		if alpha := s.cfg.Gateway.Scheduling.RuntimeStatsAlpha; alpha > 0 && alpha <= 1 {
			return alpha
		}
	}
	return 0.2
}

func (s *GatewayService) gatewaySchedulerWeights() config.GatewaySchedulerScoreWeights {
	if s != nil && s.cfg != nil {
		return s.cfg.Gateway.Scheduling.SchedulerScoreWeights
	}
	return config.GatewaySchedulerScoreWeights{
		Priority:  1.0,
		Load:      1.0,
		Queue:     0.7,
		ErrorRate: 0.8,
		TTFT:      0.5,
	}
}
