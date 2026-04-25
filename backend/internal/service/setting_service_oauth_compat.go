package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	settingKeyAvailableChannelsEnabled         = "available_channels_enabled"
	settingKeyChannelMonitorEnabled            = "channel_monitor_enabled"
	settingKeyWeChatConnectEnabled             = "wechat_connect_enabled"
	settingKeyWeChatConnectAppID               = "wechat_connect_app_id"
	settingKeyWeChatConnectAppSecret           = "wechat_connect_app_secret"
	settingKeyWeChatConnectOpenAppID           = "wechat_connect_open_app_id"
	settingKeyWeChatConnectOpenAppSecret       = "wechat_connect_open_app_secret"
	settingKeyWeChatConnectMPAppID             = "wechat_connect_mp_app_id"
	settingKeyWeChatConnectMPAppSecret         = "wechat_connect_mp_app_secret"
	settingKeyWeChatConnectMobileAppID         = "wechat_connect_mobile_app_id"
	settingKeyWeChatConnectMobileAppSecret     = "wechat_connect_mobile_app_secret"
	settingKeyWeChatConnectOpenEnabled         = "wechat_connect_open_enabled"
	settingKeyWeChatConnectMPEnabled           = "wechat_connect_mp_enabled"
	settingKeyWeChatConnectMobileEnabled       = "wechat_connect_mobile_enabled"
	settingKeyWeChatConnectMode                = "wechat_connect_mode"
	settingKeyWeChatConnectScopes              = "wechat_connect_scopes"
	settingKeyWeChatConnectRedirectURL         = "wechat_connect_redirect_url"
	settingKeyWeChatConnectFrontendRedirectURL = "wechat_connect_frontend_redirect_url"
	settingKeyOIDCConnectEnabled               = "oidc_connect_enabled"
	settingKeyOIDCConnectProviderName          = "oidc_connect_provider_name"
	settingKeyOIDCConnectClientID              = "oidc_connect_client_id"
	settingKeyOIDCConnectClientSecret          = "oidc_connect_client_secret"
	settingKeyOIDCConnectIssuerURL             = "oidc_connect_issuer_url"
	settingKeyOIDCConnectDiscoveryURL          = "oidc_connect_discovery_url"
	settingKeyOIDCConnectAuthorizeURL          = "oidc_connect_authorize_url"
	settingKeyOIDCConnectTokenURL              = "oidc_connect_token_url"
	settingKeyOIDCConnectUserInfoURL           = "oidc_connect_userinfo_url"
	settingKeyOIDCConnectJWKSURL               = "oidc_connect_jwks_url"
	settingKeyOIDCConnectScopes                = "oidc_connect_scopes"
	settingKeyOIDCConnectRedirectURL           = "oidc_connect_redirect_url"
	settingKeyOIDCConnectFrontendRedirectURL   = "oidc_connect_frontend_redirect_url"
	settingKeyOIDCConnectTokenAuthMethod       = "oidc_connect_token_auth_method"
	settingKeyOIDCConnectUsePKCE               = "oidc_connect_use_pkce"
	settingKeyOIDCConnectValidateIDToken       = "oidc_connect_validate_id_token"
	settingKeyOIDCConnectAllowedSigningAlgs    = "oidc_connect_allowed_signing_algs"
	settingKeyOIDCConnectClockSkewSeconds      = "oidc_connect_clock_skew_seconds"
	settingKeyOIDCConnectRequireEmailVerified  = "oidc_connect_require_email_verified"
	settingKeyOIDCConnectUserInfoEmailPath     = "oidc_connect_userinfo_email_path"
	settingKeyOIDCConnectUserInfoIDPath        = "oidc_connect_userinfo_id_path"
	settingKeyOIDCConnectUserInfoUsernamePath  = "oidc_connect_userinfo_username_path"
	defaultOIDCFrontendRedirect                = "/auth/oidc/callback"
	defaultOIDCAllowedSigningAlgs              = "RS256,ES256,PS256"
)

