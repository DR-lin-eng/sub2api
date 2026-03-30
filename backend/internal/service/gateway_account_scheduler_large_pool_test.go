//go:build unit

package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayService_SelectAccountWithLoadAwareness_LargePoolEscapesInitialHotset(t *testing.T) {
	ctx := context.Background()
	accounts := make([]Account, 0, 256)
	accountsByID := make(map[int64]*Account, 256)
	loadMap := make(map[int64]*AccountLoadInfo, 256)
	for i := 0; i < 256; i++ {
		accountID := int64(9000 + i)
		accounts = append(accounts, Account{
			ID:          accountID,
			Platform:    PlatformAnthropic,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		})
	}
	for i := range accounts {
		account := &accounts[i]
		accountsByID[account.ID] = account
		loadMap[account.ID] = &AccountLoadInfo{
			AccountID:          account.ID,
			CurrentConcurrency: 0,
			WaitingCount:       0,
			LoadRate:           0,
		}
	}

	cfg := testConfig()
	cfg.Gateway.Scheduling.LoadBatchEnabled = true
	cfg.Gateway.Scheduling.LBTopK = 7
	cfg.Gateway.Scheduling.SchedulerScoreWeights.Priority = 1
	cfg.Gateway.Scheduling.SchedulerScoreWeights.Load = 1
	cfg.Gateway.Scheduling.SchedulerScoreWeights.Queue = 1
	cfg.Gateway.Scheduling.SchedulerScoreWeights.ErrorRate = 1
	cfg.Gateway.Scheduling.SchedulerScoreWeights.TTFT = 1

	svc := &GatewayService{
		accountRepo:        &mockAccountRepoForPlatform{accounts: accounts, accountsByID: accountsByID},
		cache:              &mockGatewayCacheForPlatform{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(&mockConcurrencyCache{loadMap: loadMap}),
	}

	selected := make(map[int64]int)
	var maxSelectedID int64
	for i := 0; i < 256; i++ {
		result, err := svc.SelectAccountWithLoadAwareness(
			ctx,
			nil,
			fmt.Sprintf("gateway_large_pool_%03d", i),
			"claude-3-5-sonnet-20241022",
			nil,
			"",
		)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Account)
		selected[result.Account.ID]++
		if result.Account.ID > maxSelectedID {
			maxSelectedID = result.Account.ID
		}
		if result.ReleaseFunc != nil {
			result.ReleaseFunc()
		}
	}

	require.Greater(t, len(selected), 8, "large pool scheduling should still fan out inside the expanded old-account window")
	require.Greater(t, maxSelectedID, int64(9016), "selection should reach beyond the old fixed 16-account hotset")
}
