package service

import (
	"testing"
	"time"
)

func TestBuildConcurrencyStats_SkipsAccountMapWhenNotRequested(t *testing.T) {
	groupA := &Group{ID: 10, Name: "A", Platform: PlatformOpenAI}
	groupB := &Group{ID: 20, Name: "B", Platform: PlatformOpenAI}
	accounts := []Account{
		{ID: 1, Name: "acc-1", Platform: PlatformOpenAI, Concurrency: 8, Groups: []*Group{groupA}},
		{ID: 2, Name: "acc-2", Platform: PlatformOpenAI, Concurrency: 4, Groups: []*Group{groupA, groupB}},
	}
	loadMap := map[int64]*AccountLoadInfo{
		1: {AccountID: 1, CurrentConcurrency: 3, WaitingCount: 1},
		2: {AccountID: 2, CurrentConcurrency: 1, WaitingCount: 0},
	}

	platform, group, account := buildConcurrencyStats(accounts, loadMap, nil, false)
	if len(account) != 0 {
		t.Fatalf("expected no account-level rows when includeAccount=false, got %d", len(account))
	}
	if got := platform[PlatformOpenAI]; got == nil || got.MaxCapacity != 12 || got.CurrentInUse != 4 || got.WaitingInQueue != 1 {
		t.Fatalf("unexpected platform aggregate: %#v", got)
	}
	if got := group[groupA.ID]; got == nil || got.MaxCapacity != 12 || got.CurrentInUse != 4 || got.WaitingInQueue != 1 {
		t.Fatalf("unexpected group A aggregate: %#v", got)
	}
	if got := group[groupB.ID]; got == nil || got.MaxCapacity != 4 || got.CurrentInUse != 1 || got.WaitingInQueue != 0 {
		t.Fatalf("unexpected group B aggregate: %#v", got)
	}
}

func TestBuildAccountAvailabilityStats_SkipsAccountMapWhenNotRequested(t *testing.T) {
	group := &Group{ID: 10, Name: "A", Platform: PlatformOpenAI}
	rateLimitedUntil := time.Now().Add(5 * time.Minute)
	accounts := []Account{
		{ID: 1, Name: "healthy", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, Groups: []*Group{group}},
		{ID: 2, Name: "limited", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, RateLimitResetAt: &rateLimitedUntil, Groups: []*Group{group}},
	}

	platform, grouped, account := buildAccountAvailabilityStats(accounts, nil, false, time.Now())
	if len(account) != 0 {
		t.Fatalf("expected no account-level availability rows when includeAccount=false, got %d", len(account))
	}
	if got := platform[PlatformOpenAI]; got == nil || got.TotalAccounts != 2 || got.AvailableCount != 1 || got.RateLimitCount != 1 {
		t.Fatalf("unexpected platform availability aggregate: %#v", got)
	}
	if got := grouped[group.ID]; got == nil || got.TotalAccounts != 2 || got.AvailableCount != 1 || got.RateLimitCount != 1 {
		t.Fatalf("unexpected group availability aggregate: %#v", got)
	}
}

func TestSummarizeGatewaySchedulerPool(t *testing.T) {
	accounts := []Account{
		{ID: 1},
		{ID: 2},
		{ID: 3},
		{ID: 0},
	}
	loadMap := map[int64]*AccountLoadInfo{
		1: {AccountID: 1, CurrentConcurrency: 2},
		2: {AccountID: 2, WaitingCount: 3},
		3: {AccountID: 3, CurrentConcurrency: 0, WaitingCount: 0},
	}

	poolAccountsTotal, activeSchedulingAccounts := summarizeGatewaySchedulerPool(accounts, loadMap)
	if poolAccountsTotal != 3 {
		t.Fatalf("expected 3 pool accounts, got %d", poolAccountsTotal)
	}
	if activeSchedulingAccounts != 2 {
		t.Fatalf("expected 2 active scheduling accounts, got %d", activeSchedulingAccounts)
	}
}
