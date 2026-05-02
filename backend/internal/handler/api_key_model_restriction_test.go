package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestAPIKeyAllowsRequestedModel(t *testing.T) {
	apiKey := &service.APIKey{AllowedModels: []string{"gpt-5.4", "gemini-2.5-pro"}}

	if !apiKeyAllowsRequestedModel(apiKey, "gpt-5.4") {
		t.Fatalf("expected exact model to be allowed")
	}
	if !apiKeyAllowsRequestedModel(apiKey, "models/gemini-2.5-pro") {
		t.Fatalf("expected models/ prefixed gemini model to be allowed")
	}
	if apiKeyAllowsRequestedModel(apiKey, "gpt-5.5") {
		t.Fatalf("expected non-allowlisted model to be rejected")
	}
}

func TestFilterOpenAIModelsForAPIKey(t *testing.T) {
	apiKey := &service.APIKey{AllowedModels: []string{"gpt-5.4"}}
	models := []openai.Model{
		{ID: "gpt-5.4"},
		{ID: "gpt-5.5"},
	}

	filtered := filterOpenAIModelsForAPIKey(apiKey, models)
	if len(filtered) != 1 || filtered[0].ID != "gpt-5.4" {
		t.Fatalf("unexpected filtered models: %#v", filtered)
	}
}

func TestFilterGeminiModelsForAPIKey(t *testing.T) {
	apiKey := &service.APIKey{AllowedModels: []string{"gemini-2.5-pro"}}
	models := []gemini.Model{
		{Name: "models/gemini-2.5-pro"},
		{Name: "models/gemini-2.5-flash"},
	}

	filtered := filterGeminiModelsForAPIKey(apiKey, models)
	if len(filtered) != 1 || filtered[0].Name != "models/gemini-2.5-pro" {
		t.Fatalf("unexpected filtered models: %#v", filtered)
	}
}
