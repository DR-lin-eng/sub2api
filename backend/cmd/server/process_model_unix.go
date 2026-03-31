//go:build unix && !windows

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	rustffi "github.com/Wei-Shaw/sub2api/internal/rustbridge/ffi"
	"golang.org/x/sys/unix"
)

type managedWorker struct {
	cmd        *exec.Cmd
	role       string
	index      int
	generation int
	stopping   bool
}

type workerExitEvent struct {
	pid int
	err error
}

func runMasterOrSingleProcess(cfg *config.Config, buildInfo handler.BuildInfo) error {
	if !isMasterWorkerModeEnabled(cfg) {
		return runSingleProcess(cfg, buildInfo)
	}
	return runMasterProcess(cfg, buildInfo)
}

func runMasterProcess(cfg *config.Config, buildInfo handler.BuildInfo) error {
	listener, err := net.Listen("tcp", cfg.Server.Address())
	if err != nil {
		return fmt.Errorf("master listen on %s: %w", cfg.Server.Address(), err)
	}
	defer func() { _ = listener.Close() }()

	supervisor := &masterSupervisor{
		cfg:           cfg,
		buildInfo:     buildInfo,
		listener:      listener,
		exitCh:        make(chan workerExitEvent, 32),
		currentByPID:  make(map[int]*managedWorker),
		currentBySlot: make(map[int]*managedWorker),
	}

	if err := supervisor.startGeneration(cfg); err != nil {
		return err
	}

	log.Printf("Master started on %s with %d worker(s)", cfg.Server.Address(), len(supervisor.currentBySlot))

	sigCh := make(chan os.Signal, 8)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGUSR1)
	defer signal.Stop(sigCh)

	for {
		select {
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				if supervisor.cfg == nil || !supervisor.cfg.Process.ConfigReloadSignalEnabled {
					log.Printf("Ignoring SIGHUP because process.config_reload_signal_enabled=false")
					continue
				}
				if err := supervisor.reload(); err != nil {
					log.Printf("Master reload failed: %v", err)
				}
			case syscall.SIGUSR1:
				if supervisor.cfg == nil || !supervisor.cfg.Process.LogReopenSignalEnabled {
					log.Printf("Ignoring SIGUSR1 because process.log_reopen_signal_enabled=false")
					continue
				}
				if err := reopenLoggerFromConfig(); err != nil {
					log.Printf("Master log reopen failed: %v", err)
				}
				supervisor.broadcast(syscall.SIGUSR1)
			case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
				return supervisor.shutdown(sig)
			}
		case exit := <-supervisor.exitCh:
			supervisor.handleWorkerExit(exit)
		}
	}
}

func runWorkerProcess(cfg *config.Config, buildInfo handler.BuildInfo) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	workerIndex := currentWorkerIndex()
	workerGeneration := currentWorkerGeneration()
	if cfg.Process.EnableCPUAffinity {
		if err := applyWorkerCPUAffinity(workerIndex); err != nil {
			log.Printf("Worker %d failed to apply CPU affinity: %v", workerIndex, err)
		}
	}

	app, err := applicationBuilder(buildInfo)
	if err != nil {
		return fmt.Errorf("initialize worker application: %w", err)
	}
	defer app.Cleanup()

	stopRustUpstream, err := startRustSidecarUpstreamServer(cfg, app.Server)
	if err != nil {
		return fmt.Errorf("start rust sidecar upstream server: %w", err)
	}
	defer stopRustUpstream()

	stopRustSidecar, err := startRustSidecarProcess(cfg)
	if err != nil {
		return fmt.Errorf("start rust sidecar process: %w", err)
	}
	defer stopRustSidecar()

	if err := rustffi.Configure(cfg.Rust.FFI); err != nil {
		return fmt.Errorf("configure rust ffi: %w", err)
	}
	runtime := resolveApplicationRuntime(cfg, app)
	if runtime == nil {
		return errors.New("application ingress runtime is nil")
	}

	listener, err := inheritedListener()
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- runtime.Serve(listener)
	}()

	if err := signalChildReady(); err != nil {
		return err
	}

	log.Printf(
		"Worker ready: pid=%d generation=%d index=%d addr=%s role=%s",
		os.Getpid(),
		workerGeneration,
		workerIndex,
		runtime.Addr(),
		processRoleWorker,
	)

	sigCh := make(chan os.Signal, 8)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1)
	defer signal.Stop(sigCh)

	for {
		select {
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGUSR1:
				if cfg == nil || !cfg.Process.LogReopenSignalEnabled {
					continue
				}
				if err := reopenLoggerFromConfig(); err != nil {
					log.Printf("Worker %d log reopen failed: %v", workerIndex, err)
				}
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				ctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout(cfg))
				err := runtime.Shutdown(ctx)
				cancel()
				if err != nil {
					_ = listener.Close()
					if closeErr := runtime.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
						log.Printf("Worker %d force close failed: %v", workerIndex, closeErr)
					}
					return fmt.Errorf("worker graceful shutdown failed: %w", err)
				}
			}
		case err := <-serveErrCh:
			return err
		}
	}
}

