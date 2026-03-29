package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	defaultAccountImportWorkerCount    = 2
	defaultAccountImportChunkSize      = 100
	defaultAccountImportQueueTTL       = 24 * time.Hour
	defaultAccountImportMaxFastPathRows = 5000
	defaultAccountImportClaimTTL       = 2 * time.Minute
	defaultAccountImportRecoveryInterval = 30 * time.Second
	defaultAccountImportMaxChunkAttempts = 5
)

type AccountImportAccountStore interface {
	CreateImportPlaceholders(ctx context.Context, accounts []*Account) error
	LookupMinAccountIDsByDedupFingerprint(ctx context.Context, fingerprints []string) (map[string]int64, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*Account, error)
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
	SetSchedulable(ctx context.Context, id int64, schedulable bool) error
	BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error
}

type AccountImportBatchRepository interface {
	EnqueueBatch(ctx context.Context, batch AccountImportBatch, chunkSize int, ttl time.Duration) error
	ClaimNextChunk(ctx context.Context, workerID string, blockTimeout time.Duration, leaseTTL time.Duration) (*AccountImportChunkClaim, error)
	CompleteChunk(ctx context.Context, claim *AccountImportChunkClaim, progress AccountImportBatchProgress) error
	RetryChunk(ctx context.Context, claim *AccountImportChunkClaim, progress AccountImportBatchProgress, maxAttempts int, failureMessage string) error
	RequeueExpiredClaims(ctx context.Context, limit int, leaseTTL time.Duration) (int, error)
}

type AccountImportBatch struct {
	BatchID    string                   `json:"batch_id"`
	Filename   string                   `json:"filename,omitempty"`
	AccountIDs []int64                  `json:"account_ids,omitempty"`
	CreatedAt  time.Time                `json:"created_at"`
	Progress   AccountImportBatchProgress `json:"progress"`
}

type AccountImportBatchProgress struct {
	CompletedChunks   int    `json:"completed_chunks"`
	CompletedAccounts int    `json:"completed_accounts"`
	DuplicateAccounts int    `json:"duplicate_accounts"`
	FailedAccounts    int    `json:"failed_accounts"`
	LastError         string `json:"last_error,omitempty"`
}

type AccountImportChunkClaim struct {
	Token     string  `json:"token"`
	BatchID   string  `json:"batch_id"`
	AccountIDs []int64 `json:"account_ids,omitempty"`
	Attempt   int     `json:"attempt"`
}

type AccountImportPayload struct {
	Filename             string
	GroupIDs             []int64
	SkipDefaultGroupBind bool
	Proxies              []AccountImportProxy
	Accounts             []AccountImportAccount
}

type AccountImportProxy struct {
	ProxyKey string
	Name     string
	Protocol string
	Host     string
	Port     int
	Username string
	Password string
	Status   string
}

type AccountImportAccount struct {
	Name               string
	Notes              *string
	Platform           string
	Type               string
	Credentials        map[string]any
	Extra              map[string]any
	GroupIDs           []int64
	ProxyKey           string
	Concurrency        int
	Priority           int
	RateMultiplier     *float64
	ExpiresAt          *int64
	AutoPauseOnExpired *bool
}

type AccountImportItemError struct {
	Kind     string `json:"kind"`
	Name     string `json:"name,omitempty"`
	ProxyKey string `json:"proxy_key,omitempty"`
	Message  string `json:"message"`
}

type AccountImportResult struct {
	BatchID            string                   `json:"batch_id,omitempty"`
	ProxyCreated       int                      `json:"proxy_created"`
	ProxyReused        int                      `json:"proxy_reused"`
	ProxyFailed        int                      `json:"proxy_failed"`
	AccountEnqueued    int                      `json:"account_enqueued"`
	PlaceholderCreated int                      `json:"placeholder_created"`
	AccountFailed      int                      `json:"account_failed"`
	Errors             []AccountImportItemError `json:"errors,omitempty"`
}

type AccountImportProgressFunc func(stage string, current, total int, message string)

