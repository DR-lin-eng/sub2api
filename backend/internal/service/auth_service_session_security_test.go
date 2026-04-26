//go:build unit

package service

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type authSessionUserRepoStub struct {
	users         map[int64]*User
	usersByEmail  map[string]*User
	getByIDErr    error
	getByEmailErr error
	updateErr     error
}

func (s *authSessionUserRepoStub) Create(context.Context, *User) error { panic("unexpected") }
func (s *authSessionUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	user, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}
func (s *authSessionUserRepoStub) GetByEmail(_ context.Context, email string) (*User, error) {
	if s.getByEmailErr != nil {
		return nil, s.getByEmailErr
	}
	user, ok := s.usersByEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}
func (s *authSessionUserRepoStub) GetFirstAdmin(context.Context) (*User, error) { panic("unexpected") }
func (s *authSessionUserRepoStub) Update(_ context.Context, user *User) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	clone := *user
	s.users[user.ID] = &clone
	if s.usersByEmail == nil {
		s.usersByEmail = map[string]*User{}
	}
	s.usersByEmail[user.Email] = &clone
	return nil
}
func (s *authSessionUserRepoStub) Delete(context.Context, int64) error { panic("unexpected") }
func (s *authSessionUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *authSessionUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *authSessionUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected")
}
func (s *authSessionUserRepoStub) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected")
}
func (s *authSessionUserRepoStub) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected")
}
func (s *authSessionUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected")
}
func (s *authSessionUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (s *authSessionUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected")
}
func (s *authSessionUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected")
}
func (s *authSessionUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected")
}
func (s *authSessionUserRepoStub) EnableTotp(context.Context, int64) error { panic("unexpected") }
func (s *authSessionUserRepoStub) DisableTotp(context.Context, int64) error {
	panic("unexpected")
}

type memoryRefreshTokenCache struct {
	live                 map[string]*RefreshTokenData
	consumed             map[string]string
	userTokens           map[int64]map[string]struct{}
	familyTokens         map[string]map[string]struct{}
	deleteUserCalls      int
	deleteUserErr        error
	storeConsumedErr     error
	getConsumedErr       error
	deleteTokenFamilyErr error
}

func newMemoryRefreshTokenCache() *memoryRefreshTokenCache {
	return &memoryRefreshTokenCache{
		live:         map[string]*RefreshTokenData{},
		consumed:     map[string]string{},
		userTokens:   map[int64]map[string]struct{}{},
		familyTokens: map[string]map[string]struct{}{},
	}
}

func (c *memoryRefreshTokenCache) StoreRefreshToken(_ context.Context, tokenHash string, data *RefreshTokenData, _ time.Duration) error {
	clone := *data
	c.live[tokenHash] = &clone
	return nil
}

func (c *memoryRefreshTokenCache) GetRefreshToken(_ context.Context, tokenHash string) (*RefreshTokenData, error) {
	data, ok := c.live[tokenHash]
	if !ok {
		return nil, ErrRefreshTokenNotFound
	}
	clone := *data
	return &clone, nil
}

func (c *memoryRefreshTokenCache) StoreConsumedRefreshTokenFamily(_ context.Context, tokenHash string, familyID string, _ time.Duration) error {
	if c.storeConsumedErr != nil {
		return c.storeConsumedErr
	}
	c.consumed[tokenHash] = familyID
	return nil
}

func (c *memoryRefreshTokenCache) GetConsumedRefreshTokenFamily(_ context.Context, tokenHash string) (string, bool, error) {
	if c.getConsumedErr != nil {
		return "", false, c.getConsumedErr
	}
	familyID, ok := c.consumed[tokenHash]
	return familyID, ok, nil
}

func (c *memoryRefreshTokenCache) DeleteRefreshToken(_ context.Context, tokenHash string) error {
	if data, ok := c.live[tokenHash]; ok {
		if tokens := c.userTokens[data.UserID]; tokens != nil {
			delete(tokens, tokenHash)
			if len(tokens) == 0 {
				delete(c.userTokens, data.UserID)
			}
		}
		if tokens := c.familyTokens[data.FamilyID]; tokens != nil {
			delete(tokens, tokenHash)
			if len(tokens) == 0 {
				delete(c.familyTokens, data.FamilyID)
			}
		}
	}
	delete(c.live, tokenHash)
	return nil
}

func (c *memoryRefreshTokenCache) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
	c.deleteUserCalls++
	if c.deleteUserErr != nil {
		return c.deleteUserErr
	}
	for tokenHash := range c.userTokens[userID] {
		if err := c.DeleteRefreshToken(ctx, tokenHash); err != nil {
			return err
		}
	}
	delete(c.userTokens, userID)
	return nil
}

