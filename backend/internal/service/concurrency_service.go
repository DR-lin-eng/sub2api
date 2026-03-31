package service

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"golang.org/x/sync/singleflight"
)

// ConcurrencyCache 定义并发控制的缓存接口
// 使用有序集合存储槽位，按时间戳清理过期条目
type ConcurrencyCache interface {
	// 账号槽位管理
	// 键格式: concurrency:account:{accountID}（有序集合，成员为 requestID）
	AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error)
	ReleaseAccountSlot(ctx context.Context, accountID int64, requestID string) error
	GetAccountConcurrency(ctx context.Context, accountID int64) (int, error)
	GetAccountConcurrencyBatch(ctx context.Context, accountIDs []int64) (map[int64]int, error)

	// 账号等待队列（账号级）
	IncrementAccountWaitCount(ctx context.Context, accountID int64, maxWait int) (bool, error)
	DecrementAccountWaitCount(ctx context.Context, accountID int64) error
	GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error)

	// 用户槽位管理
	// 键格式: concurrency:user:{userID}（有序集合，成员为 requestID）
	AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error)
	ReleaseUserSlot(ctx context.Context, userID int64, requestID string) error
	GetUserConcurrency(ctx context.Context, userID int64) (int, error)

	// 等待队列计数（只在首次创建时设置 TTL）
	IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error)
	DecrementWaitCount(ctx context.Context, userID int64) error

	// 批量负载查询（只读）
	GetAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error)
	GetUsersLoadBatch(ctx context.Context, users []UserWithConcurrency) (map[int64]*UserLoadInfo, error)

	// 清理过期槽位（后台任务）
	CleanupExpiredAccountSlots(ctx context.Context, accountID int64) error

	// 启动时清理旧进程遗留槽位与等待计数
	CleanupStaleProcessSlots(ctx context.Context, activeRequestPrefix string) error
}

var (
	requestIDPrefix  = initRequestIDPrefix()
	requestIDCounter atomic.Uint64
)

func initRequestIDPrefix() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err == nil {
		return "r" + strconv.FormatUint(binary.BigEndian.Uint64(b), 36)
	}
	fallback := uint64(time.Now().UnixNano()) ^ (uint64(os.Getpid()) << 16)
	return "r" + strconv.FormatUint(fallback, 36)
}

func RequestIDPrefix() string {
	return requestIDPrefix
}

func generateRequestID() string {
	seq := requestIDCounter.Add(1)
	return requestIDPrefix + "-" + strconv.FormatUint(seq, 36)
}

func (s *ConcurrencyService) CleanupStaleProcessSlots(ctx context.Context) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.CleanupStaleProcessSlots(ctx, RequestIDPrefix())
}

const (
	// Default extra wait slots beyond concurrency limit
	defaultExtraWaitSlots = 20
)

var accountLoadCacheTTL = 250 * time.Millisecond
var accountWaitCountCacheTTL = 250 * time.Millisecond

type cachedAccountLoadSnapshot struct {
	CurrentConcurrency int
	WaitingCount       int
	ExpiresAtUnixNano  int64
}

type cachedAccountWaitCountSnapshot struct {
	Count             int
	ExpiresAtUnixNano int64
}

type cachedUserLoadSnapshot struct {
	CurrentConcurrency int
	WaitingCount       int
	ExpiresAtUnixNano  int64
}

type cachedUserWaitCountSnapshot struct {
	Count             int
	ExpiresAtUnixNano int64
}

type localWaitTurnCoordinator struct {
	mu     sync.Mutex
	queues map[string][]*localWaitTurnWaiter
}

type localWaitTurnWaiter struct {
	turnCh chan struct{}
}

func (w *localWaitTurnWaiter) signal() {
	if w == nil || w.turnCh == nil {
		return
	}
	select {
	case w.turnCh <- struct{}{}:
	default:
	}
}

func newLocalWaitTurnCoordinator() *localWaitTurnCoordinator {
	return &localWaitTurnCoordinator{
		queues: make(map[string][]*localWaitTurnWaiter),
	}
}

func localWaitTurnKey(slotType string, id int64) string {
	slotType = strings.TrimSpace(slotType)
	if slotType == "" || id <= 0 {
		return ""
	}
	return slotType + ":" + strconv.FormatInt(id, 10)
}