type AccountImportService struct {
	accountStore      AccountImportAccountStore
	batchRepo         AccountImportBatchRepository
	proxyRepo         ProxyRepository
	groupRepo         GroupRepository
	soraAccountRepo   SoraAccountRepository
	schedulerSnapshot *SchedulerSnapshotService
	cfg               *config.Config

	workerCount      int
	chunkSize        int
	queueTTL         time.Duration
	maxFastPathRows  int
	claimTTL         time.Duration
	recoveryInterval time.Duration
	maxChunkAttempts int

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

func NewAccountImportService(
	accountStore AccountImportAccountStore,
	batchRepo AccountImportBatchRepository,
	proxyRepo ProxyRepository,
	groupRepo GroupRepository,
	soraAccountRepo SoraAccountRepository,
	schedulerSnapshot *SchedulerSnapshotService,
	cfg *config.Config,
) *AccountImportService {
	svc := &AccountImportService{
		accountStore:      accountStore,
		batchRepo:         batchRepo,
		proxyRepo:         proxyRepo,
		groupRepo:         groupRepo,
		soraAccountRepo:   soraAccountRepo,
		schedulerSnapshot: schedulerSnapshot,
		cfg:               cfg,
		workerCount:       defaultAccountImportWorkerCount,
		chunkSize:         defaultAccountImportChunkSize,
		queueTTL:          defaultAccountImportQueueTTL,
		maxFastPathRows:   defaultAccountImportMaxFastPathRows,
		claimTTL:          defaultAccountImportClaimTTL,
		recoveryInterval:  defaultAccountImportRecoveryInterval,
		maxChunkAttempts:  defaultAccountImportMaxChunkAttempts,
		stopCh:            make(chan struct{}),
	}
	if cfg != nil {
		if cfg.AccountImport.WorkerCount > 0 {
			svc.workerCount = cfg.AccountImport.WorkerCount
		}
		if cfg.AccountImport.ChunkSize > 0 {
			svc.chunkSize = cfg.AccountImport.ChunkSize
		}
		if cfg.AccountImport.QueueTTL > 0 {
			svc.queueTTL = cfg.AccountImport.QueueTTL
		}
		if cfg.AccountImport.MaxFastPathRows > 0 {
			svc.maxFastPathRows = cfg.AccountImport.MaxFastPathRows
		}
		if cfg.AccountImport.ClaimTTL > 0 {
			svc.claimTTL = cfg.AccountImport.ClaimTTL
		}
		if cfg.AccountImport.RecoveryInterval > 0 {
			svc.recoveryInterval = cfg.AccountImport.RecoveryInterval
		}
		if cfg.AccountImport.MaxChunkAttempts > 0 {
			svc.maxChunkAttempts = cfg.AccountImport.MaxChunkAttempts
		}
	}
	return svc
}

func (s *AccountImportService) Start() {
	if s == nil || s.batchRepo == nil || s.accountStore == nil {
		return
	}
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go func(workerIndex int) {
			defer s.wg.Done()
			s.runWorker(fmt.Sprintf("account-import-%d", workerIndex+1))
		}(i)
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runRecoveryLoop()
	}()
}