type masterSupervisor struct {
	cfg           *config.Config
	buildInfo     handler.BuildInfo
	listener      net.Listener
	generation    int
	shuttingDown  bool
	exitCh        chan workerExitEvent
	coordinator   *managedWorker
	currentByPID  map[int]*managedWorker
	currentBySlot map[int]*managedWorker
}

func (s *masterSupervisor) startGeneration(cfg *config.Config) error {
	nextGeneration := s.generation + 1
	newCoordinator, err := s.spawnCoordinator(cfg, nextGeneration)
	if err != nil {
		return err
	}
	newWorkers := make(map[int]*managedWorker, resolvedWorkerCount(cfg))

	for i := 0; i < resolvedWorkerCount(cfg); i++ {
		worker, err := s.spawnWorker(cfg, nextGeneration, i)
		if err != nil {
			s.stopWorker(newCoordinator, syscall.SIGTERM)
			for _, started := range newWorkers {
				s.stopWorker(started, syscall.SIGTERM)
			}
			deadline := time.Now().Add(reloadTimeout(cfg))
			s.waitWorkerStopped(newCoordinator, deadline)
			for _, started := range newWorkers {
				s.waitWorkerStopped(started, deadline)
			}
			return err
		}
		newWorkers[i] = worker
	}

	oldCoordinator := s.coordinator
	oldWorkers := s.currentBySlot
	s.coordinator = newCoordinator
	s.currentBySlot = newWorkers
	s.generation = nextGeneration
	s.currentByPID[newCoordinator.cmd.Process.Pid] = newCoordinator
	for _, worker := range newWorkers {
		s.currentByPID[worker.cmd.Process.Pid] = worker
	}

	deadline := time.Now().Add(reloadTimeout(cfg))
	s.stopWorker(oldCoordinator, syscall.SIGTERM)
	for _, worker := range oldWorkers {
		s.stopWorker(worker, syscall.SIGTERM)
	}
	s.waitWorkerStopped(oldCoordinator, deadline)
	for _, worker := range oldWorkers {
		s.waitWorkerStopped(worker, deadline)
	}

	return nil
}

func (s *masterSupervisor) reload() error {
	nextCfg, err := config.LoadForBootstrap()
	if err != nil {
		return fmt.Errorf("reload config: %w", err)
	}
	if !isMasterWorkerModeEnabled(nextCfg) {
		log.Printf("Ignoring reload that disables process.mode=%q while master is active", strings.TrimSpace(nextCfg.Process.Mode))
		return nil
	}
	if nextCfg.Server.Address() != s.cfg.Server.Address() {
		log.Printf(
			"Reload detected server address change %q -> %q; inherited listener keeps the original address until full restart",
			s.cfg.Server.Address(),
			nextCfg.Server.Address(),
		)
	}

	if err := s.startGeneration(nextCfg); err != nil {
		return err
	}

	s.cfg = nextCfg
	if err := logger.Init(logger.OptionsFromConfig(nextCfg.Log)); err != nil {
		log.Printf("Master logger reload failed: %v", err)
	}
	log.Printf("Master reload succeeded: generation=%d workers=%d", s.generation, len(s.currentBySlot))
	return nil
}