func (c *localWaitTurnCoordinator) register(slotType string, id int64) (<-chan struct{}, func()) {
	if c == nil {
		return nil, func() {}
	}
	key := localWaitTurnKey(slotType, id)
	if key == "" {
		return nil, func() {}
	}

	waiter := &localWaitTurnWaiter{turnCh: make(chan struct{}, 1)}

	c.mu.Lock()
	queue := c.queues[key]
	isHead := len(queue) == 0
	c.queues[key] = append(queue, waiter)
	c.mu.Unlock()

	if isHead {
		waiter.signal()
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			c.release(key, waiter)
		})
	}
	return waiter.turnCh, release
}

func (c *localWaitTurnCoordinator) release(key string, waiter *localWaitTurnWaiter) {
	if c == nil || key == "" || waiter == nil {
		return
	}

	var next *localWaitTurnWaiter
	c.mu.Lock()
	queue := c.queues[key]
	index := -1
	for i, item := range queue {
		if item == waiter {
			index = i
			break
		}
	}
	if index < 0 {
		c.mu.Unlock()
		return
	}

	wasHead := index == 0
	queue = append(queue[:index], queue[index+1:]...)
	if len(queue) == 0 {
		delete(c.queues, key)
	} else {
		c.queues[key] = queue
		if wasHead {
			next = queue[0]
		}
	}
	c.mu.Unlock()

	if next != nil {
		next.signal()
	}
}

// ConcurrencyService manages concurrent request limiting for accounts and users
type ConcurrencyService struct {
	cache                 ConcurrencyCache
	fairWaitQueueEnabled  atomic.Bool
	accountLoadCache      sync.Map
	accountWaitCountCache sync.Map
	userLoadCache         sync.Map
	userWaitCountCache    sync.Map
	localWaitTurns        *localWaitTurnCoordinator
	accountLoadSF         singleflight.Group
	userLoadSF            singleflight.Group
}

// NewConcurrencyService creates a new ConcurrencyService
func NewConcurrencyService(cache ConcurrencyCache) *ConcurrencyService {
	return &ConcurrencyService{
		cache:          cache,
		localWaitTurns: newLocalWaitTurnCoordinator(),
	}
}

type waitTicketQueueCache interface {
	EnqueueUserWaitTicket(ctx context.Context, userID int64, ticketID string) error
	IsUserWaitTicketTurn(ctx context.Context, userID int64, ticketID string) (bool, error)
	RemoveUserWaitTicket(ctx context.Context, userID int64, ticketID string) error

	EnqueueAccountWaitTicket(ctx context.Context, accountID int64, ticketID string) error
	IsAccountWaitTicketTurn(ctx context.Context, accountID int64, ticketID string) (bool, error)
	RemoveAccountWaitTicket(ctx context.Context, accountID int64, ticketID string) error
}

func NewConcurrencyTicketID() string {
	return generateRequestID()
}

func (s *ConcurrencyService) SetFairWaitQueueEnabled(enabled bool) {
	if s == nil {
		return
	}
	s.fairWaitQueueEnabled.Store(enabled)
}

func (s *ConcurrencyService) FairWaitQueueEnabled() bool {
	if s == nil {
		return false
	}
	return s.fairWaitQueueEnabled.Load()
}

func (s *ConcurrencyService) RegisterLocalWaitTurn(slotType string, id int64) (<-chan struct{}, func()) {
	if s == nil || s.localWaitTurns == nil {
		return nil, func() {}
	}
	return s.localWaitTurns.register(slotType, id)
}

func (s *ConcurrencyService) ticketQueueCache() waitTicketQueueCache {
	if s == nil || s.cache == nil {
		return nil
	}
	cache, _ := s.cache.(waitTicketQueueCache)
	return cache
}

// AcquireResult represents the result of acquiring a concurrency slot
type AcquireResult struct {
	Acquired    bool
	ReleaseFunc func() // Must be called when done (typically via defer)
}

type AccountWithConcurrency struct {
	ID             int64
	MaxConcurrency int
}

type UserWithConcurrency struct {
	ID             int64
	MaxConcurrency int
}

type AccountLoadInfo struct {
	AccountID          int64
	CurrentConcurrency int
	WaitingCount       int
	LoadRate           int // 0-100+ (percent)
}

type UserLoadInfo struct {
	UserID             int64
	CurrentConcurrency int
	WaitingCount       int
	LoadRate           int // 0-100+ (percent)
}

