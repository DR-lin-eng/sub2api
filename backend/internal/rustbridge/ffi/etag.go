package ffi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// BuildETagFromAny is the stable seam for future Rust hash acceleration.
// The default implementation intentionally stays pure Go until an actual cdylib
// is wired in behind the same interface.
func BuildETagFromAny(payload any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return BuildETagFromBytes(raw)
}

func BuildETagFromBytes(raw []byte) string {
	if lib, ok := currentDynamicLibrary(); ok {
		if digest, ok := lib.CallSHA256Hex(raw); ok && digest != "" {
			recordMetric(ffiMetricHash, true)
			return "\"" + digest + "\""
		}
	}
	recordMetric(ffiMetricHash, false)
	if len(raw) == 0 {
		return "\"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\""
	}
	sum := sha256.Sum256(raw)
	return "\"" + hex.EncodeToString(sum[:]) + "\""
}