func (s *masterSupervisor) shutdown(sig os.Signal) error {
	s.shuttingDown = true
	log.Printf("Master received %s, shutting down coordinator and %d worker(s)", sig.String(), len(s.currentBySlot))

	deadline := time.Now().Add(gracefulShutdownTimeout(s.cfg))
	s.stopWorker(s.coordinator, syscall.SIGTERM)
	for _, worker := range s.currentBySlot {
		s.stopWorker(worker, syscall.SIGTERM)
	}
	s.waitWorkerStopped(s.coordinator, deadline)
	for _, worker := range s.currentBySlot {
		s.waitWorkerStopped(worker, deadline)
	}
	return nil
}

func (s *masterSupervisor) broadcast(sig syscall.Signal) {
	if s.coordinator != nil && s.coordinator.cmd != nil && s.coordinator.cmd.Process != nil {
		if err := s.coordinator.cmd.Process.Signal(sig); err != nil && !errors.Is(err, os.ErrProcessDone) {
			log.Printf("Failed to signal coordinator pid=%d with %s: %v", s.coordinator.cmd.Process.Pid, sig.String(), err)
		}
	}
	for _, worker := range s.currentBySlot {
		if worker == nil || worker.cmd == nil || worker.cmd.Process == nil {
			continue
		}
		if err := worker.cmd.Process.Signal(sig); err != nil && !errors.Is(err, os.ErrProcessDone) {
			log.Printf("Failed to signal worker pid=%d with %s: %v", worker.cmd.Process.Pid, sig.String(), err)
		}
	}
}

func (s *masterSupervisor) stopWorker(worker *managedWorker, sig syscall.Signal) {
	if worker == nil || worker.cmd == nil || worker.cmd.Process == nil || worker.stopping {
		return
	}
	worker.stopping = true
	if err := worker.cmd.Process.Signal(sig); err != nil && !errors.Is(err, os.ErrProcessDone) {
		log.Printf("Failed to stop worker pid=%d: %v", worker.cmd.Process.Pid, err)
	}
}