func (s *SettingService) GetChannelMonitorRuntime(ctx context.Context) ChannelMonitorRuntime {
	if s == nil || s.settingRepo == nil {
		return ChannelMonitorRuntime{}
	}
	value, err := s.getSettingValueCached(ctx, settingKeyChannelMonitorEnabled)
	if err != nil {
		return ChannelMonitorRuntime{}
	}
	return ChannelMonitorRuntime{Enabled: strings.TrimSpace(value) == "true"}
}

func (s *SettingService) GetAvailableChannelsRuntime(ctx context.Context) AvailableChannelsRuntime {
	if s == nil || s.settingRepo == nil {
		return AvailableChannelsRuntime{}
	}
	value, err := s.getSettingValueCached(ctx, settingKeyAvailableChannelsEnabled)
	if err != nil {
		return AvailableChannelsRuntime{}
	}
	return AvailableChannelsRuntime{Enabled: strings.TrimSpace(value) == "true"}
}

func parseWeChatCapabilityFlag(raw string, fallback bool) bool {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	return strings.TrimSpace(raw) == "true"
}

func normalizeWeChatStoredMode(openEnabled, mpEnabled, mobileEnabled bool, mode string) string {
	mode = normalizeWeChatConnectModeSetting(mode)
	switch mode {
	case "open":
		if openEnabled {
			return "open"
		}
	case "mp":
		if mpEnabled {
			return "mp"
		}
	case "mobile":
		if mobileEnabled {
			return "mobile"
		}
	}
	switch {
	case openEnabled:
		return "open"
	case mpEnabled:
		return "mp"
	case mobileEnabled:
		return "mobile"
	default:
		return mode
	}
}

func (s *SettingService) effectiveWeChatConnectOAuthConfig(settings map[string]string) WeChatConnectOAuthConfig {
	base := config.WeChatConnectConfig{}
	if s != nil && s.cfg != nil {
		base = s.cfg.WeChat
	}

	enabled := base.Enabled
	if raw, ok := settings[settingKeyWeChatConnectEnabled]; ok {
		enabled = strings.TrimSpace(raw) == "true"
	}

	legacyAppID := strings.TrimSpace(firstNonEmpty(
		settings[settingKeyWeChatConnectAppID],
		base.AppID,
		base.OpenAppID,
		base.MPAppID,
		base.MobileAppID,
	))
	legacyAppSecret := strings.TrimSpace(firstNonEmpty(
		settings[settingKeyWeChatConnectAppSecret],
		base.AppSecret,
		base.OpenAppSecret,
		base.MPAppSecret,
		base.MobileAppSecret,
	))
	openAppID := strings.TrimSpace(firstNonEmpty(settings[settingKeyWeChatConnectOpenAppID], base.OpenAppID, legacyAppID))
	openAppSecret := strings.TrimSpace(firstNonEmpty(settings[settingKeyWeChatConnectOpenAppSecret], base.OpenAppSecret, legacyAppSecret))
	mpAppID := strings.TrimSpace(firstNonEmpty(settings[settingKeyWeChatConnectMPAppID], base.MPAppID, legacyAppID))
	mpAppSecret := strings.TrimSpace(firstNonEmpty(settings[settingKeyWeChatConnectMPAppSecret], base.MPAppSecret, legacyAppSecret))
	mobileAppID := strings.TrimSpace(firstNonEmpty(settings[settingKeyWeChatConnectMobileAppID], base.MobileAppID, legacyAppID))
	mobileAppSecret := strings.TrimSpace(firstNonEmpty(settings[settingKeyWeChatConnectMobileAppSecret], base.MobileAppSecret, legacyAppSecret))

	modeRaw := firstNonEmpty(settings[settingKeyWeChatConnectMode], base.Mode)
	openEnabled := parseWeChatCapabilityFlag(settings[settingKeyWeChatConnectOpenEnabled], base.OpenEnabled)
	mpEnabled := parseWeChatCapabilityFlag(settings[settingKeyWeChatConnectMPEnabled], base.MPEnabled)
	mobileEnabled := parseWeChatCapabilityFlag(settings[settingKeyWeChatConnectMobileEnabled], base.MobileEnabled)
	if enabled && !openEnabled && !mpEnabled && !mobileEnabled {
		switch normalizeWeChatConnectModeSetting(modeRaw) {
		case "mp":
			mpEnabled = true
		case "mobile":
			mobileEnabled = true
		default:
			openEnabled = true
		}
	}
	mode := normalizeWeChatStoredMode(openEnabled, mpEnabled, mobileEnabled, modeRaw)

	return WeChatConnectOAuthConfig{
		Enabled:             enabled,
		LegacyAppID:         legacyAppID,
		LegacyAppSecret:     legacyAppSecret,
		OpenAppID:           openAppID,
		OpenAppSecret:       openAppSecret,
		MPAppID:             mpAppID,
		MPAppSecret:         mpAppSecret,
		MobileAppID:         mobileAppID,
		MobileAppSecret:     mobileAppSecret,
		OpenEnabled:         openEnabled,
		MPEnabled:           mpEnabled,
		MobileEnabled:       mobileEnabled,
		Mode:                mode,
		Scopes:              normalizeWeChatConnectScopeSetting(firstNonEmpty(settings[settingKeyWeChatConnectScopes], base.Scopes), mode),
		RedirectURL:         strings.TrimSpace(firstNonEmpty(settings[settingKeyWeChatConnectRedirectURL], base.RedirectURL)),
		FrontendRedirectURL: strings.TrimSpace(firstNonEmpty(settings[settingKeyWeChatConnectFrontendRedirectURL], base.FrontendRedirectURL, defaultWeChatConnectFrontend)),
	}
}