// AcquireAccountSlot attempts to acquire a concurrency slot for an account.
// If the account is at max concurrency, it waits until a slot is available or timeout.
// Returns a release function that MUST be called when the request completes.
func (s *ConcurrencyService) AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int) (*AcquireResult, error) {
	// If maxConcurrency is 0 or negative, no limit
	if maxConcurrency <= 0 {
		return &AcquireResult{
			Acquired:    true,
			ReleaseFunc: func() {},
		}, nil
	}

	requestID := generateRequestID()

	acquired, err := s.cache.AcquireAccountSlot(ctx, accountID, maxConcurrency, requestID)
	if err != nil {
		return nil, err
	}

	if acquired {
		return &AcquireResult{
			Acquired: true,
			ReleaseFunc: func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := s.cache.ReleaseAccountSlot(bgCtx, accountID, requestID); err != nil {
					logger.LegacyPrintf("service.concurrency", "Warning: failed to release account slot for %d (req=%s): %v", accountID, requestID, err)
				}
			},
		}, nil
	}

	return &AcquireResult{
		Acquired:    false,
		ReleaseFunc: nil,
	}, nil
}

// AcquireUserSlot attempts to acquire a concurrency slot for a user.
// If the user is at max concurrency, it waits until a slot is available or timeout.
// Returns a release function that MUST be called when the request completes.
func (s *ConcurrencyService) AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int) (*AcquireResult, error) {
	if maxConcurrency <= 0 {
		return &AcquireResult{
			Acquired:    true,
			ReleaseFunc: func() {},
		}, nil
	}

	requestID := generateRequestID()

	acquired, err := s.cache.AcquireUserSlot(ctx, userID, maxConcurrency, requestID)
	if err != nil {
		return nil, err
	}

	if acquired {
		return &AcquireResult{
			Acquired: true,
			ReleaseFunc: func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := s.cache.ReleaseUserSlot(bgCtx, userID, requestID); err != nil {
					logger.LegacyPrintf("service.concurrency", "Warning: failed to release user slot for %d (req=%s): %v", userID, requestID, err)
				}
			},
		}, nil
	}

	return &AcquireResult{
		Acquired:    false,
		ReleaseFunc: nil,
	}, nil
}

// IncrementWaitCount attempts to increment the wait queue counter for a user.
// Returns true if successful, false if the wait queue is full.
func (s *ConcurrencyService) IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error) {
	if s.cache == nil {
		return true, nil
	}

	result, err := s.cache.IncrementWaitCount(ctx, userID, maxWait)
	if err != nil {
		logger.LegacyPrintf("service.concurrency", "Warning: increment wait count failed for user %d: %v", userID, err)
		return true, nil
	}
	if result {
		s.updateCachedUserWaitCountDelta(userID, 1, 1)
	}
	return result, nil
}

// DecrementWaitCount decrements the wait queue counter for a user.
func (s *ConcurrencyService) DecrementWaitCount(ctx context.Context, userID int64) {
	if s.cache == nil {
		return
	}

	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.cache.DecrementWaitCount(bgCtx, userID); err != nil {
		logger.LegacyPrintf("service.concurrency", "Warning: decrement wait count failed for user %d: %v", userID, err)
		return
	}
	s.updateCachedUserWaitCountDelta(userID, -1, 0)
}

// IncrementAccountWaitCount increments the wait queue counter for an account.
func (s *ConcurrencyService) IncrementAccountWaitCount(ctx context.Context, accountID int64, maxWait int) (bool, error) {
	if s.cache == nil {
		return true, nil
	}

	result, err := s.cache.IncrementAccountWaitCount(ctx, accountID, maxWait)
	if err != nil {
		logger.LegacyPrintf("service.concurrency", "Warning: increment wait count failed for account %d: %v", accountID, err)
		return true, nil
	}
	if result {
		s.updateCachedAccountWaitCountDelta(accountID, 1, 1)
	}
	return result, nil
}

// DecrementAccountWaitCount decrements the wait queue counter for an account.
func (s *ConcurrencyService) DecrementAccountWaitCount(ctx context.Context, accountID int64) {
	if s.cache == nil {
		return
	}

	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.cache.DecrementAccountWaitCount(bgCtx, accountID); err != nil {
		logger.LegacyPrintf("service.concurrency", "Warning: decrement wait count failed for account %d: %v", accountID, err)
		return
	}
	s.updateCachedAccountWaitCountDelta(accountID, -1, 0)
}

