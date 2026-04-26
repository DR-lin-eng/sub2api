//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type authHandlerSettingRepoStub struct {
	values map[string]string
}

func (s *authHandlerSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected")
}
func (s *authHandlerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}
func (s *authHandlerSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}
func (s *authHandlerSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}
func (s *authHandlerSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}
func (s *authHandlerSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}
func (s *authHandlerSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type authHandlerUserRepoStub struct {
	usersByID    map[int64]*service.User
	usersByEmail map[string]*service.User
}

func (s *authHandlerUserRepoStub) cloneUser(user *service.User) *service.User {
	if user == nil {
		return nil
	}
	clone := *user
	return &clone
}

func (s *authHandlerUserRepoStub) Create(context.Context, *service.User) error { panic("unexpected") }
func (s *authHandlerUserRepoStub) GetByID(_ context.Context, id int64) (*service.User, error) {
	user, ok := s.usersByID[id]
	if !ok {
		return nil, service.ErrUserNotFound
	}
	return s.cloneUser(user), nil
}
func (s *authHandlerUserRepoStub) GetByEmail(_ context.Context, email string) (*service.User, error) {
	user, ok := s.usersByEmail[email]
	if !ok {
		return nil, service.ErrUserNotFound
	}
	return s.cloneUser(user), nil
}
func (s *authHandlerUserRepoStub) GetFirstAdmin(context.Context) (*service.User, error) {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) Update(_ context.Context, user *service.User) error {
	clone := *user
	s.usersByID[user.ID] = &clone
	s.usersByEmail[user.Email] = &clone
	return nil
}
func (s *authHandlerUserRepoStub) Delete(context.Context, int64) error { panic("unexpected") }
func (s *authHandlerUserRepoStub) List(context.Context, pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected")
}
func (s *authHandlerUserRepoStub) EnableTotp(context.Context, int64) error  { panic("unexpected") }
func (s *authHandlerUserRepoStub) DisableTotp(context.Context, int64) error { panic("unexpected") }

type authHandlerTotpCacheStub struct {
	loginSessions map[string]*service.TotpLoginSession
}

func (s *authHandlerTotpCacheStub) GetSetupSession(context.Context, int64) (*service.TotpSetupSession, error) {
	panic("unexpected")
}
func (s *authHandlerTotpCacheStub) SetSetupSession(context.Context, int64, *service.TotpSetupSession, time.Duration) error {
	panic("unexpected")
}
func (s *authHandlerTotpCacheStub) DeleteSetupSession(context.Context, int64) error {
	panic("unexpected")
}
func (s *authHandlerTotpCacheStub) GetLoginSession(_ context.Context, tempToken string) (*service.TotpLoginSession, error) {
	session, ok := s.loginSessions[tempToken]
	if !ok {
		return nil, service.ErrTotpSetupExpired
	}
	clone := *session
	return &clone, nil
}
func (s *authHandlerTotpCacheStub) SetLoginSession(_ context.Context, tempToken string, session *service.TotpLoginSession, _ time.Duration) error {
	if s.loginSessions == nil {
		s.loginSessions = map[string]*service.TotpLoginSession{}
	}
	clone := *session
	s.loginSessions[tempToken] = &clone
	return nil
}
func (s *authHandlerTotpCacheStub) DeleteLoginSession(_ context.Context, tempToken string) error {
	delete(s.loginSessions, tempToken)
	return nil
}
func (s *authHandlerTotpCacheStub) IncrementVerifyAttempts(context.Context, int64) (int, error) {
	panic("unexpected")
}
func (s *authHandlerTotpCacheStub) GetVerifyAttempts(context.Context, int64) (int, error) {
	return 0, nil
}
func (s *authHandlerTotpCacheStub) ClearVerifyAttempts(context.Context, int64) error {
	return nil
}

type authHandlerResponseEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Reason  string          `json:"reason"`
	Data    json.RawMessage `json:"data"`
}

type authHandlerBackendHarness struct {
	handler   *AuthHandler
	totpCache *authHandlerTotpCacheStub
	users     map[string]*service.User
}