func (s *SettingService) parseWeChatConnectOAuthConfig(settings map[string]string) (WeChatConnectOAuthConfig, error) {
	cfg := s.effectiveWeChatConnectOAuthConfig(settings)

	if !cfg.Enabled || (!cfg.OpenEnabled && !cfg.MPEnabled && !cfg.MobileEnabled) {
		return WeChatConnectOAuthConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "wechat oauth is disabled")
	}
	if cfg.OpenEnabled {
		if cfg.AppIDForMode("open") == "" {
			return WeChatConnectOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat oauth pc app id not configured")
		}
		if cfg.AppSecretForMode("open") == "" {
			return WeChatConnectOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat oauth pc app secret not configured")
		}
	}
	if cfg.MPEnabled {
		if cfg.AppIDForMode("mp") == "" {
			return WeChatConnectOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat oauth official account app id not configured")
		}
		if cfg.AppSecretForMode("mp") == "" {
			return WeChatConnectOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat oauth official account app secret not configured")
		}
	}
	if cfg.MobileEnabled {
		if cfg.AppIDForMode("mobile") == "" {
			return WeChatConnectOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat oauth mobile app id not configured")
		}
		if cfg.AppSecretForMode("mobile") == "" {
			return WeChatConnectOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat oauth mobile app secret not configured")
		}
	}
	if value := strings.TrimSpace(cfg.RedirectURL); value != "" {
		if err := config.ValidateAbsoluteHTTPURL(value); err != nil {
			return WeChatConnectOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat oauth redirect url invalid")
		}
	}
	if err := config.ValidateFrontendRedirectURL(cfg.FrontendRedirectURL); err != nil {
		return WeChatConnectOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "wechat oauth frontend redirect url invalid")
	}
	return cfg, nil
}