// GetAccountWaitingCount gets current wait queue count for an account.
func (s *ConcurrencyService) GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error) {
	if s.cache == nil {
		return 0, nil
	}
	now := time.Now()
	if count, ok := s.getCachedAccountWaitCount(accountID, now); ok {
		return count, nil
	}
	count, err := s.cache.GetAccountWaitingCount(ctx, accountID)
	if err != nil {
		return 0, err
	}
	s.storeCachedAccountWaitCount(accountID, count, now)
	return count, nil
}

func (s *ConcurrencyService) EnqueueUserWaitTicket(ctx context.Context, userID int64, ticketID string) error {
	cache := s.ticketQueueCache()
	if cache == nil || strings.TrimSpace(ticketID) == "" {
		return nil
	}
	return cache.EnqueueUserWaitTicket(ctx, userID, ticketID)
}

func (s *ConcurrencyService) IsUserWaitTicketTurn(ctx context.Context, userID int64, ticketID string) (bool, error) {
	cache := s.ticketQueueCache()
	if cache == nil || strings.TrimSpace(ticketID) == "" {
		return true, nil
	}
	return cache.IsUserWaitTicketTurn(ctx, userID, ticketID)
}

func (s *ConcurrencyService) RemoveUserWaitTicket(ctx context.Context, userID int64, ticketID string) {
	cache := s.ticketQueueCache()
	if cache == nil || strings.TrimSpace(ticketID) == "" {
		return
	}
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cache.RemoveUserWaitTicket(bgCtx, userID, ticketID); err != nil {
		logger.LegacyPrintf("service.concurrency", "Warning: remove user wait ticket failed for user %d: %v", userID, err)
	}
}

func (s *ConcurrencyService) EnqueueAccountWaitTicket(ctx context.Context, accountID int64, ticketID string) error {
	cache := s.ticketQueueCache()
	if cache == nil || strings.TrimSpace(ticketID) == "" {
		return nil
	}
	return cache.EnqueueAccountWaitTicket(ctx, accountID, ticketID)
}

func (s *ConcurrencyService) IsAccountWaitTicketTurn(ctx context.Context, accountID int64, ticketID string) (bool, error) {
	cache := s.ticketQueueCache()
	if cache == nil || strings.TrimSpace(ticketID) == "" {
		return true, nil
	}
	return cache.IsAccountWaitTicketTurn(ctx, accountID, ticketID)
}

func (s *ConcurrencyService) RemoveAccountWaitTicket(ctx context.Context, accountID int64, ticketID string) {
	cache := s.ticketQueueCache()
	if cache == nil || strings.TrimSpace(ticketID) == "" {
		return
	}
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cache.RemoveAccountWaitTicket(bgCtx, accountID, ticketID); err != nil {
		logger.LegacyPrintf("service.concurrency", "Warning: remove account wait ticket failed for account %d: %v", accountID, err)
	}
}

// CalculateMaxWait calculates the maximum wait queue size for a user
// maxWait = userConcurrency + defaultExtraWaitSlots
func CalculateMaxWait(userConcurrency int) int {
	if userConcurrency <= 0 {
		userConcurrency = 1
	}
	return userConcurrency + defaultExtraWaitSlots
}

func (s *ConcurrencyService) getCachedAccountWaitCount(accountID int64, now time.Time) (int, bool) {
	if s == nil || accountID <= 0 {
		return 0, false
	}
	value, ok := s.accountWaitCountCache.Load(accountID)
	if !ok {
		return 0, false
	}
	snapshot, ok := value.(cachedAccountWaitCountSnapshot)
	if !ok {
		s.accountWaitCountCache.Delete(accountID)
		return 0, false
	}
	if snapshot.ExpiresAtUnixNano <= now.UnixNano() {
		s.accountWaitCountCache.Delete(accountID)
		return 0, false
	}
	if snapshot.Count < 0 {
		s.accountWaitCountCache.Delete(accountID)
		return 0, false
	}
	return snapshot.Count, true
}

func (s *ConcurrencyService) storeCachedAccountWaitCount(accountID int64, count int, now time.Time) {
	if s == nil || accountID <= 0 {
		return
	}
	if count < 0 {
		count = 0
	}
	ttl := accountWaitCountCacheTTL
	if ttl <= 0 {
		return
	}
	s.accountWaitCountCache.Store(accountID, cachedAccountWaitCountSnapshot{
		Count:             count,
		ExpiresAtUnixNano: now.Add(ttl).UnixNano(),
	})
}

