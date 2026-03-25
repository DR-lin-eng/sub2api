//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminAuthJWTValidatesTokenVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil)

	admin := &service.User{
		ID:           1,
		Email:        "admin@example.com",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
		TokenVersion: 2,
		Concurrency:  1,
	}

	userRepo := &stubUserRepo{
		getByID: func(ctx context.Context, id int64) (*service.User, error) {
			if id != admin.ID {
				return nil, service.ErrUserNotFound
			}
			clone := *admin
			return &clone, nil
		},
	}
	userService := service.NewUserService(userRepo, nil, nil)

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAdminAuthMiddleware(authService, userService, nil)))
	router.GET("/t", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	t.Run("token_version_mismatch_rejected", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion - 1,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "TOKEN_REVOKED")
	})

	t.Run("token_version_match_allows", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("websocket_token_version_mismatch_rejected", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion - 1,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Sec-WebSocket-Protocol", "sub2api-admin, jwt."+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "TOKEN_REVOKED")
	})

	t.Run("websocket_token_version_match_allows", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Sec-WebSocket-Protocol", "sub2api-admin, jwt."+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAdminAuthAPIKeyRespectsBoundAdminTokenVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	admin := &service.User{
		ID:           7,
		Email:        "admin@example.com",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
		TokenVersion: 3,
		Concurrency:  2,
	}
	repo := &settingRepoStub{values: map[string]string{}}
	settingService := service.NewSettingService(repo, &config.Config{})
	key, err := settingService.GenerateAdminAPIKey(context.Background(), admin.ID, admin.TokenVersion)
	require.NoError(t, err)

	userRepo := &stubUserRepo{
		getByID: func(ctx context.Context, id int64) (*service.User, error) {
			if id != admin.ID {
				return nil, service.ErrUserNotFound
			}
			clone := *admin
			return &clone, nil
		},
		getFirstAdmin: func(ctx context.Context) (*service.User, error) {
			clone := *admin
			return &clone, nil
		},
	}
	userService := service.NewUserService(userRepo, nil, nil)

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAdminAuthMiddleware(nil, userService, settingService)))
	router.GET("/t", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	t.Run("bound_admin_allows", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("x-api-key", key)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("token_version_change_revokes_key", func(t *testing.T) {
		admin.TokenVersion++
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("x-api-key", key)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "INVALID_ADMIN_KEY")
	})
}

func TestAdminAuthAPIKeyRejectsLegacyUnboundKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingRepoStub{
		values: map[string]string{
			service.SettingKeyAdminAPIKey: "admin-legacy-test-key",
		},
	}
	settingService := service.NewSettingService(repo, &config.Config{})
	userService := service.NewUserService(&stubUserRepo{
		getFirstAdmin: func(ctx context.Context) (*service.User, error) {
			return &service.User{ID: 1, Role: service.RoleAdmin, Status: service.StatusActive}, nil
		},
	}, nil, nil)

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAdminAuthMiddleware(nil, userService, settingService)))
	router.GET("/t", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/t", nil)
	req.Header.Set("x-api-key", "admin-legacy-test-key")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "INVALID_ADMIN_KEY")
}

type stubUserRepo struct {
	getByID      func(ctx context.Context, id int64) (*service.User, error)
	getFirstAdmin func(ctx context.Context) (*service.User, error)
}

func (s *stubUserRepo) Create(ctx context.Context, user *service.User) error {
	panic("unexpected Create call")
}

func (s *stubUserRepo) GetByID(ctx context.Context, id int64) (*service.User, error) {
	if s.getByID == nil {
		panic("GetByID not stubbed")
	}
	return s.getByID(ctx, id)
}

func (s *stubUserRepo) GetByEmail(ctx context.Context, email string) (*service.User, error) {
	panic("unexpected GetByEmail call")
}

func (s *stubUserRepo) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	if s.getFirstAdmin == nil {
		panic("GetFirstAdmin not stubbed")
	}
	return s.getFirstAdmin(ctx)
}

func (s *stubUserRepo) Update(ctx context.Context, user *service.User) error {
	panic("unexpected Update call")
}

func (s *stubUserRepo) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *stubUserRepo) List(ctx context.Context, params pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *stubUserRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *stubUserRepo) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	panic("unexpected UpdateBalance call")
}

func (s *stubUserRepo) DeductBalance(ctx context.Context, id int64, amount float64) error {
	panic("unexpected DeductBalance call")
}

func (s *stubUserRepo) UpdateConcurrency(ctx context.Context, id int64, amount int) error {
	panic("unexpected UpdateConcurrency call")
}

func (s *stubUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}

func (s *stubUserRepo) RemoveGroupFromAllowedGroups(ctx context.Context, groupID int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}

func (s *stubUserRepo) RemoveGroupFromUserAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}

func (s *stubUserRepo) AddGroupToAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}

func (s *stubUserRepo) UpdateTotpSecret(ctx context.Context, userID int64, encryptedSecret *string) error {
	panic("unexpected UpdateTotpSecret call")
}

func (s *stubUserRepo) EnableTotp(ctx context.Context, userID int64) error {
	panic("unexpected EnableTotp call")
}

func (s *stubUserRepo) DisableTotp(ctx context.Context, userID int64) error {
	panic("unexpected DisableTotp call")
}

type settingRepoStub struct {
	values map[string]string
}

func (s *settingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *settingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *settingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *settingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}
