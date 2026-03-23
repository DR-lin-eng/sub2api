package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	processModeSingle       = config.ProcessModeSingle
	processModeMasterWorker = config.ProcessModeMasterWorker

	processRoleEnv             = "SUB2API_PROCESS_ROLE"
	processRoleWorker          = "worker"
	processRoleCoordinator     = "coordinator"
	processWorkerIndexEnv      = "SUB2API_PROCESS_INDEX"
	processWorkerGenerationEnv = "SUB2API_PROCESS_GENERATION"
	processListenerFDEnv       = "SUB2API_LISTENER_FD"
	processReadyFDEnv          = "SUB2API_READY_FD"
	defaultGracefulShutdown    = 5 * time.Second
	defaultWorkerReadyTimeout  = 20 * time.Second
	defaultReloadTimeout       = 60 * time.Second
	defaultRespawnBackoff      = 2 * time.Second
)

var (
	applicationBuilder            = initializeApplication
	coordinatorApplicationBuilder = initializeCoordinatorApplication
)

func runServerProcessModel(cfg *config.Config, buildInfo handler.BuildInfo) error {
	if isWorkerProcess() {
		return runWorkerProcess(cfg, buildInfo)
	}
	if isCoordinatorProcess() {
		return runCoordinatorProcess(cfg, buildInfo)
	}
	return runMasterOrSingleProcess(cfg, buildInfo)
}

func runSingleProcess(cfg *config.Config, buildInfo handler.BuildInfo) error {
	app, err := applicationBuilder(buildInfo)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	defer app.Cleanup()

	listener, err := net.Listen("tcp", app.Server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", app.Server.Addr, err)
	}
	log.Printf("Server started on %s", app.Server.Addr)
	return serveApplicationWithGracefulShutdown(app.Server, listener, gracefulShutdownTimeout(cfg))
}

func serveApplicationWithGracefulShutdown(server *http.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	if server == nil {
		return errors.New("http server is nil")
	}
	if listener == nil {
		return errors.New("listener is nil")
	}
	if shutdownTimeout <= 0 {
		shutdownTimeout = defaultGracefulShutdown
	}

	serveErrCh := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErrCh <- err
			return
		}
		serveErrCh <- nil
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-serveErrCh:
		return err
	case <-quit:
		log.Println("Shutting down server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		_ = listener.Close()
		if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			log.Printf("Force close server failed: %v", closeErr)
		}
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	if err := <-serveErrCh; err != nil {
		return err
	}

	log.Println("Server exited")
	return nil
}

func isMasterWorkerModeEnabled(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(cfg.Process.Mode), processModeMasterWorker)
}

func resolvedWorkerCount(cfg *config.Config) int {
	if cfg == nil {
		return 1
	}
	if cfg.Process.WorkerCount > 0 {
		return cfg.Process.WorkerCount
	}
	if cpuCount := runtime.NumCPU(); cpuCount > 0 {
		return cpuCount
	}
	return 1
}

func gracefulShutdownTimeout(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.Process.GracefulShutdownTimeoutSeconds <= 0 {
		return defaultGracefulShutdown
	}
	return time.Duration(cfg.Process.GracefulShutdownTimeoutSeconds) * time.Second
}

func workerReadyTimeout(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.Process.WorkerReadyTimeoutSeconds <= 0 {
		return defaultWorkerReadyTimeout
	}
	return time.Duration(cfg.Process.WorkerReadyTimeoutSeconds) * time.Second
}

func reloadTimeout(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.Process.ReloadTimeoutSeconds <= 0 {
		return defaultReloadTimeout
	}
	return time.Duration(cfg.Process.ReloadTimeoutSeconds) * time.Second
}

func respawnBackoff(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.Process.RespawnBackoffMS <= 0 {
		return defaultRespawnBackoff
	}
	return time.Duration(cfg.Process.RespawnBackoffMS) * time.Millisecond
}

func isWorkerProcess() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(processRoleEnv)), processRoleWorker)
}

func isCoordinatorProcess() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(processRoleEnv)), processRoleCoordinator)
}

func currentWorkerIndex() int {
	return parseEnvInt(processWorkerIndexEnv)
}

func currentWorkerGeneration() int {
	return parseEnvInt(processWorkerGenerationEnv)
}

func parseEnvInt(key string) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func reopenLoggerFromConfig() error {
	cfg, err := config.LoadForBootstrap()
	if err != nil {
		return err
	}
	return logger.Init(logger.OptionsFromConfig(cfg.Log))
}

func inheritedFDFromEnv(envKey string) uintptr {
	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return uintptr(value)
}

func runCoordinatorProcess(cfg *config.Config, buildInfo handler.BuildInfo) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	app, err := coordinatorApplicationBuilder(buildInfo)
	if err != nil {
		return fmt.Errorf("initialize coordinator application: %w", err)
	}
	defer app.Cleanup()

	if err := signalChildReady(); err != nil {
		return err
	}

	log.Printf("Coordinator ready: pid=%d", os.Getpid())

	sigCh := make(chan os.Signal, 8)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1)
	defer signal.Stop(sigCh)

	for {
		sig := <-sigCh
		switch sig {
		case syscall.SIGUSR1:
			if cfg.Process.LogReopenSignalEnabled {
				if err := reopenLoggerFromConfig(); err != nil {
					log.Printf("Coordinator log reopen failed: %v", err)
				}
			}
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			log.Printf("Coordinator exiting on %s", sig.String())
			return nil
		}
	}
}
