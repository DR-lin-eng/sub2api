package service

func adaptiveSchedulerSelectionTopK(baseTopK int, candidateCount int, loadSkew float64) int {
	if candidateCount <= 0 {
		return 1
	}
	if baseTopK <= 0 {
		baseTopK = 1
	}
	if baseTopK > candidateCount {
		baseTopK = candidateCount
	}

	adaptive := baseTopK
	switch {
	case candidateCount >= 8192:
		adaptive += candidateCount / 64
	case candidateCount >= 2048:
		adaptive += candidateCount / 48
	case candidateCount >= 512:
		adaptive += candidateCount / 32
	case candidateCount >= 128:
		adaptive += candidateCount / 24
	case candidateCount >= 32:
		adaptive += candidateCount / 16
	case candidateCount >= 16:
		adaptive += candidateCount / 8
	}

	switch {
	case loadSkew <= 5:
		adaptive += 2
	case loadSkew <= 10:
		adaptive++
	}

	if candidateCount >= 128 && adaptive < 32 {
		adaptive = 32
	}

	windowCap := schedulerSelectionWindowCap(candidateCount)
	if adaptive > windowCap {
		adaptive = windowCap
	}
	if adaptive > candidateCount {
		adaptive = candidateCount
	}
	return adaptive
}

func schedulerSelectionWindowCap(candidateCount int) int {
	switch {
	case candidateCount >= 8192:
		return 256
	case candidateCount >= 2048:
		return 192
	case candidateCount >= 512:
		return 128
	case candidateCount >= 128:
		return 64
	case candidateCount >= 32:
		return 32
	default:
		return 16
	}
}