func (c *memoryRefreshTokenCache) DeleteTokenFamily(ctx context.Context, familyID string) error {
	if c.deleteTokenFamilyErr != nil {
		return c.deleteTokenFamilyErr
	}
	for tokenHash := range c.familyTokens[familyID] {
		if err := c.DeleteRefreshToken(ctx, tokenHash); err != nil {
			return err
		}
	}
	delete(c.familyTokens, familyID)
	return nil
}

func (c *memoryRefreshTokenCache) AddToUserTokenSet(_ context.Context, userID int64, tokenHash string, _ time.Duration) error {
	if c.userTokens[userID] == nil {
		c.userTokens[userID] = map[string]struct{}{}
	}
	c.userTokens[userID][tokenHash] = struct{}{}
	return nil
}

func (c *memoryRefreshTokenCache) AddToFamilyTokenSet(_ context.Context, familyID string, tokenHash string, _ time.Duration) error {
	if c.familyTokens[familyID] == nil {
		c.familyTokens[familyID] = map[string]struct{}{}
	}
	c.familyTokens[familyID][tokenHash] = struct{}{}
	return nil
}

func (c *memoryRefreshTokenCache) GetUserTokenHashes(_ context.Context, userID int64) ([]string, error) {
	hashes := make([]string, 0, len(c.userTokens[userID]))
	for tokenHash := range c.userTokens[userID] {
		hashes = append(hashes, tokenHash)
	}
	sort.Strings(hashes)
	return hashes, nil
}

func (c *memoryRefreshTokenCache) GetFamilyTokenHashes(_ context.Context, familyID string) ([]string, error) {
	hashes := make([]string, 0, len(c.familyTokens[familyID]))
	for tokenHash := range c.familyTokens[familyID] {
		hashes = append(hashes, tokenHash)
	}
	sort.Strings(hashes)
	return hashes, nil
}

func (c *memoryRefreshTokenCache) IsTokenInFamily(_ context.Context, familyID string, tokenHash string) (bool, error) {
	_, ok := c.familyTokens[familyID][tokenHash]
	return ok, nil
}

func newAuthSessionService(t *testing.T, user *User, cache *memoryRefreshTokenCache) (*AuthService, *authSessionUserRepoStub) {
	t.Helper()

	repo := &authSessionUserRepoStub{
		users:        map[int64]*User{user.ID: user},
		usersByEmail: map[string]*User{user.Email: user},
	}
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                   "auth-session-secret",
			ExpireHour:               1,
			AccessTokenExpireMinutes: 60,
			RefreshTokenExpireDays:   30,
		},
	}
	return NewAuthService(nil, repo, nil, cache, cfg, nil, nil, nil, nil, nil, nil), repo
}

