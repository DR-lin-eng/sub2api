package service

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	defaultAccountModelsRefreshInterval       = time.Minute
	defaultAccountModelsRefreshRunTimeout     = 2 * time.Minute
	defaultAccountModelsRefreshMaxConcurrency = 4
)

// AccountModelsRefreshService periodically refreshes fetched upstream model lists
// for accounts that opted into auto-refresh via extra.models_refresh_interval_seconds.
type AccountModelsRefreshService struct {
	accountRepo    AccountRepository
	accountTestSvc *AccountTestService
	interval       time.Duration
	stopCh         chan struct{}
	stopOnce       sync.Once
	wg             sync.WaitGroup
	runInProgress  atomic.Bool
}

func NewAccountModelsRefreshService(
	accountRepo AccountRepository,
	accountTestSvc *AccountTestService,
	interval time.Duration,
) *AccountModelsRefreshService {
	if interval <= 0 {
		interval = defaultAccountModelsRefreshInterval
	}
	return &AccountModelsRefreshService{
		accountRepo:    accountRepo,
		accountTestSvc: accountTestSvc,
		interval:       interval,
		stopCh:         make(chan struct{}),
	}
}

func (s *AccountModelsRefreshService) Start() {
	if s == nil || s.accountRepo == nil || s.accountTestSvc == nil || s.interval <= 0 {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *AccountModelsRefreshService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *AccountModelsRefreshService) runOnce() {
	if s == nil || s.accountRepo == nil || s.accountTestSvc == nil {
		return
	}
	if !s.runInProgress.CompareAndSwap(false, true) {
		return
	}
	defer s.runInProgress.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), defaultAccountModelsRefreshRunTimeout)
	defer cancel()

	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		log.Printf("[AccountModelsRefresh] list active accounts failed: %v", err)
		return
	}
	if len(accounts) == 0 {
		return
	}

	now := time.Now()
	dueIDs := make([]int64, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if !supportsFetchedModelsRefresh(account) || !account.ShouldRefreshFetchedModels(now) {
			continue
		}
		dueIDs = append(dueIDs, account.ID)
	}
	if len(dueIDs) == 0 {
		return
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(defaultAccountModelsRefreshMaxConcurrency)
	for _, accountID := range dueIDs {
		accountID := accountID
		g.Go(func() error {
			if _, err := s.accountTestSvc.FetchAndCacheAvailableModels(gctx, accountID); err != nil {
				log.Printf("[AccountModelsRefresh] refresh account=%d failed: %v", accountID, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func supportsFetchedModelsRefresh(account *Account) bool {
	if account == nil {
		return false
	}
	if account.IsOpenAI() || account.IsGemini() {
		return true
	}
	return account.IsAnthropic() && !account.IsBedrock()
}
