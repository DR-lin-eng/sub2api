//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildScheduledTestRunUpdate_PausesImmediatelyOnAccountNotFound(t *testing.T) {
	plan := &ScheduledTestPlan{
		ID:                  1,
		Enabled:             true,
		ConsecutiveFailures: 0,
		CronExpression:      "* * * * *",
	}
	result := &ScheduledTestResult{
		Status:       "failed",
		ErrorMessage: "Account not found",
	}

	update, err := buildScheduledTestRunUpdate(plan, result, time.Now())
	require.NoError(t, err)
	require.False(t, update.Enabled)
	require.Nil(t, update.NextRunAt)
	require.NotNil(t, update.PausedAt)
	require.Contains(t, update.PauseReason, "unrecoverable")
}

func TestBuildScheduledTestRunUpdate_PausesImmediatelyOnTokenInvalidated(t *testing.T) {
	plan := &ScheduledTestPlan{
		ID:                  2,
		Enabled:             true,
		ConsecutiveFailures: 1,
		CronExpression:      "* * * * *",
	}
	result := &ScheduledTestResult{
		Status:       "failed",
		ErrorMessage: `API returned 401: {"error":{"code":"token_invalidated"}}`,
	}

	update, err := buildScheduledTestRunUpdate(plan, result, time.Now())
	require.NoError(t, err)
	require.False(t, update.Enabled)
	require.Nil(t, update.NextRunAt)
	require.NotNil(t, update.PausedAt)
}

func TestShouldPauseScheduledTestImmediately(t *testing.T) {
	require.True(t, shouldPauseScheduledTestImmediately("Account not found"))
	require.True(t, shouldPauseScheduledTestImmediately("Your authentication token has been invalidated. Please try signing in again."))
	require.True(t, shouldPauseScheduledTestImmediately("API returned 401: token_invalidated"))
	require.False(t, shouldPauseScheduledTestImmediately("Request failed: socks connect tcp EOF"))
}
