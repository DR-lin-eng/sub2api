//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

func TestListTLSFingerprintProfiles_FallsBackToConfigAndPersists(t *testing.T) {
	repo := newMockSettingRepo()
	cfg := &config.Config{}
	cfg.Gateway.TLSFingerprint = config.TLSFingerprintConfig{
		Enabled: true,
		Profiles: map[string]config.TLSProfileConfig{
			"chrome_120": {
				Name:         "Chrome 120",
				EnableGREASE: true,
				CipherSuites: []uint16{4866, 4867, 4866},
				Curves:       []uint16{29, 23},
				PointFormats: []uint8{0, 1, 0},
			},
		},
	}

	svc := NewSettingService(repo, cfg)
	result, err := svc.ListTLSFingerprintProfiles(context.Background())
	require.NoError(t, err)
	require.True(t, result.Enabled)
	require.Len(t, result.Items, 1)
	require.Equal(t, "chrome_120", result.Items[0].ProfileID)
	require.Equal(t, []uint16{4866, 4867}, result.Items[0].CipherSuites)
	require.Contains(t, repo.data, SettingKeyTLSFingerprintProfiles)
}

func TestTLSFingerprintProfileCRUD_AndLastDeleteForbidden(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	first, err := svc.CreateTLSFingerprintProfile(context.Background(), &TLSFingerprintProfile{
		ProfileID:    "alpha",
		Name:         "Alpha",
		Enabled:      true,
		EnableGREASE: true,
		CipherSuites: []uint16{4866, 4867, 4866},
	})
	require.NoError(t, err)
	require.Equal(t, "alpha", first.ProfileID)
	require.Equal(t, []uint16{4866, 4867}, first.CipherSuites)

	_, err = svc.CreateTLSFingerprintProfile(context.Background(), &TLSFingerprintProfile{
		ProfileID: "alpha",
		Name:      "Duplicate",
	})
	require.ErrorIs(t, err, ErrTLSFingerprintProfileExists)

	updated, err := svc.UpdateTLSFingerprintProfile(context.Background(), "alpha", &TLSFingerprintProfile{
		Name:         "Alpha 2",
		Enabled:      false,
		EnableGREASE: false,
		Curves:       []uint16{29, 23, 29},
		PointFormats: []uint8{0, 1, 1},
	})
	require.NoError(t, err)
	require.Equal(t, "Alpha 2", updated.Name)
	require.False(t, updated.Enabled)
	require.Equal(t, []uint16{29, 23}, updated.Curves)
	require.Equal(t, []uint8{0, 1}, updated.PointFormats)

	err = svc.DeleteTLSFingerprintProfile(context.Background(), "alpha")
	require.Error(t, err)
	require.Equal(t, "TLS_FINGERPRINT_PROFILE_LAST_DELETE_FORBIDDEN", infraerrors.Reason(err))

	_, err = svc.CreateTLSFingerprintProfile(context.Background(), &TLSFingerprintProfile{
		ProfileID: "beta",
		Name:      "Beta",
		Enabled:   true,
	})
	require.NoError(t, err)

	err = svc.DeleteTLSFingerprintProfile(context.Background(), "alpha")
	require.NoError(t, err)

	result, err := svc.ListTLSFingerprintProfiles(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "beta", result.Items[0].ProfileID)
}

func TestTLSFingerprintSettings_RefreshRuntimeUsesEnabledProfiles(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})
	tlsfingerprint.ReplaceGlobalRegistryProfiles(map[string]*tlsfingerprint.Profile{})
	t.Cleanup(func() {
		tlsfingerprint.ReplaceGlobalRegistryProfiles(map[string]*tlsfingerprint.Profile{})
	})

	_, err := svc.CreateTLSFingerprintProfile(context.Background(), &TLSFingerprintProfile{
		ProfileID: "alpha",
		Name:      "Alpha",
		Enabled:   true,
	})
	require.NoError(t, err)
	_, err = svc.CreateTLSFingerprintProfile(context.Background(), &TLSFingerprintProfile{
		ProfileID: "beta",
		Name:      "Beta",
		Enabled:   false,
	})
	require.NoError(t, err)

	err = svc.SetTLSFingerprintSettings(context.Background(), &TLSFingerprintSettings{Enabled: true})
	require.NoError(t, err)

	key, profile := tlsfingerprint.GlobalRegistry().GetProfileEntryByAccountID(0)
	require.NotNil(t, profile)
	require.Equal(t, "alpha", key)
	require.Equal(t, "Alpha", profile.Name)

	err = svc.SetTLSFingerprintSettings(context.Background(), &TLSFingerprintSettings{Enabled: false})
	require.NoError(t, err)

	key, profile = tlsfingerprint.GlobalRegistry().GetProfileEntryByAccountID(5)
	require.NotNil(t, profile)
	require.Equal(t, tlsfingerprint.DefaultProfileName, key)
}