func (s *AccountImportService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *AccountImportService) Import(ctx context.Context, payload AccountImportPayload, progress AccountImportProgressFunc) (AccountImportResult, error) {
	result := AccountImportResult{}
	totalRows := len(payload.Proxies) + len(payload.Accounts)
	if s.maxFastPathRows > 0 && len(payload.Accounts) > s.maxFastPathRows {
		return result, fmt.Errorf("import rows exceed fast path limit: %d > %d", len(payload.Accounts), s.maxFastPathRows)
	}
	report := func(stage string, current int, message string) {
		if progress != nil {
			progress(stage, current, totalRows, message)
		}
	}
	report("enqueueing", 0, "Preparing fast-path import")

	importGroupIDs := normalizeImportInt64IDs(payload.GroupIDs)
	if err := s.validateGroups(ctx, importGroupIDs); err != nil {
		return result, err
	}

	proxyIDsByKey, proxyRefsByID, proxyErrs := s.resolveProxies(ctx, payload.Proxies)
	result.ProxyCreated = proxyErrs.ProxyCreated
	result.ProxyReused = proxyErrs.ProxyReused
	result.ProxyFailed = proxyErrs.ProxyFailed
	result.Errors = append(result.Errors, proxyErrs.Errors...)
	report("proxies_ready", len(payload.Proxies), "Resolved proxies for fast-path import")

	batchID := buildAccountImportBatchID()
	defaultGroupsByPlatform := make(map[string][]int64)
	placeholders := make([]*Account, 0, len(payload.Accounts))
	groupBindings := make(map[int64][]int64)
	refreshAccounts := make([]*Account, 0, len(payload.Accounts))
	seenFingerprints := make(map[string]struct{}, len(payload.Accounts))

	for idx := range payload.Accounts {
		spec := payload.Accounts[idx]
		effectiveGroupIDs, err := s.resolveEffectiveGroupIDs(ctx, spec.Platform, normalizeImportInt64IDs(spec.GroupIDs), importGroupIDs, payload.SkipDefaultGroupBind, defaultGroupsByPlatform)
		if err != nil {
			result.AccountFailed++
			result.Errors = append(result.Errors, AccountImportItemError{Kind: "account", Name: spec.Name, Message: err.Error()})
			continue
		}

		var proxyID *int64
		var proxyRef *Proxy
		if key := strings.TrimSpace(spec.ProxyKey); key != "" {
			id, ok := proxyIDsByKey[key]
			if !ok {
				result.AccountFailed++
				result.Errors = append(result.Errors, AccountImportItemError{
					Kind:     "account",
					Name:     spec.Name,
					ProxyKey: key,
					Message:  "proxy_key not found",
				})
				continue
			}
			proxyID = &id
			if ref := proxyRefsByID[id]; ref != nil {
				proxyCopy := *ref
				proxyRef = &proxyCopy
			}
		}

		account := importedAccountToService(spec, proxyID)
		account.GroupIDs = effectiveGroupIDs
		account.Proxy = proxyRef
		account.SetSyncMetadata(AccountSyncStateSyncing, 25, "Queued for background sync", batchID, nil)
		fingerprint := buildImportedAccountFingerprint(spec)
		account.SetDedupFingerprint(fingerprint)
		if fingerprint != "" {
			if _, exists := seenFingerprints[fingerprint]; exists {
				result.AccountFailed++
				result.Errors = append(result.Errors, AccountImportItemError{
					Kind:    "account",
					Name:    spec.Name,
					Message: "duplicate account detected in import payload",
				})
				continue
			}
			seenFingerprints[fingerprint] = struct{}{}
		}
		placeholders = append(placeholders, account)
	}

	if len(placeholders) == 0 {
		if len(result.Errors) > 0 {
			return result, nil
		}
		return result, errors.New("no accounts available for import")
	}

	if err := s.accountStore.CreateImportPlaceholders(ctx, placeholders); err != nil {
		return result, err
	}
	result.PlaceholderCreated = len(placeholders)
	report("placeholders_ready", len(payload.Proxies)+len(placeholders), "Created placeholder accounts")

	for _, account := range placeholders {
		for _, groupID := range normalizeImportInt64IDs(account.GroupIDs) {
			groupBindings[groupID] = append(groupBindings[groupID], account.ID)
		}
		refreshAccounts = append(refreshAccounts, account)
	}

	if err := s.bindGroups(ctx, groupBindings); err != nil {
		return result, err
	}

	if s.schedulerSnapshot != nil {
		if err := s.schedulerSnapshot.RefreshAccounts(ctx, refreshAccounts, "account_import_fast_path"); err != nil {
			logger.LegacyPrintf("service.account_import", "[AccountImport] scheduler refresh failed: batch=%s err=%v", batchID, err)
		}
	}

	accountIDs := make([]int64, 0, len(placeholders))
	for _, account := range placeholders {
		accountIDs = append(accountIDs, account.ID)
	}
	batch := AccountImportBatch{
		BatchID:    batchID,
		Filename:   strings.TrimSpace(payload.Filename),
		AccountIDs: accountIDs,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.batchRepo.EnqueueBatch(ctx, batch, s.chunkSize, s.queueTTL); err != nil {
		s.markAccountsSyncFailed(context.Background(), accountIDs, err.Error())
		return result, err
	}

	result.BatchID = batchID
	result.AccountEnqueued = len(accountIDs)
	report("batch_enqueued", totalRows, "Fast-path import completed")
	return result, nil
}

func (s *AccountImportService) runWorker(workerID string) {
	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		claim, err := s.batchRepo.ClaimNextChunk(context.Background(), workerID, time.Second, s.claimTTL)
		if err != nil {
			logger.LegacyPrintf("service.account_import", "[AccountImport] claim chunk failed: worker=%s err=%v", workerID, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if claim == nil {
			continue
		}
		if err := s.processClaim(context.Background(), claim); err != nil {
			progress := AccountImportBatchProgress{LastError: err.Error()}
			if claim.Attempt+1 >= s.maxChunkAttempts {
				progress.FailedAccounts = len(claim.AccountIDs)
				s.markAccountsSyncFailed(context.Background(), claim.AccountIDs, err.Error())
			}
			if retryErr := s.batchRepo.RetryChunk(context.Background(), claim, progress, s.maxChunkAttempts, err.Error()); retryErr != nil {
				logger.LegacyPrintf("service.account_import", "[AccountImport] retry chunk failed: batch=%s err=%v", claim.BatchID, retryErr)
			}
		}
	}
}

func (s *AccountImportService) runRecoveryLoop() {
	ticker := time.NewTicker(s.recoveryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := s.batchRepo.RequeueExpiredClaims(context.Background(), s.workerCount*s.chunkSize, s.claimTTL); err != nil {
				logger.LegacyPrintf("service.account_import", "[AccountImport] requeue expired claims failed: err=%v", err)
			}
		case <-s.stopCh:
			return
		}
	}
}

func (s *AccountImportService) processClaim(ctx context.Context, claim *AccountImportChunkClaim) error {
	if claim == nil || len(claim.AccountIDs) == 0 {
		return nil
	}
	accounts, err := s.accountStore.GetByIDs(ctx, claim.AccountIDs)
	if err != nil {
		return err
	}
	fingerprints := make([]string, 0, len(accounts))
	accountsByID := make(map[int64]*Account, len(accounts))
	for _, account := range accounts {
		if account == nil || account.ID <= 0 {
			continue
		}
		accountsByID[account.ID] = account
		if strings.TrimSpace(account.DedupFingerprint) != "" {
			fingerprints = append(fingerprints, account.DedupFingerprint)
		}
	}
	minIDsByFingerprint, err := s.accountStore.LookupMinAccountIDsByDedupFingerprint(ctx, fingerprints)
	if err != nil {
		return err
	}

	progress := AccountImportBatchProgress{}
	for _, accountID := range claim.AccountIDs {
		account := accountsByID[accountID]
		if account == nil {
			progress.CompletedAccounts++
			continue
		}
		duplicateOf := int64(0)
		if fingerprint := strings.TrimSpace(account.DedupFingerprint); fingerprint != "" {
			duplicateOf = minIDsByFingerprint[fingerprint]
		}
		if duplicateOf > 0 && duplicateOf != account.ID {
			if err := s.accountStore.BindGroups(ctx, account.ID, nil); err != nil {
				return err
			}
			if err := s.accountStore.SetSchedulable(ctx, account.ID, false); err != nil {
				return err
			}
			if err := s.accountStore.UpdateExtra(ctx, account.ID, map[string]any{
				AccountExtraSyncStateKey:    AccountSyncStateDuplicate,
				AccountExtraSyncProgressKey: 100,
				AccountExtraSyncMessageKey:  fmt.Sprintf("duplicate of account %d", duplicateOf),
				AccountExtraDuplicateOfKey:  duplicateOf,
			}); err != nil {
				return err
			}
			progress.DuplicateAccounts++
			progress.CompletedAccounts++
			continue
		}

		if account.Platform == PlatformSora && s.soraAccountRepo != nil {
			if err := s.soraAccountRepo.Upsert(ctx, account.ID, map[string]any{
				"access_token":  account.GetCredential("access_token"),
				"refresh_token": account.GetCredential("refresh_token"),
				"session_token": account.GetCredential("session_token"),
			}); err != nil {
				if updateErr := s.accountStore.UpdateExtra(ctx, account.ID, map[string]any{
					AccountExtraSyncStateKey:    AccountSyncStateFailed,
					AccountExtraSyncProgressKey: 100,
					AccountExtraSyncMessageKey:  err.Error(),
				}); updateErr != nil {
					logger.LegacyPrintf("service.account_import", "[AccountImport] mark sora sync failed: account=%d err=%v", account.ID, updateErr)
				}
				progress.FailedAccounts++
				progress.CompletedAccounts++
				continue
			}
		}

		if err := s.accountStore.UpdateExtra(ctx, account.ID, map[string]any{
			AccountExtraSyncStateKey:    AccountSyncStateCompleted,
			AccountExtraSyncProgressKey: 100,
			AccountExtraSyncMessageKey:  "Background sync completed",
		}); err != nil {
			return err
		}
		progress.CompletedAccounts++
	}

	return s.batchRepo.CompleteChunk(ctx, claim, progress)
}

func (s *AccountImportService) bindGroups(ctx context.Context, bindings map[int64][]int64) error {
	for groupID, accountIDs := range bindings {
		accountIDs = normalizeImportInt64IDs(accountIDs)
		if groupID <= 0 || len(accountIDs) == 0 || s.groupRepo == nil {
			continue
		}
		if err := s.groupRepo.BindAccountsToGroup(ctx, groupID, accountIDs); err != nil {
			return err
		}
	}
	return nil
}

func (s *AccountImportService) validateGroups(ctx context.Context, groupIDs []int64) error {
	if len(groupIDs) == 0 || s.groupRepo == nil {
		return nil
	}
	for _, groupID := range groupIDs {
		if groupID <= 0 {
			return ErrGroupNotFound
		}
		if _, err := s.groupRepo.GetByID(ctx, groupID); err != nil {
			return err
		}
	}
	return nil
}

func (s *AccountImportService) resolveEffectiveGroupIDs(ctx context.Context, platform string, accountGroupIDs, importGroupIDs []int64, skipDefault bool, cache map[string][]int64) ([]int64, error) {
	groupIDs := normalizeImportInt64IDs(accountGroupIDs)
	if len(groupIDs) == 0 {
		groupIDs = normalizeImportInt64IDs(importGroupIDs)
	}
	if len(groupIDs) > 0 {
		if err := s.validateGroups(ctx, groupIDs); err != nil {
			return nil, err
		}
		return groupIDs, nil
	}
	if skipDefault || s.groupRepo == nil {
		return nil, nil
	}
	platform = strings.TrimSpace(platform)
	if cached, ok := cache[platform]; ok {
		return cached, nil
	}
	groups, err := s.groupRepo.ListActiveByPlatform(ctx, platform)
	if err != nil {
		return nil, err
	}
	defaultName := platform + "-default"
	for _, group := range groups {
		if group.Name == defaultName {
			cache[platform] = []int64{group.ID}
			return cache[platform], nil
		}
	}
	cache[platform] = nil
	return nil, nil
}

type proxyResolutionResult struct {
	ProxyCreated int
	ProxyReused  int
	ProxyFailed  int
	Errors       []AccountImportItemError
}

func (s *AccountImportService) resolveProxies(ctx context.Context, specs []AccountImportProxy) (map[string]int64, map[int64]*Proxy, proxyResolutionResult) {
	result := proxyResolutionResult{}
	keyToID := make(map[string]int64, len(specs))
	refsByID := make(map[int64]*Proxy)
	if len(specs) == 0 {
		return keyToID, refsByID, result
	}

	existingProxies, err := s.listAllProxies(ctx)
	if err != nil {
		result.ProxyFailed = len(specs)
		for _, spec := range specs {
			result.Errors = append(result.Errors, AccountImportItemError{
				Kind:     "proxy",
				Name:     spec.Name,
				ProxyKey: strings.TrimSpace(spec.ProxyKey),
				Message:  err.Error(),
			})
		}
		return keyToID, refsByID, result
	}
	for i := range existingProxies {
		proxy := existingProxies[i]
		key := buildImportProxyKey(proxy.Protocol, proxy.Host, proxy.Port, proxy.Username, proxy.Password)
		keyToID[key] = proxy.ID
		proxyCopy := proxy
		refsByID[proxy.ID] = &proxyCopy
	}

	for _, spec := range specs {
		key := strings.TrimSpace(spec.ProxyKey)
		if key == "" {
			key = buildImportProxyKey(spec.Protocol, spec.Host, spec.Port, spec.Username, spec.Password)
		}
		if existingID, ok := keyToID[key]; ok && existingID > 0 {
			result.ProxyReused++
			continue
		}
		proxy := &Proxy{
			Name:     defaultImportProxyName(spec.Name),
			Protocol: spec.Protocol,
			Host:     spec.Host,
			Port:     spec.Port,
			Username: spec.Username,
			Password: spec.Password,
			Status:   normalizeImportProxyStatus(spec.Status),
		}
		if proxy.Status == "" {
			proxy.Status = StatusActive
		}
		if err := s.proxyRepo.Create(ctx, proxy); err != nil {
			result.ProxyFailed++
			result.Errors = append(result.Errors, AccountImportItemError{
				Kind:     "proxy",
				Name:     spec.Name,
				ProxyKey: key,
				Message:  err.Error(),
			})
			continue
		}
		keyToID[key] = proxy.ID
		proxyCopy := *proxy
		refsByID[proxy.ID] = &proxyCopy
		result.ProxyCreated++
	}
	return keyToID, refsByID, result
}

func (s *AccountImportService) listAllProxies(ctx context.Context) ([]Proxy, error) {
	if s.proxyRepo == nil {
		return nil, errors.New("proxy repository not configured")
	}
	page := 1
	pageSize := 500
	out := make([]Proxy, 0, pageSize)
	for {
		items, result, err := s.proxyRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: page, PageSize: pageSize}, "", "", "")
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if result == nil || len(out) >= int(result.Total) || len(items) == 0 {
			break
		}
		page++
	}
	return out, nil
}

