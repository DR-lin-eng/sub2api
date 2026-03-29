//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type gemini429AutoDeleteRepo struct {
	mockAccountRepoForGemini
	deletedID     int64
	rateLimitedID int64
}

func (r *gemini429AutoDeleteRepo) Delete(ctx context.Context, id int64) error {
	r.deletedID = id
	return nil
}

func (r *gemini429AutoDeleteRepo) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	r.rateLimitedID = id
	return nil
}

func TestGeminiHandleGeminiUpstreamError_AutoDelete429(t *testing.T) {
	repo := &gemini429AutoDeleteRepo{}
	settings := NewSettingService(&hotSettingRepoStub{
		values: map[string]string{
			SettingKeyAutoDelete429Accounts: "true",
		},
	}, &config.Config{})
	rateSvc := NewRateLimitService(repo, nil, nil, nil, nil)
	rateSvc.SetSettingService(settings)

	svc := &GeminiMessagesCompatService{
		accountRepo:      repo,
		rateLimitService: rateSvc,
	}
	account := &Account{
		ID:       42,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
	}

	svc.handleGeminiUpstreamError(
		context.Background(),
		account,
		http.StatusTooManyRequests,
		http.Header{},
		[]byte(`{"error":{"message":"too many requests"}}`),
	)

	require.Equal(t, account.ID, repo.deletedID)
	require.Zero(t, repo.rateLimitedID)
}
