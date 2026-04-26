//go:build !unix || windows

package main

import (
	"log"
	"os"
	"runtime"
	"syscall"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
)

func runMasterOrSingleProcess(cfg *config.Config, buildInfo handler.BuildInfo) error {
	if isMasterWorkerModeEnabled(cfg) {
		log.Printf("process.mode=%q is not supported on %s, falling back to single-process mode", cfg.Process.Mode, runtimeOS())
	}
	return runSingleProcess(cfg, buildInfo)
}

func runWorkerProcess(cfg *config.Config, buildInfo handler.BuildInfo) error {
	return runSingleProcess(cfg, buildInfo)
}

func signalChildReady() error {
	return nil
}

func coordinatorProcessSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}
}

func isLogReopenSignal(sig os.Signal) bool {
	return false
}

func applyWorkerCPUAffinityPlatform(workerIndex int) error {
	return nil
}

func runtimeOS() string {
	return runtime.GOOS
}
