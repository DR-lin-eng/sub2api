//go:build unix && !linux && !windows

package main

func applyWorkerCPUAffinityPlatform(workerIndex int) error {
	return nil
}
