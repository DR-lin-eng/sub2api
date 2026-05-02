package handler

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func normalizeAPIKeyAllowedModelID(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimPrefix(model, "models/")
	return model
}

func apiKeyAllowsRequestedModel(apiKey *service.APIKey, model string) bool {
	if apiKey == nil || len(apiKey.AllowedModels) == 0 {
		return true
	}
	requested := normalizeAPIKeyAllowedModelID(model)
	if requested == "" {
		return true
	}
	for _, allowed := range apiKey.AllowedModels {
		if normalizeAPIKeyAllowedModelID(allowed) == requested {
			return true
		}
	}
	return false
}

func apiKeyModelNotAllowedMessage(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "Requested model is not allowed for this API key"
	}
	return fmt.Sprintf("Model %q is not allowed for this API key", model)
}

func filterClaudeModelsForAPIKey(apiKey *service.APIKey, models []claude.Model) []claude.Model {
	if apiKey == nil || len(apiKey.AllowedModels) == 0 {
		return models
	}
	out := make([]claude.Model, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.ID) {
			out = append(out, model)
		}
	}
	return out
}

func filterOpenAIModelsForAPIKey(apiKey *service.APIKey, models []openai.Model) []openai.Model {
	if apiKey == nil || len(apiKey.AllowedModels) == 0 {
		return models
	}
	out := make([]openai.Model, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.ID) {
			out = append(out, model)
		}
	}
	return out
}

func filterGeminiModelsForAPIKey(apiKey *service.APIKey, models []gemini.Model) []gemini.Model {
	if apiKey == nil || len(apiKey.AllowedModels) == 0 {
		return models
	}
	out := make([]gemini.Model, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.Name) {
			out = append(out, model)
		}
	}
	return out
}

func filterAntigravityClaudeModelsForAPIKey(apiKey *service.APIKey, models []antigravity.ClaudeModel) []antigravity.ClaudeModel {
	if apiKey == nil || len(apiKey.AllowedModels) == 0 {
		return models
	}
	out := make([]antigravity.ClaudeModel, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.ID) {
			out = append(out, model)
		}
	}
	return out
}

func filterAntigravityGeminiModelsForAPIKey(apiKey *service.APIKey, models []antigravity.GeminiModel) []antigravity.GeminiModel {
	if apiKey == nil || len(apiKey.AllowedModels) == 0 {
		return models
	}
	out := make([]antigravity.GeminiModel, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.Name) {
			out = append(out, model)
		}
	}
	return out
}