func (s *SettingService) GetWeChatConnectOAuthConfig(ctx context.Context) (WeChatConnectOAuthConfig, error) {
	settings := map[string]string{}
	if s != nil && s.settingRepo != nil {
		keys := []string{
			settingKeyWeChatConnectEnabled,
			settingKeyWeChatConnectAppID,
			settingKeyWeChatConnectAppSecret,
			settingKeyWeChatConnectOpenAppID,
			settingKeyWeChatConnectOpenAppSecret,
			settingKeyWeChatConnectMPAppID,
			settingKeyWeChatConnectMPAppSecret,
			settingKeyWeChatConnectMobileAppID,
			settingKeyWeChatConnectMobileAppSecret,
			settingKeyWeChatConnectOpenEnabled,
			settingKeyWeChatConnectMPEnabled,
			settingKeyWeChatConnectMobileEnabled,
			settingKeyWeChatConnectMode,
			settingKeyWeChatConnectScopes,
			settingKeyWeChatConnectRedirectURL,
			settingKeyWeChatConnectFrontendRedirectURL,
		}
		loaded, err := s.settingRepo.GetMultiple(ctx, keys)
		if err != nil {
			return WeChatConnectOAuthConfig{}, fmt.Errorf("get wechat connect settings: %w", err)
		}
		settings = loaded
	}
	return s.parseWeChatConnectOAuthConfig(settings)
}

func oidcUsePKCECompatibilityDefault(base config.OIDCConnectConfig) bool {
	if base.UsePKCEExplicit {
		return base.UsePKCE
	}
	return true
}

func oidcValidateIDTokenCompatibilityDefault(base config.OIDCConnectConfig) bool {
	if base.ValidateIDTokenExplicit {
		return base.ValidateIDToken
	}
	return true
}