func (s *ConcurrencyService) updateCachedAccountWaitCountDelta(accountID int64, delta int, fallback int) {
	if s == nil || accountID <= 0 {
		return
	}
	now := time.Now()
	count := fallback
	if cached, ok := s.getCachedAccountWaitCount(accountID, now); ok {
		count = cached + delta
	}
	if count < 0 {
		count = 0
	}
	s.storeCachedAccountWaitCount(accountID, count, now)
}

func (s *ConcurrencyService) getCachedUserWaitCount(userID int64, now time.Time) (int, bool) {
	if s == nil || userID <= 0 {
		return 0, false
	}
	value, ok := s.userWaitCountCache.Load(userID)
	if !ok {
		return 0, false
	}
	snapshot, ok := value.(cachedUserWaitCountSnapshot)
	if !ok {
		s.userWaitCountCache.Delete(userID)
		return 0, false
	}
	if snapshot.ExpiresAtUnixNano <= now.UnixNano() {
		s.userWaitCountCache.Delete(userID)
		return 0, false
	}
	if snapshot.Count < 0 {
		s.userWaitCountCache.Delete(userID)
		return 0, false
	}
	return snapshot.Count, true
}

func (s *ConcurrencyService) storeCachedUserWaitCount(userID int64, count int, now time.Time) {
	if s == nil || userID <= 0 {
		return
	}
	if count < 0 {
		count = 0
	}
	ttl := accountWaitCountCacheTTL
	if ttl <= 0 {
		return
	}
	s.userWaitCountCache.Store(userID, cachedUserWaitCountSnapshot{
		Count:             count,
		ExpiresAtUnixNano: now.Add(ttl).UnixNano(),
	})
}

func (s *ConcurrencyService) updateCachedUserWaitCountDelta(userID int64, delta int, fallback int) {
	if s == nil || userID <= 0 {
		return
	}
	now := time.Now()
	count := fallback
	if cached, ok := s.getCachedUserWaitCount(userID, now); ok {
		count = cached + delta
	}
	if count < 0 {
		count = 0
	}
	s.storeCachedUserWaitCount(userID, count, now)
}

func (s *ConcurrencyService) getCachedUserLoadSnapshot(userID int64, now time.Time) (cachedUserLoadSnapshot, bool) {
	if s == nil || userID <= 0 {
		return cachedUserLoadSnapshot{}, false
	}
	value, ok := s.userLoadCache.Load(userID)
	if !ok {
		return cachedUserLoadSnapshot{}, false
	}
	snapshot, ok := value.(cachedUserLoadSnapshot)
	if !ok {
		s.userLoadCache.Delete(userID)
		return cachedUserLoadSnapshot{}, false
	}
	if snapshot.ExpiresAtUnixNano <= now.UnixNano() {
		s.userLoadCache.Delete(userID)
		return cachedUserLoadSnapshot{}, false
	}
	return snapshot, true
}

func (s *ConcurrencyService) storeCachedUserLoadSnapshot(userID int64, currentConcurrency int, waitingCount int, now time.Time) {
	if s == nil || userID <= 0 {
		return
	}
	ttl := accountLoadCacheTTL
	if ttl <= 0 {
		return
	}
	s.userLoadCache.Store(userID, cachedUserLoadSnapshot{
		CurrentConcurrency: currentConcurrency,
		WaitingCount:       waitingCount,
		ExpiresAtUnixNano:  now.Add(ttl).UnixNano(),
	})
}

func accountLoadRate(currentConcurrency int, waitingCount int, maxConcurrency int) int {
	if maxConcurrency <= 0 {
		return 0
	}
	return (currentConcurrency + waitingCount) * 100 / maxConcurrency
}

func accountLoadBatchKey(accounts []AccountWithConcurrency) string {
	if len(accounts) == 0 {
		return ""
	}
	ids := make([]int64, 0, len(accounts))
	seen := make(map[int64]struct{}, len(accounts))
	for _, account := range accounts {
		if account.ID <= 0 {
			continue
		}
		if _, ok := seen[account.ID]; ok {
			continue
		}
		seen[account.ID] = struct{}{}
		ids = append(ids, account.ID)
	}
	if len(ids) == 0 {
		return ""
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var b strings.Builder
	b.Grow(len(ids) * 10)
	b.WriteString("account:")
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatInt(id, 10))
	}
	return b.String()
}

