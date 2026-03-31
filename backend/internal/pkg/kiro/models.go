package kiro

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
)

var DefaultModels = []claude.Model{
	{
		ID:          "claude-sonnet-4-5-20250929",
		Type:        "model",
		DisplayName: "Claude Sonnet 4.5",
		CreatedAt:   "2025-09-29T00:00:00Z",
	},
	{
		ID:          "claude-sonnet-4-5-20250929-thinking",
		Type:        "model",
		DisplayName: "Claude Sonnet 4.5 (Thinking)",
		CreatedAt:   "2025-09-29T00:00:00Z",
	},
	{
		ID:          "claude-opus-4-5-20251101",
		Type:        "model",
		DisplayName: "Claude Opus 4.5",
		CreatedAt:   "2025-11-01T00:00:00Z",
	},
	{
		ID:          "claude-opus-4-5-20251101-thinking",
		Type:        "model",
		DisplayName: "Claude Opus 4.5 (Thinking)",
		CreatedAt:   "2025-11-01T00:00:00Z",
	},
	{
		ID:          "claude-sonnet-4-6",
		Type:        "model",
		DisplayName: "Claude Sonnet 4.6",
		CreatedAt:   "2026-02-06T00:00:00Z",
	},
	{
		ID:          "claude-sonnet-4-6-thinking",
		Type:        "model",
		DisplayName: "Claude Sonnet 4.6 (Thinking)",
		CreatedAt:   "2026-02-06T00:00:00Z",
	},
	{
		ID:          "claude-opus-4-6",
		Type:        "model",
		DisplayName: "Claude Opus 4.6",
		CreatedAt:   "2026-02-06T00:00:00Z",
	},
	{
		ID:          "claude-opus-4-6-thinking",
		Type:        "model",
		DisplayName: "Claude Opus 4.6 (Thinking)",
		CreatedAt:   "2026-02-06T00:00:00Z",
	},
	{
		ID:          "claude-haiku-4-5-20251001",
		Type:        "model",
		DisplayName: "Claude Haiku 4.5",
		CreatedAt:   "2025-10-01T00:00:00Z",
	},
	{
		ID:          "claude-haiku-4-5-20251001-thinking",
		Type:        "model",
		DisplayName: "Claude Haiku 4.5 (Thinking)",
		CreatedAt:   "2025-10-01T00:00:00Z",
	},
}

func DefaultModelIDs() []string {
	out := make([]string, 0, len(DefaultModels))
	for _, model := range DefaultModels {
		out = append(out, model.ID)
	}
	return out
}

func MapModel(model string) string {
	modelLower := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(modelLower, "sonnet"):
		if strings.Contains(modelLower, "4-6") || strings.Contains(modelLower, "4.6") {
			return "claude-sonnet-4.6"
		}
		return "claude-sonnet-4.5"
	case strings.Contains(modelLower, "opus"):
		if strings.Contains(modelLower, "4-5") || strings.Contains(modelLower, "4.5") {
			return "claude-opus-4.5"
		}
		return "claude-opus-4.6"
	case strings.Contains(modelLower, "haiku"):
		return "claude-haiku-4.5"
	default:
		return ""
	}
}