func (s *AccountImportService) markAccountsSyncFailed(ctx context.Context, ids []int64, message string) {
	for _, id := range normalizeImportInt64IDs(ids) {
		_ = s.accountStore.UpdateExtra(ctx, id, map[string]any{
			AccountExtraSyncStateKey:    AccountSyncStateFailed,
			AccountExtraSyncProgressKey: 100,
			AccountExtraSyncMessageKey:  strings.TrimSpace(message),
		})
	}
}

func normalizeImportInt64IDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func importedAccountToService(spec AccountImportAccount, proxyID *int64) *Account {
	account := &Account{
		Name:        spec.Name,
		Notes:       spec.Notes,
		Platform:    spec.Platform,
		Type:        spec.Type,
		Credentials: copyImportJSONMap(spec.Credentials),
		Extra:       copyImportJSONMap(spec.Extra),
		ProxyID:     proxyID,
		Concurrency: spec.Concurrency,
		Priority:    spec.Priority,
		Status:      StatusActive,
		Schedulable: true,
	}
	if spec.RateMultiplier != nil {
		account.RateMultiplier = spec.RateMultiplier
	}
	if spec.ExpiresAt != nil && *spec.ExpiresAt > 0 {
		expiresAt := time.Unix(*spec.ExpiresAt, 0)
		account.ExpiresAt = &expiresAt
	}
	if spec.AutoPauseOnExpired != nil {
		account.AutoPauseOnExpired = *spec.AutoPauseOnExpired
	} else {
		account.AutoPauseOnExpired = true
	}
	return account
}

