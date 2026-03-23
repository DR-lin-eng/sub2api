package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsCoordinatorProcessIsCaseInsensitive(t *testing.T) {
	t.Setenv(processRoleEnv, "COORDINATOR")
	require.True(t, isCoordinatorProcess())

	t.Setenv(processRoleEnv, " worker ")
	require.False(t, isCoordinatorProcess())
}

func TestCoordinatorRoleConstantIsStable(t *testing.T) {
	require.Equal(t, "coordinator", processRoleCoordinator)
}
