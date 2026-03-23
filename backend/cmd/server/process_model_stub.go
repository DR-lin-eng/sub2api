//go:build !unix || windows

package main

import (
	"log"
	"runtime"

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

func runtimeOS() string {
	return runtime.GOOS
}
