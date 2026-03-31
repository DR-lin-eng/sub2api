package kiro

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func NormalizeMachineID(machineID string) string {
	trimmed := strings.TrimSpace(machineID)
	if len(trimmed) == 64 && isHex(trimmed) {
		return trimmed
	}

	withoutDashes := strings.ReplaceAll(trimmed, "-", "")
	if len(withoutDashes) == 32 && isHex(withoutDashes) {
		return withoutDashes + withoutDashes
	}

	return ""
}

func GenerateMachineID(credentialMachineID, globalMachineID, refreshToken string) string {
	if normalized := NormalizeMachineID(credentialMachineID); normalized != "" {
		return normalized
	}
	if normalized := NormalizeMachineID(globalMachineID); normalized != "" {
		return normalized
	}
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("KotlinNativeAPI/" + refreshToken))
	return hex.EncodeToString(sum[:])
}

func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}