func userLoadBatchKey(users []UserWithConcurrency) string {
	if len(users) == 0 {
		return ""
	}
	ids := make([]int64, 0, len(users))
	seen := make(map[int64]struct{}, len(users))
	for _, user := range users {
		if user.ID <= 0 {
			continue
		}
		if _, ok := seen[user.ID]; ok {
			continue
		}
		seen[user.ID] = struct{}{}
		ids = append(ids, user.ID)
	}
	if len(ids) == 0 {
		return ""
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var b strings.Builder
	b.Grow(len(ids) * 10)
	b.WriteString("user:")
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatInt(id, 10))
	}
	return b.String()
}

func (s *ConcurrencyService) getCachedAccountLoadSnapshot(accountID int64, now time.Time) (cachedAccountLoadSnapshot, bool) {
	if s == nil || accountID <= 0 {
		return cachedAccountLoadSnapshot{}, false
	}
	value, ok := s.accountLoadCache.Load(accountID)
	if !ok {
		return cachedAccountLoadSnapshot{}, false
	}
	snapshot, ok := value.(cachedAccountLoadSnapshot)
	if !ok {
		s.accountLoadCache.Delete(accountID)
		return cachedAccountLoadSnapshot{}, false
	}
	if snapshot.ExpiresAtUnixNano <= now.UnixNano() {
		s.accountLoadCache.Delete(accountID)
		return cachedAccountLoadSnapshot{}, false
	}
	return snapshot, true
}

func (s *ConcurrencyService) storeCachedAccountLoadSnapshot(accountID int64, currentConcurrency int, waitingCount int, now time.Time) {
	if s == nil || accountID <= 0 {
		return
	}
	ttl := accountLoadCacheTTL
	if ttl <= 0 {
		return
	}
	s.accountLoadCache.Store(accountID, cachedAccountLoadSnapshot{
		CurrentConcurrency: currentConcurrency,
		WaitingCount:       waitingCount,
		ExpiresAtUnixNano:  now.Add(ttl).UnixNano(),
	})
}

// GetAccountsLoadBatch returns load info for multiple accounts.
func (s *ConcurrencyService) GetAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	if s.cache == nil {
		return map[int64]*AccountLoadInfo{}, nil
	}
	if len(accounts) == 0 {
		return map[int64]*AccountLoadInfo{}, nil
	}

	now := time.Now()
	result := make(map[int64]*AccountLoadInfo, len(accounts))
	missing := make([]AccountWithConcurrency, 0, len(accounts))
	missingIDs := make(map[int64]struct{}, len(accounts))

	for _, account := range accounts {
		if account.ID <= 0 {
			continue
		}
		if snapshot, ok := s.getCachedAccountLoadSnapshot(account.ID, now); ok {
			result[account.ID] = &AccountLoadInfo{
				AccountID:          account.ID,
				CurrentConcurrency: snapshot.CurrentConcurrency,
				WaitingCount:       snapshot.WaitingCount,
				LoadRate:           accountLoadRate(snapshot.CurrentConcurrency, snapshot.WaitingCount, account.MaxConcurrency),
			}
			continue
		}
		if _, seen := missingIDs[account.ID]; seen {
			continue
		}
		missingIDs[account.ID] = struct{}{}
		missing = append(missing, account)
	}

	if len(missing) == 0 {
		return result, nil
	}

	freshAny, err, _ := s.accountLoadSF.Do(accountLoadBatchKey(missing), func() (any, error) {
		return s.cache.GetAccountsLoadBatch(ctx, missing)
	})
	if err != nil {
		return nil, err
	}
	fresh, _ := freshAny.(map[int64]*AccountLoadInfo)
	if fresh == nil {
		fresh = map[int64]*AccountLoadInfo{}
	}

	refreshAt := time.Now()
	for _, account := range missing {
		info := fresh[account.ID]
		if info == nil {
			info = &AccountLoadInfo{AccountID: account.ID}
		}
		s.storeCachedAccountWaitCount(account.ID, info.WaitingCount, refreshAt)
		s.storeCachedAccountLoadSnapshot(account.ID, info.CurrentConcurrency, info.WaitingCount, refreshAt)
		result[account.ID] = &AccountLoadInfo{
			AccountID:          account.ID,
			CurrentConcurrency: info.CurrentConcurrency,
			WaitingCount:       info.WaitingCount,
			LoadRate:           accountLoadRate(info.CurrentConcurrency, info.WaitingCount, account.MaxConcurrency),
		}
	}

	return result, nil
}

