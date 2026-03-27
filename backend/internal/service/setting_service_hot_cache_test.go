//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type hotSettingRepoStub struct {
	values map[string]string
	calls  map[string]int
}

func (s *hotSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *hotSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s.calls == nil {
		s.calls = map[string]int{}
	}
	s.calls[key]++
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *hotSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *hotSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *hotSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for k, v := range settings {
		s.values[k] = v
	}
	return nil
}

func (s *hotSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *hotSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestSettingService_HotSettingCacheHitsAndInvalidates(t *testing.T) {
	repo := &hotSettingRepoStub{
		values: map[string]string{
			SettingKeyFrontendURL:                  "https://a.example.com",
			SettingKeyAutoDelete401Accounts:        "true",
			SettingKeyOpsMonitoringEnabled:         "true",
			SettingKeyOpsRealtimeMonitoringEnabled: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.Equal(t, "https://a.example.com", svc.GetFrontendURL(context.Background()))
	require.Equal(t, "https://a.example.com", svc.GetFrontendURL(context.Background()))
	require.Equal(t, 1, repo.calls[SettingKeyFrontendURL])

	require.True(t, svc.IsAutoDelete401AccountsEnabled(context.Background()))
	require.True(t, svc.IsAutoDelete401AccountsEnabled(context.Background()))
	require.Equal(t, 1, repo.calls[SettingKeyAutoDelete401Accounts])

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		FrontendURL:                  "https://b.example.com",
		OpsMonitoringEnabled:         true,
		OpsRealtimeMonitoringEnabled: true,
	})
	require.NoError(t, err)

	require.Equal(t, "https://b.example.com", svc.GetFrontendURL(context.Background()))
	require.Equal(t, 2, repo.calls[SettingKeyFrontendURL])
}

func TestSettingService_GetStreamTimeoutSettingsUsesHotCache(t *testing.T) {
	repo := &hotSettingRepoStub{
		values: map[string]string{
			SettingKeyStreamTimeoutSettings: `{"enabled":true,"action":"temp_unsched","temp_unsched_minutes":9,"threshold_count":2,"threshold_window_minutes":5}`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	first, err := svc.GetStreamTimeoutSettings(context.Background())
	require.NoError(t, err)
	second, err := svc.GetStreamTimeoutSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, 9, first.TempUnschedMinutes)
	require.Equal(t, first.TempUnschedMinutes, second.TempUnschedMinutes)
	require.Equal(t, 1, repo.calls[SettingKeyStreamTimeoutSettings])
}
