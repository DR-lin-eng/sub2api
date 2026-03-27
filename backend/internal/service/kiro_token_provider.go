package service

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	kiroTokenRefreshSkew = 3 * time.Minute
	kiroTokenCacheSkew   = 5 * time.Minute
)

type KiroTokenProvider struct {
	accountRepo     AccountRepository
	tokenCache      GeminiTokenCache
	refreshAPI      *OAuthRefreshAPI
	executor        OAuthRefreshExecutor
	refreshPolicy   ProviderRefreshPolicy
	kiroUsageClient *KiroUsageService
}

func NewKiroTokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	kiroUsageClient *KiroUsageService,
) *KiroTokenProvider {
	return &KiroTokenProvider{
		accountRepo:     accountRepo,
		tokenCache:      tokenCache,
		kiroUsageClient: kiroUsageClient,
		refreshPolicy:   ClaudeProviderRefreshPolicy(),
	}
}

func (p *KiroTokenProvider) SetRefreshAPI(api *OAuthRefreshAPI, executor OAuthRefreshExecutor) {
	p.refreshAPI = api
	p.executor = executor
}

func (p *KiroTokenProvider) SetRefreshPolicy(policy ProviderRefreshPolicy) {
	p.refreshPolicy = policy
}

func (p *KiroTokenProvider) GetAccessToken(ctx context.Context, account *Account) (string, error) {
	if account == nil {
		return "", errors.New("account is nil")
	}
	if account.Platform != PlatformKiro || account.Type != AccountTypeOAuth {
		return "", errors.New("not a kiro oauth account")
	}

	cacheKey := KiroTokenCacheKey(account)
	if p.tokenCache != nil {
		if token, err := p.tokenCache.GetAccessToken(ctx, cacheKey); err == nil && token != "" {
			return token, nil
		}
	}

	expiresAt := account.GetCredentialAsTime("expires_at")
	needsRefresh := expiresAt == nil || time.Until(*expiresAt) <= kiroTokenRefreshSkew
	refreshFailed := false

	if needsRefresh && p.refreshAPI != nil && p.executor != nil {
		result, err := p.refreshAPI.RefreshIfNeeded(ctx, account, p.executor, kiroTokenRefreshSkew)
		if err != nil {
			if p.refreshPolicy.OnRefreshError == ProviderRefreshErrorReturn {
				return "", err
			}
			refreshFailed = true
		} else if result.LockHeld {
			if p.refreshPolicy.OnLockHeld == ProviderLockHeldWaitForCache && p.tokenCache != nil {
				time.Sleep(200 * time.Millisecond)
				if token, cacheErr := p.tokenCache.GetAccessToken(ctx, cacheKey); cacheErr == nil && token != "" {
					return token, nil
				}
			}
		} else {
			account = result.Account
			expiresAt = account.GetCredentialAsTime("expires_at")
		}
	}

	accessToken := account.GetCredential("access_token")
	if accessToken == "" {
		return "", errors.New("access_token not found in credentials")
	}

	if p.tokenCache != nil {
		latestAccount, isStale := CheckTokenVersion(ctx, account, p.accountRepo)
		if isStale && latestAccount != nil {
			accessToken = latestAccount.GetCredential("access_token")
			if accessToken == "" {
				return "", errors.New("access_token not found after version check")
			}
		} else {
			ttl := 30 * time.Minute
			if refreshFailed {
				if p.refreshPolicy.FailureTTL > 0 {
					ttl = p.refreshPolicy.FailureTTL
				} else {
					ttl = time.Minute
				}
			} else if expiresAt != nil {
				until := time.Until(*expiresAt)
				switch {
				case until > kiroTokenCacheSkew:
					ttl = until - kiroTokenCacheSkew
				case until > 0:
					ttl = until
				default:
					ttl = time.Minute
				}
			}
			_ = p.tokenCache.SetAccessToken(ctx, cacheKey, accessToken, ttl)
		}
	}

	return accessToken, nil
}

func KiroRegion(account *Account) string {
	if account == nil {
		return "us-east-1"
	}
	if value := account.GetCredential("api_region"); value != "" {
		return value
	}
	if value := account.GetCredential("auth_region"); value != "" {
		return value
	}
	if value := account.GetCredential("region"); value != "" {
		return value
	}
	return "us-east-1"
}

func KiroAuthRegion(account *Account) string {
	if account == nil {
		return "us-east-1"
	}
	if value := account.GetCredential("auth_region"); value != "" {
		return value
	}
	if value := account.GetCredential("region"); value != "" {
		return value
	}
	return "us-east-1"
}

func KiroVersion(account *Account) string {
	if account == nil || account.Extra == nil {
		return "0.10.0"
	}
	if value, ok := account.Extra["kiro_version"].(string); ok && value != "" {
		return value
	}
	return "0.10.0"
}

func KiroSystemVersion(account *Account) string {
	if account == nil || account.Extra == nil {
		return "darwin#24.6.0"
	}
	if value, ok := account.Extra["system_version"].(string); ok && value != "" {
		return value
	}
	return "darwin#24.6.0"
}

func KiroNodeVersion(account *Account) string {
	if account == nil || account.Extra == nil {
		return "22.21.1"
	}
	if value, ok := account.Extra["node_version"].(string); ok && value != "" {
		return value
	}
	return "22.21.1"
}

func KiroMachineID(account *Account) string {
	if account == nil {
		return ""
	}
	return fmt.Sprintf("%s|%s", account.GetCredential("machine_id"), account.GetCredential("refresh_token"))
}