// GetUsersLoadBatch returns load info for multiple users.
func (s *ConcurrencyService) GetUsersLoadBatch(ctx context.Context, users []UserWithConcurrency) (map[int64]*UserLoadInfo, error) {
	if s.cache == nil {
		return map[int64]*UserLoadInfo{}, nil
	}
	if len(users) == 0 {
		return map[int64]*UserLoadInfo{}, nil
	}

	now := time.Now()
	result := make(map[int64]*UserLoadInfo, len(users))
	missing := make([]UserWithConcurrency, 0, len(users))
	missingIDs := make(map[int64]struct{}, len(users))

	for _, user := range users {
		if user.ID <= 0 {
			continue
		}
		if snapshot, ok := s.getCachedUserLoadSnapshot(user.ID, now); ok {
			result[user.ID] = &UserLoadInfo{
				UserID:             user.ID,
				CurrentConcurrency: snapshot.CurrentConcurrency,
				WaitingCount:       snapshot.WaitingCount,
				LoadRate:           accountLoadRate(snapshot.CurrentConcurrency, snapshot.WaitingCount, user.MaxConcurrency),
			}
			continue
		}
		if _, seen := missingIDs[user.ID]; seen {
			continue
		}
		missingIDs[user.ID] = struct{}{}
		missing = append(missing, user)
	}

	if len(missing) == 0 {
		return result, nil
	}

	freshAny, err, _ := s.userLoadSF.Do(userLoadBatchKey(missing), func() (any, error) {
		return s.cache.GetUsersLoadBatch(ctx, missing)
	})
	if err != nil {
		return nil, err
	}
	fresh, _ := freshAny.(map[int64]*UserLoadInfo)
	if fresh == nil {
		fresh = map[int64]*UserLoadInfo{}
	}

	refreshAt := time.Now()
	for _, user := range missing {
		info := fresh[user.ID]
		if info == nil {
			info = &UserLoadInfo{UserID: user.ID}
		}
		s.storeCachedUserWaitCount(user.ID, info.WaitingCount, refreshAt)
		s.storeCachedUserLoadSnapshot(user.ID, info.CurrentConcurrency, info.WaitingCount, refreshAt)
		result[user.ID] = &UserLoadInfo{
			UserID:             user.ID,
			CurrentConcurrency: info.CurrentConcurrency,
			WaitingCount:       info.WaitingCount,
			LoadRate:           accountLoadRate(info.CurrentConcurrency, info.WaitingCount, user.MaxConcurrency),
		}
	}

	return result, nil
}

// CleanupExpiredAccountSlots removes expired slots for one account (background task).
func (s *ConcurrencyService) CleanupExpiredAccountSlots(ctx context.Context, accountID int64) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.CleanupExpiredAccountSlots(ctx, accountID)
}

// StartSlotCleanupWorker starts a background cleanup worker for expired account slots.
func (s *ConcurrencyService) StartSlotCleanupWorker(accountRepo AccountRepository, interval time.Duration) {
	if s == nil || s.cache == nil || accountRepo == nil || interval <= 0 {
		return
	}

	runCleanup := func() {
		listCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		accounts, err := accountRepo.ListSchedulable(listCtx)
		cancel()
		if err != nil {
			logger.LegacyPrintf("service.concurrency", "Warning: list schedulable accounts failed: %v", err)
			return
		}
		for _, account := range accounts {
			accountCtx, accountCancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := s.cache.CleanupExpiredAccountSlots(accountCtx, account.ID)
			accountCancel()
			if err != nil {
				logger.LegacyPrintf("service.concurrency", "Warning: cleanup expired slots failed for account %d: %v", account.ID, err)
			}
		}
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		runCleanup()
		for range ticker.C {
			runCleanup()
		}
	}()
}

// GetAccountConcurrencyBatch gets current concurrency counts for multiple accounts
// Returns a map of accountID -> current concurrency count
func (s *ConcurrencyService) GetAccountConcurrencyBatch(ctx context.Context, accountIDs []int64) (map[int64]int, error) {
	if len(accountIDs) == 0 {
		return map[int64]int{}, nil
	}
	if s.cache == nil {
		result := make(map[int64]int, len(accountIDs))
		for _, accountID := range accountIDs {
			result[accountID] = 0
		}
		return result, nil
	}
	return s.cache.GetAccountConcurrencyBatch(ctx, accountIDs)
}