func buildImportedAccountFingerprint(spec AccountImportAccount) string {
	get := func(key string) string {
		if spec.Credentials == nil {
			return ""
		}
		raw, ok := spec.Credentials[key]
		if !ok || raw == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprintf("%v", raw))
	}

	platform := strings.ToLower(strings.TrimSpace(spec.Platform))
	accountType := strings.ToLower(strings.TrimSpace(spec.Type))
	baseURL := strings.ToLower(strings.TrimSpace(get("base_url")))
	name := strings.ToLower(strings.TrimSpace(spec.Name))

	switch {
	case get("api_key") != "":
		return fmt.Sprintf("%s|%s|api_key|%s|%s", platform, accountType, strings.TrimSpace(get("api_key")), baseURL)
	case get("chatgpt_account_id") != "":
		return fmt.Sprintf("%s|%s|chatgpt_account_id|%s", platform, accountType, strings.TrimSpace(get("chatgpt_account_id")))
	case get("chatgpt_user_id") != "":
		return fmt.Sprintf("%s|%s|chatgpt_user_id|%s", platform, accountType, strings.TrimSpace(get("chatgpt_user_id")))
	case get("project_id") != "":
		return fmt.Sprintf("%s|%s|project_id|%s", platform, accountType, strings.TrimSpace(get("project_id")))
	case get("refresh_token") != "":
		return fmt.Sprintf("%s|%s|refresh_token|%s", platform, accountType, strings.TrimSpace(get("refresh_token")))
	case get("access_token") != "":
		return fmt.Sprintf("%s|%s|access_token|%s|%s|%s", platform, accountType, strings.TrimSpace(get("access_token")), name, baseURL)
	case get("email") != "":
		return fmt.Sprintf("%s|%s|email|%s|%s", platform, accountType, strings.ToLower(strings.TrimSpace(get("email"))), baseURL)
	default:
		return ""
	}
}

func buildImportProxyKey(protocol, host string, port int, username, password string) string {
	return fmt.Sprintf("%s|%s|%d|%s|%s", strings.TrimSpace(protocol), strings.TrimSpace(host), port, strings.TrimSpace(username), strings.TrimSpace(password))
}

func normalizeImportProxyStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", StatusActive:
		return StatusActive
	case "inactive", StatusDisabled:
		return StatusDisabled
	case StatusError:
		return StatusError
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func defaultImportProxyName(name string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return "imported-proxy"
}

func copyImportJSONMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func buildAccountImportBatchID() string {
	return fmt.Sprintf("import-%d", time.Now().UTC().UnixNano())
}
