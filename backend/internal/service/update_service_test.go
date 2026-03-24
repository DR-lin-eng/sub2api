package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type updateServiceGitHubClientStub struct {
	repo string
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(_ context.Context, repo string) (*GitHubRelease, error) {
	s.repo = repo
	return &GitHubRelease{TagName: "v1.2.3"}, nil
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	return nil
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	return nil, nil
}

func TestUpdateServiceUsesConfiguredReleaseRepo(t *testing.T) {
	client := &updateServiceGitHubClientStub{}
	svc := NewUpdateService(nil, client, "1.0.0", "release", "https://github.com/example/sub2api.git")

	_, err := svc.fetchLatestRelease(context.Background())
	require.NoError(t, err)
	require.Equal(t, "example/sub2api", client.repo)
}

func TestUpdateServiceFallsBackToDefaultReleaseRepo(t *testing.T) {
	client := &updateServiceGitHubClientStub{}
	svc := NewUpdateService(nil, client, "1.0.0", "release", "")

	_, err := svc.fetchLatestRelease(context.Background())
	require.NoError(t, err)
	require.Equal(t, defaultReleaseRepo, client.repo)
}

func TestProvideUpdateServicePrefersConfigRepo(t *testing.T) {
	client := &updateServiceGitHubClientStub{}
	svc := ProvideUpdateService(
		nil,
		client,
		BuildInfo{
			Version:     "1.0.0",
			BuildType:   "release",
			ReleaseRepo: "build/sub2api",
		},
		&config.Config{
			Update: config.UpdateConfig{
				Repo: "config/sub2api",
			},
		},
	)

	_, err := svc.fetchLatestRelease(context.Background())
	require.NoError(t, err)
	require.Equal(t, "config/sub2api", client.repo)
}
