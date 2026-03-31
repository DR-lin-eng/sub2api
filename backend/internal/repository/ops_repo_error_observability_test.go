package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsRepositoryInsertErrorLog_PersistsObservabilityFields(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	requestType := int16(service.RequestTypeStream)
	createdAt := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)
	input := &service.OpsInsertErrorLogInput{
		RequestID:        "req-1",
		ClientRequestID:  "creq-1",
		Platform:         "openai",
		Model:            "gpt-5",
		RequestPath:      "/v1/chat/completions",
		Stream:           true,
		InboundEndpoint:  "/v1/chat/completions",
		UpstreamEndpoint: "/v1/responses",
		RequestedModel:   "gpt-5",
		UpstreamModel:    "gpt-5.4-mini",
		RequestType:      &requestType,
		UserAgent:        "codex",
		ErrorPhase:       "upstream",
		ErrorType:        "upstream_error",
		Severity:         "P1",
		StatusCode:       502,
		ErrorMessage:     "upstream failed",
		ErrorBody:        `{"error":"bad gateway"}`,
		ErrorSource:      "upstream_http",
		ErrorOwner:       "provider",
		IsRetryable:      true,
		RetryCount:       1,
		CreatedAt:        createdAt,
	}

	mock.ExpectQuery("INSERT INTO ops_error_logs").
		WithArgs(anySliceToDriverValues(opsInsertErrorLogArgs(input))...).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))

	id, err := repo.InsertErrorLog(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, int64(7), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryListErrorLogs_ReturnsObservabilityFields(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	createdAt := time.Date(2026, 3, 29, 11, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM ops_error_logs e WHERE 1=1 AND COALESCE\\(e.status_code, 0\\) >= 400 AND COALESCE\\(e.is_business_limited,false\\) = false").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "created_at", "error_phase", "error_type", "error_owner", "error_source", "severity", "status_code",
		"platform", "model", "is_retryable", "retry_count", "resolved", "resolved_at", "resolved_by_user_id",
		"resolved_by_name", "resolved_retry_id", "client_request_id", "request_id", "error_message", "user_id",
		"user_email", "api_key_id", "account_id", "account_name", "group_id", "group_name", "client_ip",
		"request_path", "stream", "inbound_endpoint", "upstream_endpoint", "requested_model", "upstream_model", "request_type",
	}).AddRow(
		int64(5), createdAt, "upstream", "upstream_error", "provider", "upstream_http", "P1", int64(503),
		"openai", "gpt-5", true, 2, false, nil, nil,
		"", nil, "creq-5", "req-5", "provider overloaded", int64(10),
		"user@example.com", int64(11), int64(12), "acc-12", int64(13), "group-13", "127.0.0.1",
		"/v1/chat/completions", true, "/v1/chat/completions", "/v1/responses", "gpt-5", "gpt-5.4-mini", int64(service.RequestTypeStream),
	)

	mock.ExpectQuery("SELECT\\s+e.id,").
		WithArgs(20, 0).
		WillReturnRows(rows)

	result, err := repo.ListErrorLogs(context.Background(), &service.OpsErrorLogFilter{})
	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	require.Equal(t, "/v1/chat/completions", result.Errors[0].InboundEndpoint)
	require.Equal(t, "/v1/responses", result.Errors[0].UpstreamEndpoint)
	require.Equal(t, "gpt-5", result.Errors[0].RequestedModel)
	require.Equal(t, "gpt-5.4-mini", result.Errors[0].UpstreamModel)
	require.NotNil(t, result.Errors[0].RequestType)
	require.Equal(t, int16(service.RequestTypeStream), *result.Errors[0].RequestType)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryGetErrorLogByID_AllowsLegacyNullObservabilityFields(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	createdAt := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT\\s+e.id,").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "error_phase", "error_type", "error_owner", "error_source", "severity", "status_code",
			"platform", "model", "is_retryable", "retry_count", "resolved", "resolved_at", "resolved_by_user_id",
			"resolved_retry_id", "client_request_id", "request_id", "error_message", "error_body", "upstream_status_code",
			"upstream_error_message", "upstream_error_detail", "upstream_errors", "is_business_limited", "user_id",
			"user_email", "api_key_id", "account_id", "account_name", "group_id", "group_name", "client_ip",
			"request_path", "stream", "inbound_endpoint", "upstream_endpoint", "requested_model", "upstream_model",
			"request_type", "user_agent", "auth_latency_ms", "routing_latency_ms", "upstream_latency_ms",
			"response_latency_ms", "time_to_first_token_ms", "request_body", "request_body_truncated", "request_body_bytes", "request_headers",
		}).AddRow(
			int64(9), createdAt, "request", "api_error", "platform", "gateway", "P2", int64(502),
			"openai", "gpt-5", true, 0, false, nil, nil,
			nil, "creq-9", "req-9", "gateway failed", `{"error":"bad gateway"}`, nil,
			"", "", "null", false, nil,
			"", nil, nil, "", nil, "", nil,
			"/v1/chat/completions", false, "", "", "", "",
			nil, "codex", nil, nil, nil,
			nil, nil, "null", false, nil, "null",
		))

	detail, err := repo.GetErrorLogByID(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, "", detail.InboundEndpoint)
	require.Equal(t, "", detail.UpstreamEndpoint)
	require.Equal(t, "", detail.RequestedModel)
	require.Equal(t, "", detail.UpstreamModel)
	require.Nil(t, detail.RequestType)
	require.Equal(t, "", detail.RequestBody)
	require.Equal(t, "", detail.RequestHeaders)
	require.Equal(t, "", detail.UpstreamErrors)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsNullInt16(t *testing.T) {
	require.Equal(t, sql.NullInt64{}, opsNullInt16(nil))
	require.Equal(t, sql.NullInt64{}, opsNullInt16(ptrInt16(0)))
	require.Equal(t, sql.NullInt64{Int64: 2, Valid: true}, opsNullInt16(ptrInt16(2)))
}

func ptrInt16(v int16) *int16 {
	return &v
}
