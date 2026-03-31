package ffi

import (
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type runtimeState struct {
	mu      sync.RWMutex
	cfg     config.RustFFIConfig
	lib     *dynamicLibrary
	loadErr error
}

var globalRuntime runtimeState

func Configure(cfg config.RustFFIConfig) error {
	globalRuntime.mu.Lock()
	defer globalRuntime.mu.Unlock()

	globalRuntime.cfg = cfg
	globalRuntime.loadErr = nil

	featureEnabled := cfg.HashEnabled || cfg.StreamingEnabled || cfg.CompressionEnabled
	if !cfg.Enabled || !featureEnabled || strings.TrimSpace(cfg.LibraryPath) == "" {
		if globalRuntime.lib != nil {
			_ = globalRuntime.lib.Close()
			globalRuntime.lib = nil
		}
		return nil
	}

	if globalRuntime.lib != nil && globalRuntime.lib.path == strings.TrimSpace(cfg.LibraryPath) {
		return nil
	}

	nextLib, err := loadDynamicLibrary(strings.TrimSpace(cfg.LibraryPath))
	if err != nil {
		globalRuntime.loadErr = err
		return err
	}

	if globalRuntime.lib != nil {
		_ = globalRuntime.lib.Close()
	}
	globalRuntime.lib = nextLib
	return nil
}

func currentDynamicLibrary() (*dynamicLibrary, bool) {
	globalRuntime.mu.RLock()
	defer globalRuntime.mu.RUnlock()
	if globalRuntime.lib == nil || !globalRuntime.cfg.Enabled {
		return nil, false
	}
	return globalRuntime.lib, true
}

func currentRuntimeConfig() config.RustFFIConfig {
	globalRuntime.mu.RLock()
	defer globalRuntime.mu.RUnlock()
	return globalRuntime.cfg
}