func (s *masterSupervisor) waitWorkerStopped(worker *managedWorker, deadline time.Time) {
	if worker == nil || worker.cmd == nil || worker.cmd.Process == nil {
		return
	}
	for time.Now().Before(deadline) {
		if _, ok := s.currentByPID[worker.cmd.Process.Pid]; !ok {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := worker.cmd.Process.Signal(syscall.SIGKILL); err != nil && !errors.Is(err, os.ErrProcessDone) {
		log.Printf("Failed to SIGKILL worker pid=%d: %v", worker.cmd.Process.Pid, err)
	}
}

func (s *masterSupervisor) handleWorkerExit(exit workerExitEvent) {
	worker, ok := s.currentByPID[exit.pid]
	if !ok {
		return
	}
	delete(s.currentByPID, exit.pid)
	if worker.role == processRoleCoordinator {
		if s.coordinator != nil && s.coordinator.cmd != nil && s.coordinator.cmd.Process != nil && s.coordinator.cmd.Process.Pid == exit.pid {
			s.coordinator = nil
		}
		if exit.err != nil {
			log.Printf("Coordinator exited: pid=%d generation=%d stopping=%t err=%v", exit.pid, worker.generation, worker.stopping, exit.err)
		} else {
			log.Printf("Coordinator exited: pid=%d generation=%d stopping=%t", exit.pid, worker.generation, worker.stopping)
		}
		if s.shuttingDown || worker.stopping || worker.generation != s.generation {
			return
		}
		time.Sleep(respawnBackoff(s.cfg))
		replacement, err := s.spawnCoordinator(s.cfg, s.generation)
		if err != nil {
			log.Printf("Failed to respawn coordinator generation=%d: %v", worker.generation, err)
			return
		}
		s.coordinator = replacement
		s.currentByPID[replacement.cmd.Process.Pid] = replacement
		log.Printf("Respawned coordinator: old_pid=%d new_pid=%d generation=%d", exit.pid, replacement.cmd.Process.Pid, replacement.generation)
		return
	}
	currentWorker, currentSlot := s.currentBySlot[worker.index]
	if currentSlot && currentWorker != nil && currentWorker.cmd != nil && currentWorker.cmd.Process != nil && currentWorker.cmd.Process.Pid == exit.pid {
		delete(s.currentBySlot, worker.index)
	}

	if exit.err != nil && !errors.Is(exit.err, http.ErrServerClosed) {
		log.Printf("Worker exited: pid=%d generation=%d index=%d stopping=%t err=%v", exit.pid, worker.generation, worker.index, worker.stopping, exit.err)
	} else {
		log.Printf("Worker exited: pid=%d generation=%d index=%d stopping=%t", exit.pid, worker.generation, worker.index, worker.stopping)
	}

	if s.shuttingDown || worker.stopping || worker.generation != s.generation {
		return
	}

	time.Sleep(respawnBackoff(s.cfg))
	replacement, err := s.spawnWorker(s.cfg, s.generation, worker.index)
	if err != nil {
		log.Printf("Failed to respawn worker index=%d generation=%d: %v", worker.index, worker.generation, err)
		return
	}
	s.currentBySlot[worker.index] = replacement
	s.currentByPID[replacement.cmd.Process.Pid] = replacement
	log.Printf("Respawned worker: old_pid=%d new_pid=%d generation=%d index=%d", exit.pid, replacement.cmd.Process.Pid, replacement.generation, replacement.index)
}

func (s *masterSupervisor) spawnWorker(cfg *config.Config, generation, index int) (*managedWorker, error) {
	listenerFile, err := duplicateListenerFile(s.listener)
	if err != nil {
		return nil, err
	}
	defer func() { _ = listenerFile.Close() }()

	readyR, readyW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create worker ready pipe: %w", err)
	}
	defer func() { _ = readyR.Close() }()

	cmd := exec.Command(os.Args[0])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{listenerFile, readyW}
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("%s=%s", processRoleEnv, processRoleWorker),
		fmt.Sprintf("%s=%d", processWorkerIndexEnv, index),
		fmt.Sprintf("%s=%d", processWorkerGenerationEnv, generation),
		fmt.Sprintf("%s=%d", processListenerFDEnv, 3),
		fmt.Sprintf("%s=%d", processReadyFDEnv, 4),
	)

	if err := cmd.Start(); err != nil {
		_ = readyW.Close()
		return nil, fmt.Errorf("start worker index=%d generation=%d: %w", index, generation, err)
	}
	_ = readyW.Close()

	readyCh := make(chan error, 1)
	go func() {
		defer func() { _ = readyR.Close() }()
		line, readErr := bufio.NewReader(readyR).ReadString('\n')
		if readErr != nil {
			readyCh <- fmt.Errorf("worker readiness pipe: %w", readErr)
			return
		}
		if strings.TrimSpace(line) != "ready" {
			readyCh <- fmt.Errorf("unexpected worker readiness payload: %q", strings.TrimSpace(line))
			return
		}
		readyCh <- nil
	}()

	worker := &managedWorker{
		cmd:        cmd,
		role:       processRoleWorker,
		index:      index,
		generation: generation,
	}

	go func(pid int) {
		s.exitCh <- workerExitEvent{pid: pid, err: cmd.Wait()}
	}(cmd.Process.Pid)

	select {
	case err := <-readyCh:
		if err != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			return nil, err
		}
	case <-time.After(workerReadyTimeout(cfg)):
		_ = cmd.Process.Signal(syscall.SIGTERM)
		return nil, fmt.Errorf("worker ready timeout: generation=%d index=%d", generation, index)
	}

	return worker, nil
}

