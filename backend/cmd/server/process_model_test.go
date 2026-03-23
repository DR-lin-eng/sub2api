package main

import (
	"runtime"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestIsMasterWorkerModeEnabled(t *testing.T) {
	require.False(t, isMasterWorkerModeEnabled(nil))
	require.False(t, isMasterWorkerModeEnabled(&config.Config{
		Process: config.ProcessConfig{Mode: config.ProcessModeSingle},
	}))
	require.True(t, isMasterWorkerModeEnabled(&config.Config{
		Process: config.ProcessConfig{Mode: config.ProcessModeMasterWorker},
	}))
}

func TestResolvedWorkerCount(t *testing.T) {
	require.Equal(t, 1, resolvedWorkerCount(nil))
	require.Equal(t, 8, resolvedWorkerCount(&config.Config{
		Process: config.ProcessConfig{WorkerCount: 8},
	}))

	autoCount := resolvedWorkerCount(&config.Config{
		Process: config.ProcessConfig{WorkerCount: 0},
	})
	require.Equal(t, runtime.NumCPU(), autoCount)
}

func TestProcessTimeoutHelpersUseConfig(t *testing.T) {
	cfg := &config.Config{
		Process: config.ProcessConfig{
			GracefulShutdownTimeoutSeconds: 12,
			WorkerReadyTimeoutSeconds:      18,
			ReloadTimeoutSeconds:           70,
			RespawnBackoffMS:               2500,
		},
	}

	require.Equal(t, 12*time.Second, gracefulShutdownTimeout(cfg))
	require.Equal(t, 18*time.Second, workerReadyTimeout(cfg))
	require.Equal(t, 70*time.Second, reloadTimeout(cfg))
	require.Equal(t, 2500*time.Millisecond, respawnBackoff(cfg))
}

func TestProcessTimeoutHelpersFallbackToDefaults(t *testing.T) {
	cfg := &config.Config{}

	require.Equal(t, defaultGracefulShutdown, gracefulShutdownTimeout(cfg))
	require.Equal(t, defaultWorkerReadyTimeout, workerReadyTimeout(cfg))
	require.Equal(t, defaultReloadTimeout, reloadTimeout(cfg))
	require.Equal(t, defaultRespawnBackoff, respawnBackoff(cfg))
}

func TestParseEnvInt(t *testing.T) {
	t.Setenv("PROCESS_MODEL_TEST_INT", "")
	require.Equal(t, 0, parseEnvInt("PROCESS_MODEL_TEST_INT"))

	t.Setenv("PROCESS_MODEL_TEST_INT", " 17 ")
	require.Equal(t, 17, parseEnvInt("PROCESS_MODEL_TEST_INT"))

	t.Setenv("PROCESS_MODEL_TEST_INT", "bad")
	require.Equal(t, 0, parseEnvInt("PROCESS_MODEL_TEST_INT"))
}

func TestCurrentWorkerHelpersReadEnvironment(t *testing.T) {
	t.Setenv(processWorkerIndexEnv, "3")
	t.Setenv(processWorkerGenerationEnv, "42")

	require.Equal(t, 3, currentWorkerIndex())
	require.Equal(t, 42, currentWorkerGeneration())
}

func TestIsWorkerProcessIsCaseInsensitive(t *testing.T) {
	t.Setenv(processRoleEnv, "WORKER")
	require.True(t, isWorkerProcess())

	t.Setenv(processRoleEnv, " coordinator ")
	require.False(t, isWorkerProcess())
}

func TestInheritedFDFromEnv(t *testing.T) {
	t.Setenv(processReadyFDEnv, "")
	require.Equal(t, uintptr(0), inheritedFDFromEnv(processReadyFDEnv))

	t.Setenv(processReadyFDEnv, " 4 ")
	require.Equal(t, uintptr(4), inheritedFDFromEnv(processReadyFDEnv))

	t.Setenv(processReadyFDEnv, "bad")
	require.Equal(t, uintptr(0), inheritedFDFromEnv(processReadyFDEnv))
}
