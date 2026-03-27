package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/pkg/kiro"
)

type KiroUsageLimits struct {
	NextDateReset    *float64              `json:"nextDateReset"`
	SubscriptionInfo *KiroSubscriptionInfo `json:"subscriptionInfo"`
	UsageBreakdowns  []KiroUsageBreakdown  `json:"usageBreakdownList"`
}

type KiroSubscriptionInfo struct {
	SubscriptionTitle *string `json:"subscriptionTitle"`
}

type KiroUsageBreakdown struct {
	CurrentUsageWithPrecision float64          `json:"currentUsageWithPrecision"`
	UsageLimitWithPrecision   float64          `json:"usageLimitWithPrecision"`
	Bonuses                   []KiroUsageBonus `json:"bonuses"`
	FreeTrialInfo             *KiroFreeTrial   `json:"freeTrialInfo"`
	NextDateReset             *float64         `json:"nextDateReset"`
}

type KiroUsageBonus struct {
	CurrentUsage float64 `json:"currentUsage"`
	UsageLimit   float64 `json:"usageLimit"`
	Status       *string `json:"status"`
}

type KiroFreeTrial struct {
	CurrentUsageWithPrecision float64 `json:"currentUsageWithPrecision"`
	UsageLimitWithPrecision   float64 `json:"usageLimitWithPrecision"`
	FreeTrialStatus           *string `json:"freeTrialStatus"`
}

type KiroUsageService struct{}

func NewKiroUsageService() *KiroUsageService {
	return &KiroUsageService{}
}

func (s *KiroUsageService) FetchUsageLimits(ctx context.Context, account *Account, accessToken string) (*KiroUsageLimits, error) {
	if account == nil {
		return nil, fmt.Errorf("account is nil")
	}
	host := fmt.Sprintf("q.%s.amazonaws.com", KiroRegion(account))
	url := fmt.Sprintf("https://%s/getUsageLimits?origin=AI_EDITOR&resourceType=AGENTIC_REQUEST", host)
	if profileARN := account.GetCredential("profile_arn"); profileARN != "" {
		url += "&profileArn=" + profileARN
	}

	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL: accountProxyURL(account),
		Timeout:  60 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	machineID := kiro.GenerateMachineID(account.GetCredential("machine_id"), "", account.GetCredential("refresh_token"))
	kiroVersion := KiroVersion(account)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("host", host)
	req.Header.Set("amz-sdk-invocation-id", generateRequestID())
	req.Header.Set("amz-sdk-request", "attempt=1; max=1")
	req.Header.Set("x-amz-user-agent", fmt.Sprintf("aws-sdk-js/1.0.0 KiroIDE-%s-%s", kiroVersion, machineID))
	req.Header.Set("User-Agent", fmt.Sprintf("aws-sdk-js/1.0.0 ua/2.1 os/%s lang/js md/nodejs#%s api/codewhispererruntime#1.0.0 m/N,E KiroIDE-%s-%s", KiroSystemVersion(account), KiroNodeVersion(account), kiroVersion, machineID))
	req.Header.Set("Connection", "close")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("kiro usage upstream returned %d", resp.StatusCode)
	}

	var out KiroUsageLimits
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (k *KiroUsageLimits) SubscriptionTitle() string {
	if k == nil || k.SubscriptionInfo == nil || k.SubscriptionInfo.SubscriptionTitle == nil {
		return ""
	}
	return *k.SubscriptionInfo.SubscriptionTitle
}

func (k *KiroUsageLimits) CurrentUsage() float64 {
	breakdown := k.primaryBreakdown()
	if breakdown == nil {
		return 0
	}
	total := breakdown.CurrentUsageWithPrecision
	if breakdown.FreeTrialInfo != nil && breakdown.FreeTrialInfo.isActive() {
		total += breakdown.FreeTrialInfo.CurrentUsageWithPrecision
	}
	for _, bonus := range breakdown.Bonuses {
		if bonus.isActive() {
			total += bonus.CurrentUsage
		}
	}
	return total
}

func (k *KiroUsageLimits) UsageLimit() float64 {
	breakdown := k.primaryBreakdown()
	if breakdown == nil {
		return 0
	}
	total := breakdown.UsageLimitWithPrecision
	if breakdown.FreeTrialInfo != nil && breakdown.FreeTrialInfo.isActive() {
		total += breakdown.FreeTrialInfo.UsageLimitWithPrecision
	}
	for _, bonus := range breakdown.Bonuses {
		if bonus.isActive() {
			total += bonus.UsageLimit
		}
	}
	return total
}

func (k *KiroUsageLimits) Remaining() float64 {
	remaining := k.UsageLimit() - k.CurrentUsage()
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (k *KiroUsageLimits) ResetAt() *time.Time {
	if k == nil {
		return nil
	}
	if breakdown := k.primaryBreakdown(); breakdown != nil && breakdown.NextDateReset != nil {
		t := unixFloatToTime(*breakdown.NextDateReset)
		return &t
	}
	if k.NextDateReset != nil {
		t := unixFloatToTime(*k.NextDateReset)
		return &t
	}
	return nil
}

func (k *KiroUsageLimits) primaryBreakdown() *KiroUsageBreakdown {
	if k == nil || len(k.UsageBreakdowns) == 0 {
		return nil
	}
	return &k.UsageBreakdowns[0]
}

func (b KiroUsageBonus) isActive() bool {
	return b.Status != nil && *b.Status == "ACTIVE"
}

func (f *KiroFreeTrial) isActive() bool {
	return f != nil && f.FreeTrialStatus != nil && *f.FreeTrialStatus == "ACTIVE"
}

func unixFloatToTime(v float64) time.Time {
	sec := int64(v)
	nsec := int64((v - float64(sec)) * float64(time.Second))
	return time.Unix(sec, nsec).UTC()
}
