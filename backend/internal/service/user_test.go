//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUser_EffectiveConcurrency(t *testing.T) {
	require.Equal(t, 0, (*User)(nil).EffectiveConcurrency())
	require.Equal(t, 8, (&User{Concurrency: 8}).EffectiveConcurrency())
	require.Equal(t, 0, (&User{Concurrency: 8, UnlimitedConcurrency: true}).EffectiveConcurrency())
}
