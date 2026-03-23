//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldSuppressAccountTestErrorLog(t *testing.T) {
	require.True(t, shouldSuppressAccountTestErrorLog("Account not found"))
	require.True(t, shouldSuppressAccountTestErrorLog("context canceled"))
	require.True(t, shouldSuppressAccountTestErrorLog(`API returned 401: {"error":{"code":"token_invalidated"}}`))
	require.False(t, shouldSuppressAccountTestErrorLog("Request failed: socks connect tcp EOF"))
}