func TestAuthServiceRevokeAllUserSessions_BumpsTokenVersionAndRevokesRefreshTokens(t *testing.T) {
	user := &User{ID: 1, Email: "user@example.com", Role: RoleUser, Status: StatusActive, TokenVersion: 7}
	cache := newMemoryRefreshTokenCache()
	svc, repo := newAuthSessionService(t, user, cache)

	pair, err := svc.GenerateTokenPair(context.Background(), user, "")
	require.NoError(t, err)
	require.NotEmpty(t, pair.RefreshToken)

	require.NoError(t, svc.RevokeAllUserSessions(context.Background(), user.ID))
	require.Equal(t, int64(8), repo.users[user.ID].TokenVersion)
	require.Equal(t, 1, cache.deleteUserCalls)

	_, err = svc.RefreshTokenPair(context.Background(), pair.RefreshToken)
	require.ErrorIs(t, err, ErrRefreshTokenInvalid)
}

func TestAuthServiceRevokeAllUserSessions_SwallowsRefreshCleanupFailureAfterTokenVersionBump(t *testing.T) {
	user := &User{ID: 2, Email: "user2@example.com", Role: RoleUser, Status: StatusActive, TokenVersion: 3}
	cache := newMemoryRefreshTokenCache()
	cache.deleteUserErr = errors.New("redis unavailable")
	svc, repo := newAuthSessionService(t, user, cache)

	err := svc.RevokeAllUserSessions(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, int64(4), repo.users[user.ID].TokenVersion)
	require.Equal(t, 1, cache.deleteUserCalls)
}

func TestAuthServiceRefreshTokenPair_ReplayRevokesFamily(t *testing.T) {
	user := &User{ID: 3, Email: "rotate@example.com", Role: RoleUser, Status: StatusActive, TokenVersion: 1}
	cache := newMemoryRefreshTokenCache()
	svc, _ := newAuthSessionService(t, user, cache)

	firstPair, err := svc.GenerateTokenPair(context.Background(), user, "")
	require.NoError(t, err)

	secondPair, err := svc.RefreshTokenPair(context.Background(), firstPair.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, secondPair.RefreshToken)

	_, err = svc.RefreshTokenPair(context.Background(), firstPair.RefreshToken)
	require.ErrorIs(t, err, ErrRefreshTokenReused)

	_, err = svc.RefreshTokenPair(context.Background(), secondPair.RefreshToken)
	require.ErrorIs(t, err, ErrRefreshTokenInvalid)
}

func TestAuthServiceRefreshTokenPair_RandomInvalidTokenIsNotMarkedAsReuse(t *testing.T) {
	user := &User{ID: 4, Email: "invalid@example.com", Role: RoleUser, Status: StatusActive, TokenVersion: 1}
	cache := newMemoryRefreshTokenCache()
	svc, _ := newAuthSessionService(t, user, cache)

	_, err := svc.RefreshTokenPair(context.Background(), "rt_deadbeef")
	require.ErrorIs(t, err, ErrRefreshTokenInvalid)
}

func TestAuthServiceRefreshTokenPair_ExpiredTokenStillReturnsExpired(t *testing.T) {
	user := &User{ID: 5, Email: "expired@example.com", Role: RoleUser, Status: StatusActive, TokenVersion: 1}
	cache := newMemoryRefreshTokenCache()
	svc, _ := newAuthSessionService(t, user, cache)

	refreshToken := "rt_expiredtoken"
	tokenHash := hashToken(refreshToken)
	data := &RefreshTokenData{
		UserID:       user.ID,
		TokenVersion: user.TokenVersion,
		FamilyID:     "family-expired",
		CreatedAt:    time.Now().Add(-2 * time.Hour),
		ExpiresAt:    time.Now().Add(-1 * time.Minute),
	}
	require.NoError(t, cache.StoreRefreshToken(context.Background(), tokenHash, data, time.Minute))
	require.NoError(t, cache.AddToUserTokenSet(context.Background(), user.ID, tokenHash, time.Minute))
	require.NoError(t, cache.AddToFamilyTokenSet(context.Background(), data.FamilyID, tokenHash, time.Minute))

	_, err := svc.RefreshTokenPair(context.Background(), refreshToken)
	require.ErrorIs(t, err, ErrRefreshTokenExpired)
}