func (s *SettingService) GetOIDCConnectOAuthConfig(ctx context.Context) (config.OIDCConnectConfig, error) {
	effective := config.OIDCConnectConfig{}
	if s != nil && s.cfg != nil {
		effective = s.cfg.OIDC
	}

	if s != nil && s.settingRepo != nil {
		keys := []string{
			settingKeyOIDCConnectEnabled,
			settingKeyOIDCConnectProviderName,
			settingKeyOIDCConnectClientID,
			settingKeyOIDCConnectClientSecret,
			settingKeyOIDCConnectIssuerURL,
			settingKeyOIDCConnectDiscoveryURL,
			settingKeyOIDCConnectAuthorizeURL,
			settingKeyOIDCConnectTokenURL,
			settingKeyOIDCConnectUserInfoURL,
			settingKeyOIDCConnectJWKSURL,
			settingKeyOIDCConnectScopes,
			settingKeyOIDCConnectRedirectURL,
			settingKeyOIDCConnectFrontendRedirectURL,
			settingKeyOIDCConnectTokenAuthMethod,
			settingKeyOIDCConnectUsePKCE,
			settingKeyOIDCConnectValidateIDToken,
			settingKeyOIDCConnectAllowedSigningAlgs,
			settingKeyOIDCConnectClockSkewSeconds,
			settingKeyOIDCConnectRequireEmailVerified,
			settingKeyOIDCConnectUserInfoEmailPath,
			settingKeyOIDCConnectUserInfoIDPath,
			settingKeyOIDCConnectUserInfoUsernamePath,
		}
		settings, err := s.settingRepo.GetMultiple(ctx, keys)
		if err != nil {
			return config.OIDCConnectConfig{}, fmt.Errorf("get oidc connect settings: %w", err)
		}

		if raw, ok := settings[settingKeyOIDCConnectEnabled]; ok {
			effective.Enabled = strings.TrimSpace(raw) == "true"
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectProviderName]); value != "" {
			effective.ProviderName = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectClientID]); value != "" {
			effective.ClientID = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectClientSecret]); value != "" {
			effective.ClientSecret = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectIssuerURL]); value != "" {
			effective.IssuerURL = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectDiscoveryURL]); value != "" {
			effective.DiscoveryURL = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectAuthorizeURL]); value != "" {
			effective.AuthorizeURL = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectTokenURL]); value != "" {
			effective.TokenURL = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectUserInfoURL]); value != "" {
			effective.UserInfoURL = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectJWKSURL]); value != "" {
			effective.JWKSURL = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectScopes]); value != "" {
			effective.Scopes = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectRedirectURL]); value != "" {
			effective.RedirectURL = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectFrontendRedirectURL]); value != "" {
			effective.FrontendRedirectURL = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectTokenAuthMethod]); value != "" {
			effective.TokenAuthMethod = strings.ToLower(value)
		}
		if raw, ok := settings[settingKeyOIDCConnectUsePKCE]; ok {
			effective.UsePKCE = strings.TrimSpace(raw) == "true"
		} else {
			effective.UsePKCE = oidcUsePKCECompatibilityDefault(effective)
		}
		if raw, ok := settings[settingKeyOIDCConnectValidateIDToken]; ok {
			effective.ValidateIDToken = strings.TrimSpace(raw) == "true"
		} else {
			effective.ValidateIDToken = oidcValidateIDTokenCompatibilityDefault(effective)
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectAllowedSigningAlgs]); value != "" {
			effective.AllowedSigningAlgs = value
		}
		if raw := strings.TrimSpace(settings[settingKeyOIDCConnectClockSkewSeconds]); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				effective.ClockSkewSeconds = parsed
			}
		}
		if raw, ok := settings[settingKeyOIDCConnectRequireEmailVerified]; ok {
			effective.RequireEmailVerified = strings.TrimSpace(raw) == "true"
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectUserInfoEmailPath]); value != "" {
			effective.UserInfoEmailPath = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectUserInfoIDPath]); value != "" {
			effective.UserInfoIDPath = value
		}
		if value := strings.TrimSpace(settings[settingKeyOIDCConnectUserInfoUsernamePath]); value != "" {
			effective.UserInfoUsernamePath = value
		}
	}

	if !effective.Enabled {
		return config.OIDCConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	if strings.TrimSpace(effective.ProviderName) == "" {
		effective.ProviderName = "OIDC"
	}
	if strings.TrimSpace(effective.Scopes) == "" {
		effective.Scopes = "openid email profile"
	}
	if strings.TrimSpace(effective.FrontendRedirectURL) == "" {
		effective.FrontendRedirectURL = defaultOIDCFrontendRedirect
	}
	if strings.TrimSpace(effective.TokenAuthMethod) == "" {
		effective.TokenAuthMethod = "client_secret_post"
	}
	if strings.TrimSpace(effective.AllowedSigningAlgs) == "" {
		effective.AllowedSigningAlgs = defaultOIDCAllowedSigningAlgs
	}
	if effective.ClockSkewSeconds <= 0 {
		effective.ClockSkewSeconds = 120
	}

	if strings.TrimSpace(effective.ClientID) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client id not configured")
	}
	if strings.TrimSpace(effective.AuthorizeURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url not configured")
	}
	if strings.TrimSpace(effective.TokenURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url not configured")
	}
	if strings.TrimSpace(effective.UserInfoURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url not configured")
	}
	if strings.TrimSpace(effective.JWKSURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth jwks url not configured")
	}
	if strings.TrimSpace(effective.RedirectURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url not configured")
	}
	if strings.TrimSpace(effective.IssuerURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth issuer url not configured")
	}

	if err := config.ValidateAbsoluteHTTPURL(effective.AuthorizeURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.TokenURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.UserInfoURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.JWKSURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth jwks url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.RedirectURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url invalid")
	}
	if err := config.ValidateFrontendRedirectURL(effective.FrontendRedirectURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url invalid")
	}

	switch strings.ToLower(strings.TrimSpace(effective.TokenAuthMethod)) {
	case "", "client_secret_post", "client_secret_basic":
		if strings.TrimSpace(effective.ClientSecret) == "" {
			return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client secret not configured")
		}
	case "none":
		if !effective.UsePKCE {
			return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth pkce must be enabled when token_auth_method=none")
		}
	default:
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token_auth_method invalid")
	}

	return effective, nil
}
