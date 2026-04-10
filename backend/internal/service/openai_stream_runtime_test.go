package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestShouldMutateAccountStateFromOpenAIHealthPrefetch(t *testing.T) {
	tests := []struct {
		statusCode int
		want       bool
	}{
		{statusCode: http.StatusBadRequest, want: true},
		{statusCode: http.StatusUnauthorized, want: true},
		{statusCode: http.StatusPaymentRequired, want: true},
		{statusCode: http.StatusForbidden, want: true},
		{statusCode: http.StatusTooManyRequests, want: false},
		{statusCode: http.StatusBadGateway, want: false},
		{statusCode: http.StatusServiceUnavailable, want: false},
		{statusCode: http.StatusGatewayTimeout, want: false},
	}

	for _, tt := range tests {
		if got := shouldMutateAccountStateFromOpenAIHealthPrefetch(tt.statusCode); got != tt.want {
			t.Fatalf("shouldMutateAccountStateFromOpenAIHealthPrefetch(%d) = %v, want %v", tt.statusCode, got, tt.want)
		}
	}
}

func TestShouldPersistOpenAITempUnschedule(t *testing.T) {
	account := &Account{ID: 1, ProxyID: ptrInt64(10)}
	proxyErr := newProxyRequestFailoverError(account, "http://proxy", errors.New("dial tcp timeout"))
	if shouldPersistOpenAITempUnschedule(proxyErr) {
		t.Fatalf("expected proxy/network failure to avoid persistent temp unschedule")
	}

	tokenErr := &UpstreamFailoverError{
		StatusCode:           http.StatusUnauthorized,
		TempUnscheduleFor:    20 * time.Minute,
		TempUnscheduleReason: "openai token invalidated (auto temp-unschedule 20m)",
		ResponseBody:         []byte(`{"error":{"code":"token_invalidated"}}`),
	}
	if !shouldPersistOpenAITempUnschedule(tokenErr) {
		t.Fatalf("expected token invalidated failure to persist temp unschedule")
	}
}

func TestOpenAITempUnscheduleRetryableError_ProxyFailureUsesCircuitOnly(t *testing.T) {
	failedProxyID := int64(10)
	accountRepo := &proxyFailoverAccountRepoStub{
		accounts: map[int64]*Account{
			1: {
				ID:          1,
				Name:        "openai-1",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				ProxyID:     &failedProxyID,
			},
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:            accountRepo,
		proxyCircuit:           newOpenAICircuitBreaker(1, time.Minute),
		accountCircuit:         newOpenAICircuitBreaker(1, time.Minute),
		tempUnscheduleThrottle: openAITempUnscheduleWriteThrottle(nil),
		runtimeSyncWake:        make(chan struct{}, 1),
		runtimeSyncStop:        make(chan struct{}),
		runtimeSyncPending:     make(map[int64]struct{}),
	}

	failoverErr := newProxyRequestFailoverError(accountRepo.accounts[1], "http://failed-proxy:8080", errors.New("dial tcp timeout"))
	svc.TempUnscheduleRetryableError(context.Background(), 1, failoverErr)

	if len(accountRepo.tempUnschedule) != 0 {
		t.Fatalf("expected no persistent temp unschedule writes for proxy failure, got %d", len(accountRepo.tempUnschedule))
	}
	if !svc.isOpenAICircuitBlocked(accountRepo.accounts[1]) {
		t.Fatalf("expected proxy/account circuit to block failed account after proxy failure")
	}
}

func ptrInt64(v int64) *int64 { return &v }
