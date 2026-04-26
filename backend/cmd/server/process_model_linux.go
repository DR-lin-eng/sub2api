//go:build linux

package main

import (
	"runtime"

	"golang.org/x/sys/unix"
)

func applyWorkerCPUAffinityPlatform(workerIndex int) error {
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
