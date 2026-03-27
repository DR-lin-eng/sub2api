package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	kiropkg "github.com/Wei-Shaw/sub2api/internal/pkg/kiro"
)

type KiroTokenRefresher struct{}

func NewKiroTokenRefresher() *KiroTokenRefresher {
	return &KiroTokenRefresher{}
}

func (r *KiroTokenRefresher) CacheKey(account *Account) string {
	return KiroTokenCacheKey(account)
}

func (r *KiroTokenRefresher) CanRefresh(account *Account) bool {
	return account != nil && account.Platform == PlatformKiro && account.Type == AccountTypeOAuth
}

func (r *KiroTokenRefresher) NeedsRefresh(account *Account, refreshWindow time.Duration) bool {
	expiresAt := account.GetCredentialAsTime("expires_at")
	if expiresAt == nil {
		return true
	}
	return time.Until(*expiresAt) < refreshWindow
}

func (r *KiroTokenRefresher) Refresh(ctx context.Context, account *Account) (map[string]any, error) {
	authMethod := strings.ToLower(strings.TrimSpace(account.GetCredential("auth_method")))
	if authMethod == "" {
		if account.GetCredential("client_id") != "" && account.GetCredential("client_secret") != "" {
			authMethod = "idc"
		} else {
			authMethod = "social"
		}
	}

	var (
		accessToken  string
		refreshToken = account.GetCredential("refresh_token")
		expiresAt    string
		profileARN   string
		err          error
	)

	switch authMethod {
	case "idc", "builder-id", "iam":
		accessToken, refreshToken, expiresAt, err = refreshKiroIDCToken(ctx, account)
	default:
		accessToken, refreshToken, expiresAt, profileARN, err = refreshKiroSocialToken(ctx, account)
	}
	if err != nil {
		return nil, err
	}

	newCreds := map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_at":    expiresAt,
	}
	if profileARN != "" {
		newCreds["profile_arn"] = profileARN
	}
	return MergeCredentials(account.Credentials, newCreds), nil
}

func refreshKiroSocialToken(ctx context.Context, account *Account) (accessToken, refreshToken, expiresAt, profileARN string, err error) {
	payload := map[string]any{
		"refreshToken": account.GetCredential("refresh_token"),
	}
	url := fmt.Sprintf("https://prod.%s.auth.desktop.kiro.dev/refreshToken", KiroAuthRegion(account))
	host := fmt.Sprintf("prod.%s.auth.desktop.kiro.dev", KiroAuthRegion(account))
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ProfileARN   string `json:"profileArn"`
		ExpiresIn    int64  `json:"expiresIn"`
	}
	if err = doKiroJSONRequest(ctx, account, url, host, payload, &out); err != nil {
		return "", "", "", "", err
	}
	refreshToken = out.RefreshToken
	if refreshToken == "" {
		refreshToken = account.GetCredential("refresh_token")
	}
	expiresAt = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second).UTC().Format(time.RFC3339)
	return out.AccessToken, refreshToken, expiresAt, out.ProfileARN, nil
}

func refreshKiroIDCToken(ctx context.Context, account *Account) (accessToken, refreshToken, expiresAt string, err error) {
	payload := map[string]any{
		"clientId":     account.GetCredential("client_id"),
		"clientSecret": account.GetCredential("client_secret"),
		"refreshToken": account.GetCredential("refresh_token"),
		"grantType":    "refresh_token",
	}
	url := fmt.Sprintf("https://oidc.%s.amazonaws.com/token", KiroAuthRegion(account))
	host := fmt.Sprintf("oidc.%s.amazonaws.com", KiroAuthRegion(account))
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int64  `json:"expiresIn"`
	}
	if err = doKiroJSONRequest(ctx, account, url, host, payload, &out); err != nil {
		return "", "", "", err
	}
	refreshToken = out.RefreshToken
	if refreshToken == "" {
		refreshToken = account.GetCredential("refresh_token")
	}
	expiresAt = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second).UTC().Format(time.RFC3339)
	return out.AccessToken, refreshToken, expiresAt, nil
}

func doKiroJSONRequest(ctx context.Context, account *Account, url, host string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL: accountProxyURL(account),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	machineID := kiropkg.GenerateMachineID(account.GetCredential("machine_id"), "", account.GetCredential("refresh_token"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Encoding", "gzip, compress, deflate, br")
	req.Header.Set("host", host)
	req.Header.Set("Connection", "close")
	req.Header.Set("User-Agent", fmt.Sprintf("KiroIDE-%s-%s", KiroVersion(account), machineID))
	if strings.Contains(url, "oidc.") {
		req.Header.Set("x-amz-user-agent", "aws-sdk-js/3.738.0 ua/2.1 os/other lang/js md/browser#unknown_unknown api/sso-oidc#3.738.0 m/E KiroIDE")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "*")
		req.Header.Set("sec-fetch-mode", "cors")
		req.Header.Set("User-Agent", "node")
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("kiro oauth refresh upstream returned %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