func newAuthHandlerBackendHarness(t *testing.T) *authHandlerBackendHarness {
	t.Helper()

	admin := &service.User{ID: 1, Email: "admin@example.com", Role: service.RoleAdmin, Status: service.StatusActive}
	require.NoError(t, admin.SetPassword("secret"))
	member := &service.User{ID: 2, Email: "member@example.com", Role: service.RoleUser, Status: service.StatusActive}
	require.NoError(t, member.SetPassword("secret"))
	memberTotp := &service.User{ID: 3, Email: "member-totp@example.com", Role: service.RoleUser, Status: service.StatusActive, TotpEnabled: true}
	require.NoError(t, memberTotp.SetPassword("secret"))

	userRepo := &authHandlerUserRepoStub{
		usersByID: map[int64]*service.User{
			admin.ID:      admin,
			member.ID:     member,
			memberTotp.ID: memberTotp,
		},
		usersByEmail: map[string]*service.User{
			admin.Email:      admin,
			member.Email:     member,
			memberTotp.Email: memberTotp,
		},
	}
	settingRepo := &authHandlerSettingRepoStub{values: map[string]string{}}
	settingSvc := service.NewSettingService(settingRepo, &config.Config{})
	require.NoError(t, settingSvc.UpdateSettings(context.Background(), &service.SystemSettings{
		BackendModeEnabled: true,
		TotpEnabled:        true,
	}))

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                   "handler-backend-secret",
			ExpireHour:               1,
			AccessTokenExpireMinutes: 60,
		},
	}
	authSvc := service.NewAuthService(nil, userRepo, nil, nil, cfg, settingSvc, nil, nil, nil, nil, nil)
	userSvc := service.NewUserService(userRepo, nil, nil)
	totpCache := &authHandlerTotpCacheStub{loginSessions: map[string]*service.TotpLoginSession{}}
	totpSvc := service.NewTotpService(userRepo, nil, totpCache, settingSvc, nil, nil)

	return &authHandlerBackendHarness{
		handler:   NewAuthHandler(cfg, authSvc, userSvc, settingSvc, nil, nil, totpSvc),
		totpCache: totpCache,
		users: map[string]*service.User{
			"admin":       admin,
			"member":      member,
			"member_totp": memberTotp,
		},
	}
}

func runAuthGatewayJSON(t *testing.T, payload any, fn func(gatewayctx.GatewayContext)) (int, authHandlerResponseEnvelope) {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()
	ctx := gatewayctx.NewNative(req, rec, nil, req.RemoteAddr)

	fn(ctx)

	var envelope authHandlerResponseEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	return rec.Code, envelope
}

func TestAuthHandlerLoginGateway_BackendModeRejectsNonAdminAsInvalidCredentials(t *testing.T) {
	harness := newAuthHandlerBackendHarness(t)

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{name: "wrong password", email: harness.users["member"].Email, password: "wrong"},
		{name: "valid non-admin", email: harness.users["member"].Email, password: "secret"},
		{name: "valid non-admin totp", email: harness.users["member_totp"].Email, password: "secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, envelope := runAuthGatewayJSON(t, LoginRequest{
				Email:    tt.email,
				Password: tt.password,
			}, harness.handler.LoginGateway)

			require.Equal(t, http.StatusUnauthorized, code)
			require.Equal(t, "INVALID_CREDENTIALS", envelope.Reason)
			require.Empty(t, harness.totpCache.loginSessions)
		})
	}
}

func TestAuthHandlerLoginGateway_BackendModeAllowsAdminLogin(t *testing.T) {
	harness := newAuthHandlerBackendHarness(t)

	code, envelope := runAuthGatewayJSON(t, LoginRequest{
		Email:    harness.users["admin"].Email,
		Password: "secret",
	}, harness.handler.LoginGateway)

	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 0, envelope.Code)
	require.Contains(t, string(envelope.Data), "access_token")
}

func TestAuthHandlerLogin2FAGateway_BackendModeDeletesNonAdminPendingSession(t *testing.T) {
	harness := newAuthHandlerBackendHarness(t)

	harness.totpCache.loginSessions["temp-token"] = &service.TotpLoginSession{
		UserID:      harness.users["member_totp"].ID,
		Email:       harness.users["member_totp"].Email,
		TokenExpiry: time.Now().Add(5 * time.Minute),
	}

	code, envelope := runAuthGatewayJSON(t, Login2FARequest{
		TempToken: "temp-token",
		TotpCode:  "123456",
	}, harness.handler.Login2FAGateway)

	require.Equal(t, http.StatusUnauthorized, code)
	require.Equal(t, "INVALID_CREDENTIALS", envelope.Reason)
	require.Empty(t, harness.totpCache.loginSessions)
}