func (s *masterSupervisor) spawnCoordinator(cfg *config.Config, generation int) (*managedWorker, error) {
	readyR, readyW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create coordinator ready pipe: %w", err)
	}
	defer func() { _ = readyR.Close() }()

	cmd := exec.Command(os.Args[0])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{readyW}
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("%s=%s", processRoleEnv, processRoleCoordinator),
		fmt.Sprintf("%s=%d", processWorkerGenerationEnv, generation),
		fmt.Sprintf("%s=%d", processReadyFDEnv, 3),
	)

	if err := cmd.Start(); err != nil {
		_ = readyW.Close()
		return nil, fmt.Errorf("start coordinator generation=%d: %w", generation, err)
	}
	_ = readyW.Close()

	readyCh := make(chan error, 1)
	go func() {
		defer func() { _ = readyR.Close() }()
		line, readErr := bufio.NewReader(readyR).ReadString('\n')
		if readErr != nil {
			readyCh <- fmt.Errorf("coordinator readiness pipe: %w", readErr)
			return
		}
		if strings.TrimSpace(line) != "ready" {
			readyCh <- fmt.Errorf("unexpected coordinator readiness payload: %q", strings.TrimSpace(line))
			return
		}
		readyCh <- nil
	}()

	coordinator := &managedWorker{
		cmd:        cmd,
		role:       processRoleCoordinator,
		index:      -1,
		generation: generation,
	}

	go func(pid int) {
		s.exitCh <- workerExitEvent{pid: pid, err: cmd.Wait()}
	}(cmd.Process.Pid)

	select {
	case err := <-readyCh:
		if err != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			return nil, err
		}
	case <-time.After(workerReadyTimeout(cfg)):
		_ = cmd.Process.Signal(syscall.SIGTERM)
		return nil, fmt.Errorf("coordinator ready timeout: generation=%d", generation)
	}

	return coordinator, nil
}

func duplicateListenerFile(listener net.Listener) (*os.File, error) {
	type fileListener interface {
		File() (*os.File, error)
	}
	fl, ok := listener.(fileListener)
	if !ok {
		return nil, fmt.Errorf("listener type %T does not expose File()", listener)
	}
	file, err := fl.File()
	if err != nil {
		return nil, fmt.Errorf("duplicate listener file: %w", err)
	}
	return file, nil
}

func inheritedListener() (net.Listener, error) {
	fd := inheritedFDFromEnv(processListenerFDEnv)
	if fd == 0 {
		return nil, errors.New("worker inherited listener fd is missing")
	}
	file := os.NewFile(fd, "sub2api-listener")
	if file == nil {
		return nil, errors.New("worker inherited listener file is nil")
	}
	defer func() { _ = file.Close() }()

	listener, err := net.FileListener(file)
	if err != nil {
		return nil, fmt.Errorf("restore inherited listener: %w", err)
	}
	return listener, nil
}

func signalChildReady() error {
	fd := inheritedFDFromEnv(processReadyFDEnv)
	if fd == 0 {
		return errors.New("child ready fd is missing")
	}
	file := os.NewFile(fd, "sub2api-ready")
	if file == nil {
		return errors.New("child ready file is nil")
	}
	defer func() { _ = file.Close() }()

	if _, err := file.WriteString("ready\n"); err != nil {
		return fmt.Errorf("signal child ready: %w", err)
	}
	return nil
}

func applyWorkerCPUAffinity(workerIndex int) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	cpus := runtime.NumCPU()
	if cpus <= 0 {
		return nil
	}
	targetCPU := workerIndex % cpus
	var set unix.CPUSet
	set.Zero()
	set.Set(targetCPU)
	return unix.SchedSetaffinity(0, &set)
}
