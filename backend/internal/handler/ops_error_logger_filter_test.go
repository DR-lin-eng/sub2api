package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type opsErrorLoggerSettingRepoStub struct {
	values map[string]string
}

func (s *opsErrorLoggerSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	value, err := s.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (s *opsErrorLoggerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s.values == nil {
		s.values = map[string]string{}
	}
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}
func (s *opsErrorLoggerSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return nil, nil
}
func (s *opsErrorLoggerSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}
func (s *opsErrorLoggerSettingRepoStub) Set(_ context.Context, key string, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}
func (s *opsErrorLoggerSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}
func (s *opsErrorLoggerSettingRepoStub) Delete(context.Context, string) error { return nil }

func TestShouldSkipOpsErrorLog_IgnoresInvalidAPIKeyMessage(t *testing.T) {
	settingRepo := &opsErrorLoggerSettingRepoStub{}
	opsService := service.NewOpsService(nil, settingRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cfg, err := opsService.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() error = %v", err)
	}
	cfg.IgnoreInvalidApiKeyErrors = true
	if _, err := opsService.UpdateOpsAdvancedSettings(context.Background(), cfg); err != nil {
		t.Fatalf("UpdateOpsAdvancedSettings() error = %v", err)
	}

	if !shouldSkipOpsErrorLog(context.Background(), opsService, "Invalid API key", "", "/v1/messages") {
		t.Fatalf("expected Invalid API key message to be skipped")
	}
	if !shouldSkipOpsErrorLog(context.Background(), opsService, "UNAUTHENTICATED", "{\"message\":\"API key is required\"}", "/v1beta/models") {
		t.Fatalf("expected API key required body to be skipped")
	}
}
