package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanProxyMaintenanceAssignments_SplitsFailedProxyAcrossHealthyTargets(t *testing.T) {
	checked := []proxyHealthInspection{
		{
			Proxy:   ProxyWithAccountCount{Proxy: Proxy{ID: 10, Name: "healthy-a"}, AccountCount: 5},
			Success: true,
			Accounts: []ProxyAccountSummary{
				{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5},
			},
		},
		{
			Proxy:   ProxyWithAccountCount{Proxy: Proxy{ID: 20, Name: "healthy-b"}, AccountCount: 5},
			Success: true,
			Accounts: []ProxyAccountSummary{
				{ID: 6}, {ID: 7}, {ID: 8}, {ID: 9}, {ID: 10},
			},
		},
		{
			Proxy:   ProxyWithAccountCount{Proxy: Proxy{ID: 30, Name: "failed"}, AccountCount: 6},
			Success: false,
			Message: "dial timeout",
			Accounts: []ProxyAccountSummary{
				{ID: 11}, {ID: 12}, {ID: 13}, {ID: 14}, {ID: 15}, {ID: 16},
			},
		},
	}

	assignments, unassigned := planProxyMaintenanceAssignments(checked)
	require.Empty(t, unassigned)
	require.Len(t, assignments, 2)

	countByTarget := map[int64]int{}
	for _, assignment := range assignments {
		require.Equal(t, int64(30), assignment.SourceProxyID)
		countByTarget[assignment.TargetProxyID] += assignment.AccountCount
	}
	require.Equal(t, map[int64]int{
		10: 3,
		20: 3,
	}, countByTarget)
}

func TestPlanProxyMaintenanceAssignments_RebalancesHealthyProxiesToNewTarget(t *testing.T) {
	checked := []proxyHealthInspection{
		{
			Proxy:   ProxyWithAccountCount{Proxy: Proxy{ID: 10, Name: "old-a"}, AccountCount: 6},
			Success: true,
			Accounts: []ProxyAccountSummary{
				{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}, {ID: 6},
			},
		},
		{
			Proxy:   ProxyWithAccountCount{Proxy: Proxy{ID: 20, Name: "old-b"}, AccountCount: 6},
			Success: true,
			Accounts: []ProxyAccountSummary{
				{ID: 7}, {ID: 8}, {ID: 9}, {ID: 10}, {ID: 11}, {ID: 12},
			},
		},
		{
			Proxy:    ProxyWithAccountCount{Proxy: Proxy{ID: 30, Name: "new-c"}, AccountCount: 0},
			Success:  true,
			Accounts: nil,
		},
	}

	assignments, unassigned := planProxyMaintenanceAssignments(checked)
	require.Empty(t, unassigned)
	require.Len(t, assignments, 2)

	countBySource := map[int64]int{}
	countByTarget := map[int64]int{}
	for _, assignment := range assignments {
		countBySource[assignment.SourceProxyID] += assignment.AccountCount
		countByTarget[assignment.TargetProxyID] += assignment.AccountCount
	}
	require.Equal(t, map[int64]int{
		10: 2,
		20: 2,
	}, countBySource)
	require.Equal(t, map[int64]int{
		30: 4,
	}, countByTarget)
}

func TestPlanProxyMaintenanceAssignments_BalancedHealthyProxiesDoNotMove(t *testing.T) {
	checked := []proxyHealthInspection{
		{
			Proxy:   ProxyWithAccountCount{Proxy: Proxy{ID: 10, Name: "a"}, AccountCount: 4},
			Success: true,
			Accounts: []ProxyAccountSummary{
				{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4},
			},
		},
		{
			Proxy:   ProxyWithAccountCount{Proxy: Proxy{ID: 20, Name: "b"}, AccountCount: 4},
			Success: true,
			Accounts: []ProxyAccountSummary{
				{ID: 5}, {ID: 6}, {ID: 7}, {ID: 8},
			},
		},
		{
			Proxy:   ProxyWithAccountCount{Proxy: Proxy{ID: 30, Name: "c"}, AccountCount: 4},
			Success: true,
			Accounts: []ProxyAccountSummary{
				{ID: 9}, {ID: 10}, {ID: 11}, {ID: 12},
			},
		},
	}

	assignments, unassigned := planProxyMaintenanceAssignments(checked)
	require.Empty(t, unassigned)
	require.Empty(t, assignments)
}
