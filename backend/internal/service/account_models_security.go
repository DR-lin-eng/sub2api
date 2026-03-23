package service

import (
	"sort"
	"strings"
	"time"
)

const (
	AccountCredentialMaskedValue              = "[configured]"
	AccountExtraFetchedModelsKey              = "fetched_models"
	AccountExtraModelsFetchedAtKey            = "models_fetched_at"
	AccountExtraModelsRefreshErrorKey         = "models_refresh_error"
	AccountExtraModelsRefreshIntervalSecKey   = "models_refresh_interval_seconds"
	AccountExtraModelsSourceKey               = "models_source"
)

var sensitiveCredentialKeys = map[string]struct{}{
	"access_token":          {},
	"api_key":               {},
	"authorization_code":    {},
	"aws_secret_access_key": {},
	"aws_session_token":     {},
	"client_secret":         {},
	"code":                  {},
	"code_verifier":         {},
	"id_token":              {},
	"refresh_token":         {},
	"session_key":           {},
	"session_token":         {},
	"watermark_parse_token": {},
}

func IsSensitiveCredentialKey(key string) bool {
	_, ok := sensitiveCredentialKeys[strings.ToLower(strings.TrimSpace(key))]
	return ok
}

func IsMaskedCredentialValue(value any) bool {
	text, ok := value.(string)
	if !ok {
		return false
	}
	return strings.TrimSpace(text) == AccountCredentialMaskedValue
}

func MaskSensitiveCredentials(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}

	out := make(map[string]any, len(in))
	for key, value := range in {
		if nested, ok := value.(map[string]any); ok {
			out[key] = MaskSensitiveCredentials(nested)
			continue
		}
		if IsSensitiveCredentialKey(key) && value != nil {
			out[key] = AccountCredentialMaskedValue
			continue
		}
		out[key] = value
	}
	return out
}

func MergeCredentialUpdatesPreservingSecrets(existing, incoming map[string]any) map[string]any {
	if incoming == nil {
		return nil
	}

	out := cloneJSONMap(incoming)
	for key, value := range out {
		if !IsSensitiveCredentialKey(key) || !IsMaskedCredentialValue(value) {
			continue
		}
		if existingValue, ok := existing[key]; ok {
			out[key] = existingValue
			continue
		}
		delete(out, key)
	}
	return out
}

func StripMaskedSensitiveCredentialUpdates(incoming map[string]any) map[string]any {
	if len(incoming) == 0 {
		return nil
	}

	out := make(map[string]any, len(incoming))
	for key, value := range incoming {
		if IsSensitiveCredentialKey(key) && IsMaskedCredentialValue(value) {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeAccountExtraModelIDs(raw any) []string {
	var values []string
	switch typed := raw.(type) {
	case []string:
		values = typed
	case []any:
		values = make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
	default:
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, item := range values {
		id := strings.TrimSpace(item)
		if id == "" {
			continue
		}
		if strings.HasPrefix(id, "models/") {
			id = strings.TrimPrefix(id, "models/")
		}
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func NormalizeFetchedModelIDs(ids []string) []string {
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, id)
	}
	return normalizeAccountExtraModelIDs(items)
}

func (a *Account) GetFetchedModelIDs() []string {
	if a == nil || a.Extra == nil {
		return nil
	}
	return normalizeAccountExtraModelIDs(a.Extra[AccountExtraFetchedModelsKey])
}

func (a *Account) GetModelsFetchedAt() *time.Time {
	if a == nil || a.Extra == nil {
		return nil
	}
	raw, _ := a.Extra[AccountExtraModelsFetchedAtKey].(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return &parsed
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed
	}
	return nil
}

func (a *Account) GetModelsRefreshError() string {
	if a == nil || a.Extra == nil {
		return ""
	}
	text, _ := a.Extra[AccountExtraModelsRefreshErrorKey].(string)
	return strings.TrimSpace(text)
}

func (a *Account) GetModelsRefreshIntervalSeconds() int {
	if a == nil || a.Extra == nil {
		return 0
	}
	return a.getExtraInt(AccountExtraModelsRefreshIntervalSecKey)
}

func (a *Account) ShouldRefreshFetchedModels(now time.Time) bool {
	if a == nil {
		return false
	}
	interval := a.GetModelsRefreshIntervalSeconds()
	if interval <= 0 {
		return false
	}
	lastFetchedAt := a.GetModelsFetchedAt()
	if lastFetchedAt == nil {
		return true
	}
	return !now.Before(lastFetchedAt.Add(time.Duration(interval) * time.Second))
}

func BuildFetchedModelsExtraUpdates(modelIDs []string, fetchedAt time.Time, source string) map[string]any {
	return map[string]any{
		AccountExtraFetchedModelsKey:      NormalizeFetchedModelIDs(modelIDs),
		AccountExtraModelsFetchedAtKey:    fetchedAt.UTC().Format(time.RFC3339Nano),
		AccountExtraModelsRefreshErrorKey: "",
		AccountExtraModelsSourceKey:       strings.TrimSpace(source),
	}
}

func (a *Account) OpenAIPlanType() string {
	if a == nil || !a.IsOpenAI() {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(a.GetCredential("plan_type")))
}

func cloneJSONMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
