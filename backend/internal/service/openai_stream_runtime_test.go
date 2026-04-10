package service

import (
	"net/http"
	"testing"
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
