// Package routes provides HTTP route registration and handlers.
package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

func ExecutableAdminRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil || h.Admin == nil {
		return nil
	}
	out := make([]gatewayctx.RouteDef, 0, 12)
	if h.Admin.Dashboard != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/snapshot-v2", Handler: h.Admin.Dashboard.GetSnapshotV2Gateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/stats", Handler: h.Admin.Dashboard.GetStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/realtime", Handler: h.Admin.Dashboard.GetRealtimeMetricsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/trend", Handler: h.Admin.Dashboard.GetUsageTrendGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/models", Handler: h.Admin.Dashboard.GetModelStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/groups", Handler: h.Admin.Dashboard.GetGroupStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/api-keys-trend", Handler: h.Admin.Dashboard.GetAPIKeyUsageTrendGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/users-trend", Handler: h.Admin.Dashboard.GetUserUsageTrendGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/users-ranking", Handler: h.Admin.Dashboard.GetUserSpendingRankingGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/dashboard/users-usage", Handler: h.Admin.Dashboard.GetBatchUsersUsageGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/dashboard/api-keys-usage", Handler: h.Admin.Dashboard.GetBatchAPIKeysUsageGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/dashboard/user-breakdown", Handler: h.Admin.Dashboard.GetUserBreakdownGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/dashboard/aggregation/backfill", Handler: h.Admin.Dashboard.BackfillAggregationGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Group != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/groups", Handler: h.Admin.Group.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/groups/all", Handler: h.Admin.Group.GetAllGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/groups/:id", Handler: h.Admin.Group.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/groups/:id/stats", Handler: h.Admin.Group.GetStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/groups/usage-summary", Handler: h.Admin.Group.GetUsageSummaryGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/groups/capacity-summary", Handler: h.Admin.Group.GetCapacitySummaryGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/groups/sort-order", Handler: h.Admin.Group.UpdateSortOrderGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/groups/:id/rate-multipliers", Handler: h.Admin.Group.GetGroupRateMultipliersGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/groups/:id/rate-multipliers", Handler: h.Admin.Group.BatchSetGroupRateMultipliersGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/groups/:id/rate-multipliers", Handler: h.Admin.Group.ClearGroupRateMultipliersGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/groups/:id/api-keys", Handler: h.Admin.Group.GetGroupAPIKeysGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/groups", Handler: h.Admin.Group.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/groups/:id", Handler: h.Admin.Group.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/groups/:id", Handler: h.Admin.Group.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Announcement != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/announcements", Handler: h.Admin.Announcement.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/announcements", Handler: h.Admin.Announcement.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/announcements/:id", Handler: h.Admin.Announcement.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/announcements/:id", Handler: h.Admin.Announcement.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/announcements/:id", Handler: h.Admin.Announcement.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/announcements/:id/read-status", Handler: h.Admin.Announcement.ListReadStatusGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Promo != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/promo-codes", Handler: h.Admin.Promo.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/promo-codes", Handler: h.Admin.Promo.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/promo-codes/:id", Handler: h.Admin.Promo.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/promo-codes/:id", Handler: h.Admin.Promo.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/promo-codes/:id", Handler: h.Admin.Promo.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/promo-codes/:id/usages", Handler: h.Admin.Promo.GetUsagesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Backup != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/backups/s3-config", Handler: h.Admin.Backup.GetS3ConfigGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/backups/s3-config", Handler: h.Admin.Backup.UpdateS3ConfigGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/backups/s3-config/test", Handler: h.Admin.Backup.TestS3ConnectionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/backups/schedule", Handler: h.Admin.Backup.GetScheduleGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/backups/schedule", Handler: h.Admin.Backup.UpdateScheduleGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/backups", Handler: h.Admin.Backup.CreateBackupGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/backups", Handler: h.Admin.Backup.ListBackupsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/backups/:id", Handler: h.Admin.Backup.GetBackupGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/backups/:id", Handler: h.Admin.Backup.DeleteBackupGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/backups/:id/download-url", Handler: h.Admin.Backup.GetDownloadURLGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/backups/:id/restore", Handler: h.Admin.Backup.RestoreBackupGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.DataManagement != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/data-management/agent/health", Handler: h.Admin.DataManagement.GetAgentHealthGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/data-management/config", Handler: h.Admin.DataManagement.GetConfigGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/data-management/config", Handler: h.Admin.DataManagement.UpdateConfigGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/data-management/sources/:source_type/profiles", Handler: h.Admin.DataManagement.ListSourceProfilesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/data-management/sources/:source_type/profiles", Handler: h.Admin.DataManagement.CreateSourceProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id", Handler: h.Admin.DataManagement.UpdateSourceProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id", Handler: h.Admin.DataManagement.DeleteSourceProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/data-management/sources/:source_type/profiles/:profile_id/activate", Handler: h.Admin.DataManagement.SetActiveSourceProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/data-management/s3/test", Handler: h.Admin.DataManagement.TestS3Gateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/data-management/s3/profiles", Handler: h.Admin.DataManagement.ListS3ProfilesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/data-management/s3/profiles", Handler: h.Admin.DataManagement.CreateS3ProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/data-management/s3/profiles/:profile_id", Handler: h.Admin.DataManagement.UpdateS3ProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/data-management/s3/profiles/:profile_id", Handler: h.Admin.DataManagement.DeleteS3ProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/data-management/s3/profiles/:profile_id/activate", Handler: h.Admin.DataManagement.SetActiveS3ProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/data-management/backups", Handler: h.Admin.DataManagement.CreateBackupJobGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/data-management/backups", Handler: h.Admin.DataManagement.ListBackupJobsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/data-management/backups/:job_id", Handler: h.Admin.DataManagement.GetBackupJobGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Redeem != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/redeem-codes", Handler: h.Admin.Redeem.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/redeem-codes/stats", Handler: h.Admin.Redeem.GetStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/redeem-codes/export", Handler: h.Admin.Redeem.ExportGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/redeem-codes/:id", Handler: h.Admin.Redeem.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/redeem-codes/create-and-redeem", Handler: h.Admin.Redeem.CreateAndRedeemGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/redeem-codes/generate", Handler: h.Admin.Redeem.GenerateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/redeem-codes/:id", Handler: h.Admin.Redeem.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/redeem-codes/batch-delete", Handler: h.Admin.Redeem.BatchDeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/redeem-codes/:id/expire", Handler: h.Admin.Redeem.ExpireGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.User != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/users", Handler: h.Admin.User.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/users/:id", Handler: h.Admin.User.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/users", Handler: h.Admin.User.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/users/:id", Handler: h.Admin.User.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/users/:id", Handler: h.Admin.User.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/users/:id/balance", Handler: h.Admin.User.UpdateBalanceGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/users/:id/api-keys", Handler: h.Admin.User.GetUserAPIKeysGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/users/:id/usage", Handler: h.Admin.User.GetUserUsageGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/users/:id/balance-history", Handler: h.Admin.User.GetBalanceHistoryGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/users/:id/replace-group", Handler: h.Admin.User.ReplaceGroupGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Subscription != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/subscriptions", Handler: h.Admin.Subscription.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/subscriptions/:id", Handler: h.Admin.Subscription.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/subscriptions/:id/progress", Handler: h.Admin.Subscription.GetProgressGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/subscriptions/assign", Handler: h.Admin.Subscription.AssignGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/subscriptions/bulk-assign", Handler: h.Admin.Subscription.BulkAssignGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/subscriptions/:id/extend", Handler: h.Admin.Subscription.ExtendGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/subscriptions/:id/reset-quota", Handler: h.Admin.Subscription.ResetQuotaGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/subscriptions/:id", Handler: h.Admin.Subscription.RevokeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/groups/:id/subscriptions", Handler: h.Admin.Subscription.ListByGroupGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/users/:id/subscriptions", Handler: h.Admin.Subscription.ListByUserGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Usage != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/usage", Handler: h.Admin.Usage.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/usage/stats", Handler: h.Admin.Usage.StatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/usage/search-users", Handler: h.Admin.Usage.SearchUsersGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/usage/search-api-keys", Handler: h.Admin.Usage.SearchAPIKeysGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/usage/cleanup-tasks", Handler: h.Admin.Usage.ListCleanupTasksGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/usage/cleanup-tasks", Handler: h.Admin.Usage.CreateCleanupTaskGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/usage/cleanup-tasks/:id/cancel", Handler: h.Admin.Usage.CancelCleanupTaskGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.APIKey != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/api-keys/:id", Handler: h.Admin.APIKey.UpdateGroupGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.UserAttribute != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/user-attributes", Handler: h.Admin.UserAttribute.ListDefinitionsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/user-attributes", Handler: h.Admin.UserAttribute.CreateDefinitionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/user-attributes/batch", Handler: h.Admin.UserAttribute.GetBatchUserAttributesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/user-attributes/reorder", Handler: h.Admin.UserAttribute.ReorderDefinitionsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/user-attributes/:id", Handler: h.Admin.UserAttribute.UpdateDefinitionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/user-attributes/:id", Handler: h.Admin.UserAttribute.DeleteDefinitionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/users/:id/attributes", Handler: h.Admin.UserAttribute.GetUserAttributesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/users/:id/attributes", Handler: h.Admin.UserAttribute.UpdateUserAttributesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.ScheduledTest != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/scheduled-test-plans", Handler: h.Admin.ScheduledTest.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/scheduled-test-plans/:id", Handler: h.Admin.ScheduledTest.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/scheduled-test-plans/:id", Handler: h.Admin.ScheduledTest.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/scheduled-test-plans/:id/results", Handler: h.Admin.ScheduledTest.ListResultsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/:id/scheduled-test-plans", Handler: h.Admin.ScheduledTest.ListByAccountGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.ErrorPassthrough != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/error-passthrough-rules", Handler: h.Admin.ErrorPassthrough.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/error-passthrough-rules/:id", Handler: h.Admin.ErrorPassthrough.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/error-passthrough-rules", Handler: h.Admin.ErrorPassthrough.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/error-passthrough-rules/:id", Handler: h.Admin.ErrorPassthrough.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/error-passthrough-rules/:id", Handler: h.Admin.ErrorPassthrough.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.ProxyMaintenance != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/proxy-maintenance-plans", Handler: h.Admin.ProxyMaintenance.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/proxy-maintenance-plans", Handler: h.Admin.ProxyMaintenance.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/proxy-maintenance-plans/:id", Handler: h.Admin.ProxyMaintenance.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/proxy-maintenance-plans/:id", Handler: h.Admin.ProxyMaintenance.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/proxy-maintenance-plans/:id/results", Handler: h.Admin.ProxyMaintenance.ListResultsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/proxy-maintenance/run-now", Handler: h.Admin.ProxyMaintenance.RunNowGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/proxy-maintenance/tasks/:task_id", Handler: h.Admin.ProxyMaintenance.GetTaskGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Proxy != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/proxies", Handler: h.Admin.Proxy.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/proxies/all", Handler: h.Admin.Proxy.GetAllGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/proxies/data", Handler: h.Admin.Proxy.ExportDataGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/proxies/data", Handler: h.Admin.Proxy.ImportDataGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/proxies/:id", Handler: h.Admin.Proxy.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/proxies", Handler: h.Admin.Proxy.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/proxies/:id", Handler: h.Admin.Proxy.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/proxies/:id", Handler: h.Admin.Proxy.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/proxies/batch-delete", Handler: h.Admin.Proxy.BatchDeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/proxies/:id/test", Handler: h.Admin.Proxy.TestGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/proxies/:id/quality-check", Handler: h.Admin.Proxy.CheckQualityGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/proxies/:id/stats", Handler: h.Admin.Proxy.GetStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/proxies/:id/accounts", Handler: h.Admin.Proxy.GetProxyAccountsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/proxies/batch", Handler: h.Admin.Proxy.BatchCreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.OpenAIOAuth != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/openai/generate-auth-url", Handler: h.Admin.OpenAIOAuth.GenerateAuthURLGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/openai/exchange-code", Handler: h.Admin.OpenAIOAuth.ExchangeCodeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/openai/refresh-token", Handler: h.Admin.OpenAIOAuth.RefreshTokenGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/openai/st2at", Handler: h.Admin.OpenAIOAuth.ExchangeOpenAIChatWebSessionTokenGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/openai/at2info", Handler: h.Admin.OpenAIOAuth.InspectOpenAIChatWebAccessTokenGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/openai/accounts/:id/refresh", Handler: h.Admin.OpenAIOAuth.RefreshAccountTokenGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/openai/create-from-oauth", Handler: h.Admin.OpenAIOAuth.CreateAccountFromOAuthGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/sora/generate-auth-url", Handler: h.Admin.OpenAIOAuth.GenerateAuthURLGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/sora/exchange-code", Handler: h.Admin.OpenAIOAuth.ExchangeCodeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/sora/refresh-token", Handler: h.Admin.OpenAIOAuth.RefreshTokenGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/sora/st2at", Handler: h.Admin.OpenAIOAuth.ExchangeSoraSessionTokenGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/sora/rt2at", Handler: h.Admin.OpenAIOAuth.RefreshTokenGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/sora/accounts/:id/refresh", Handler: h.Admin.OpenAIOAuth.RefreshAccountTokenGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/sora/create-from-oauth", Handler: h.Admin.OpenAIOAuth.CreateAccountFromOAuthGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.GeminiOAuth != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/gemini/oauth/auth-url", Handler: h.Admin.GeminiOAuth.GenerateAuthURLGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/gemini/oauth/exchange-code", Handler: h.Admin.GeminiOAuth.ExchangeCodeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/gemini/oauth/capabilities", Handler: h.Admin.GeminiOAuth.GetCapabilitiesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.AntigravityOAuth != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/antigravity/oauth/auth-url", Handler: h.Admin.AntigravityOAuth.GenerateAuthURLGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/antigravity/oauth/exchange-code", Handler: h.Admin.AntigravityOAuth.ExchangeCodeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/antigravity/oauth/refresh-token", Handler: h.Admin.AntigravityOAuth.RefreshTokenGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Ops != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/concurrency", Handler: h.Admin.Ops.GetConcurrencyStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/user-concurrency", Handler: h.Admin.Ops.GetUserConcurrencyStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/account-availability", Handler: h.Admin.Ops.GetAccountAvailabilityGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/realtime-traffic", Handler: h.Admin.Ops.GetRealtimeTrafficSummaryGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/gateway-scheduler", Handler: h.Admin.Ops.GetGatewaySchedulerRuntimeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/openai-ws-runtime", Handler: h.Admin.Ops.GetOpenAIWSRuntimeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/ws/qps", Handler: h.Admin.Ops.QPSWSHandlerGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/dashboard/overview", Handler: h.Admin.Ops.GetDashboardOverviewGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/dashboard/throughput-trend", Handler: h.Admin.Ops.GetDashboardThroughputTrendGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/dashboard/latency-histogram", Handler: h.Admin.Ops.GetDashboardLatencyHistogramGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/dashboard/error-trend", Handler: h.Admin.Ops.GetDashboardErrorTrendGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/dashboard/error-distribution", Handler: h.Admin.Ops.GetDashboardErrorDistributionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/dashboard/openai-token-stats", Handler: h.Admin.Ops.GetDashboardOpenAITokenStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/system-logs", Handler: h.Admin.Ops.ListSystemLogsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/ops/system-logs/cleanup", Handler: h.Admin.Ops.CleanupSystemLogsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/system-logs/health", Handler: h.Admin.Ops.GetSystemLogIngestionHealthGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/alert-rules", Handler: h.Admin.Ops.ListAlertRulesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/ops/alert-rules", Handler: h.Admin.Ops.CreateAlertRuleGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/ops/alert-rules/:id", Handler: h.Admin.Ops.UpdateAlertRuleGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/ops/alert-rules/:id", Handler: h.Admin.Ops.DeleteAlertRuleGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/alert-events", Handler: h.Admin.Ops.ListAlertEventsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/alert-events/:id", Handler: h.Admin.Ops.GetAlertEventGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/ops/alert-events/:id/status", Handler: h.Admin.Ops.UpdateAlertEventStatusGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/ops/alert-silences", Handler: h.Admin.Ops.CreateAlertSilenceGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/errors", Handler: h.Admin.Ops.GetErrorLogsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/errors/:id", Handler: h.Admin.Ops.GetErrorLogByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/errors/:id/retries", Handler: h.Admin.Ops.ListRetryAttemptsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/ops/errors/:id/retry", Handler: h.Admin.Ops.RetryErrorRequestGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/ops/errors/:id/resolve", Handler: h.Admin.Ops.UpdateErrorResolutionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/request-errors", Handler: h.Admin.Ops.ListRequestErrorsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/request-errors/:id", Handler: h.Admin.Ops.GetRequestErrorGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/request-errors/:id/upstream-errors", Handler: h.Admin.Ops.ListRequestErrorUpstreamErrorsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/ops/request-errors/:id/retry-client", Handler: h.Admin.Ops.RetryRequestErrorClientGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/ops/request-errors/:id/upstream-errors/:idx/retry", Handler: h.Admin.Ops.RetryRequestErrorUpstreamEventGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/ops/request-errors/:id/resolve", Handler: h.Admin.Ops.ResolveRequestErrorGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/upstream-errors", Handler: h.Admin.Ops.ListUpstreamErrorsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/upstream-errors/:id", Handler: h.Admin.Ops.GetUpstreamErrorGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/ops/upstream-errors/:id/retry", Handler: h.Admin.Ops.RetryUpstreamErrorGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/ops/upstream-errors/:id/resolve", Handler: h.Admin.Ops.ResolveUpstreamErrorGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/requests", Handler: h.Admin.Ops.ListRequestDetailsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/email-notification/config", Handler: h.Admin.Ops.GetEmailNotificationConfigGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/ops/email-notification/config", Handler: h.Admin.Ops.UpdateEmailNotificationConfigGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/runtime/alert", Handler: h.Admin.Ops.GetAlertRuntimeSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/ops/runtime/alert", Handler: h.Admin.Ops.UpdateAlertRuntimeSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/runtime/logging", Handler: h.Admin.Ops.GetRuntimeLogConfigGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/ops/runtime/logging", Handler: h.Admin.Ops.UpdateRuntimeLogConfigGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/ops/runtime/logging/reset", Handler: h.Admin.Ops.ResetRuntimeLogConfigGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/advanced-settings", Handler: h.Admin.Ops.GetAdvancedSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/ops/advanced-settings", Handler: h.Admin.Ops.UpdateAdvancedSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/ops/settings/metric-thresholds", Handler: h.Admin.Ops.GetMetricThresholdsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/ops/settings/metric-thresholds", Handler: h.Admin.Ops.UpdateMetricThresholdsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.System != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/system/version", Handler: h.Admin.System.GetVersionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/system/check-updates", Handler: h.Admin.System.CheckUpdatesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/system/update", Handler: h.Admin.System.PerformUpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/system/rollback", Handler: h.Admin.System.RollbackGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/system/restart", Handler: h.Admin.System.RestartServiceGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Account != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts", Handler: h.Admin.Account.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/:id", Handler: h.Admin.Account.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts", Handler: h.Admin.Account.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/accounts/:id", Handler: h.Admin.Account.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/accounts/:id", Handler: h.Admin.Account.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/check-mixed-channel", Handler: h.Admin.Account.CheckMixedChannelGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/:id/test", Handler: h.Admin.Account.TestGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/batch-test", Handler: h.Admin.Account.BatchTestGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/:id/refresh", Handler: h.Admin.Account.RefreshGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/:id/stats", Handler: h.Admin.Account.GetStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/:id/clear-error", Handler: h.Admin.Account.ClearErrorGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/:id/usage", Handler: h.Admin.Account.GetUsageGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/:id/clear-rate-limit", Handler: h.Admin.Account.ClearRateLimitGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/:id/reset-quota", Handler: h.Admin.Account.ResetQuotaGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/:id/temp-unschedulable", Handler: h.Admin.Account.GetTempUnschedulableGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/accounts/:id/temp-unschedulable", Handler: h.Admin.Account.ClearTempUnschedulableGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/:id/today-stats", Handler: h.Admin.Account.GetTodayStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/today-stats/batch", Handler: h.Admin.Account.GetBatchTodayStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/:id/schedulable", Handler: h.Admin.Account.SetSchedulableGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/:id/models", Handler: h.Admin.Account.GetAvailableModelsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/:id/recover-state", Handler: h.Admin.Account.RecoverStateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/sync/crs", Handler: h.Admin.Account.SyncFromCRSGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/sync/crs/preview", Handler: h.Admin.Account.PreviewFromCRSGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/batch-clear-error", Handler: h.Admin.Account.BatchClearErrorGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/batch-refresh", Handler: h.Admin.Account.BatchRefreshGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/batch", Handler: h.Admin.Account.BatchCreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/batch-update-credentials", Handler: h.Admin.Account.BatchUpdateCredentialsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/bulk-update", Handler: h.Admin.Account.BulkUpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/:id/models/refresh", Handler: h.Admin.Account.RefreshModelsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/:id/refresh-tier", Handler: h.Admin.Account.RefreshTierGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/batch-refresh-tier", Handler: h.Admin.Account.BatchRefreshTierGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/antigravity/default-model-mapping", Handler: h.Admin.Account.GetAntigravityDefaultModelMappingGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/data", Handler: h.Admin.Account.ExportDataGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/data", Handler: h.Admin.Account.ImportDataGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/data/export-tasks", Handler: h.Admin.Account.CreateExportTaskGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/data/export-tasks/:task_id", Handler: h.Admin.Account.GetExportTaskGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/data/upload-sessions", Handler: h.Admin.Account.CreateImportUploadSessionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/data/upload-sessions/:session_id", Handler: h.Admin.Account.GetImportUploadSessionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/accounts/data/upload-sessions/:session_id", Handler: h.Admin.Account.UploadImportChunkGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/data/upload-sessions/:session_id/finalize", Handler: h.Admin.Account.FinalizeImportUploadSessionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/data/tasks", Handler: h.Admin.Account.CreateImportTaskGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/accounts/data/tasks/:task_id", Handler: h.Admin.Account.GetImportTaskGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.OAuth != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/generate-auth-url", Handler: h.Admin.OAuth.GenerateAuthURLGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/generate-setup-token-url", Handler: h.Admin.OAuth.GenerateSetupTokenURLGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/exchange-code", Handler: h.Admin.OAuth.ExchangeCodeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/exchange-setup-token-code", Handler: h.Admin.OAuth.ExchangeSetupTokenCodeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/cookie-auth", Handler: h.Admin.OAuth.CookieAuthGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/accounts/setup-token-cookie-auth", Handler: h.Admin.OAuth.SetupTokenCookieAuthGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	if h.Admin.Setting != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/settings", Handler: h.Admin.Setting.GetSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/settings", Handler: h.Admin.Setting.UpdateSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/settings/test-smtp", Handler: h.Admin.Setting.TestSMTPConnectionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/settings/send-test-email", Handler: h.Admin.Setting.SendTestEmailGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/settings/admin-api-key", Handler: h.Admin.Setting.GetAdminAPIKeyGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/settings/admin-api-key/regenerate", Handler: h.Admin.Setting.RegenerateAdminAPIKeyGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/settings/admin-api-key", Handler: h.Admin.Setting.DeleteAdminAPIKeyGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/settings/overload-cooldown", Handler: h.Admin.Setting.GetOverloadCooldownSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/settings/overload-cooldown", Handler: h.Admin.Setting.UpdateOverloadCooldownSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/settings/stream-timeout", Handler: h.Admin.Setting.GetStreamTimeoutSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/settings/stream-timeout", Handler: h.Admin.Setting.UpdateStreamTimeoutSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/settings/rectifier", Handler: h.Admin.Setting.GetRectifierSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/settings/rectifier", Handler: h.Admin.Setting.UpdateRectifierSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/settings/beta-policy", Handler: h.Admin.Setting.GetBetaPolicySettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/settings/beta-policy", Handler: h.Admin.Setting.UpdateBetaPolicySettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/settings/tls-fingerprint", Handler: h.Admin.Setting.GetTLSFingerprintSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/settings/tls-fingerprint", Handler: h.Admin.Setting.UpdateTLSFingerprintSettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/settings/tls-fingerprint/profiles", Handler: h.Admin.Setting.ListTLSFingerprintProfilesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/settings/tls-fingerprint/profiles", Handler: h.Admin.Setting.CreateTLSFingerprintProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/settings/tls-fingerprint/profiles/:profile_id", Handler: h.Admin.Setting.UpdateTLSFingerprintProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/settings/tls-fingerprint/profiles/:profile_id", Handler: h.Admin.Setting.DeleteTLSFingerprintProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/settings/sora-s3", Handler: h.Admin.Setting.GetSoraS3SettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/settings/sora-s3", Handler: h.Admin.Setting.UpdateSoraS3SettingsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/settings/sora-s3/test", Handler: h.Admin.Setting.TestSoraS3ConnectionGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/settings/sora-s3/profiles", Handler: h.Admin.Setting.ListSoraS3ProfilesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/settings/sora-s3/profiles", Handler: h.Admin.Setting.CreateSoraS3ProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/settings/sora-s3/profiles/:profile_id", Handler: h.Admin.Setting.UpdateSoraS3ProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/settings/sora-s3/profiles/:profile_id", Handler: h.Admin.Setting.DeleteSoraS3ProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/settings/sora-s3/profiles/:profile_id/activate", Handler: h.Admin.Setting.SetActiveSoraS3ProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "admin_auth"}},
		)
	}
	return out
}

// RegisterAdminRoutes 注册管理员路由
func RegisterAdminRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	adminAuth middleware.AdminAuthMiddleware,
) {
	admin := v1.Group("/admin")
	admin.Use(gin.HandlerFunc(adminAuth))
	{
		// 仪表盘
		registerDashboardRoutes(admin, h)

		// 用户管理
		registerUserManagementRoutes(admin, h)

		// 分组管理
		registerGroupRoutes(admin, h)

		// 账号管理
		registerAccountRoutes(admin, h)

		// 公告管理
		registerAnnouncementRoutes(admin, h)

		// OpenAI OAuth
		registerOpenAIOAuthRoutes(admin, h)
		// Sora OAuth（实现复用 OpenAI OAuth 服务，入口独立）
		registerSoraOAuthRoutes(admin, h)

		// Gemini OAuth
		registerGeminiOAuthRoutes(admin, h)

		// Antigravity OAuth
		registerAntigravityOAuthRoutes(admin, h)

		// 代理管理
		registerProxyRoutes(admin, h)

		// 卡密管理
		registerRedeemCodeRoutes(admin, h)

		// 优惠码管理
		registerPromoCodeRoutes(admin, h)

		// 系统设置
		registerSettingsRoutes(admin, h)

		// 数据管理
		registerDataManagementRoutes(admin, h)

		// 数据库备份恢复
		registerBackupRoutes(admin, h)

		// 运维监控（Ops）
		registerOpsRoutes(admin, h)

		// 系统管理
		registerSystemRoutes(admin, h)

		// 订阅管理
		registerSubscriptionRoutes(admin, h)

		// 使用记录管理
		registerUsageRoutes(admin, h)

		// 用户属性管理
		registerUserAttributeRoutes(admin, h)

		// 错误透传规则管理
		registerErrorPassthroughRoutes(admin, h)

		// API Key 管理
		registerAdminAPIKeyRoutes(admin, h)

		// 定时测试计划
		registerScheduledTestRoutes(admin, h)

		// 代理自动维护计划
		registerProxyMaintenanceRoutes(admin, h)
	}
}

func registerAdminAPIKeyRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	apiKeys := admin.Group("/api-keys")
	{
		apiKeys.PUT("/:id", h.Admin.APIKey.UpdateGroup)
	}
}

func registerOpsRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	ops := admin.Group("/ops")
	{
		// Realtime ops signals
		ops.GET("/concurrency", h.Admin.Ops.GetConcurrencyStats)
		ops.GET("/user-concurrency", h.Admin.Ops.GetUserConcurrencyStats)
		ops.GET("/account-availability", h.Admin.Ops.GetAccountAvailability)
		ops.GET("/realtime-traffic", h.Admin.Ops.GetRealtimeTrafficSummary)
		ops.GET("/gateway-scheduler", h.Admin.Ops.GetGatewaySchedulerRuntime)
		ops.GET("/openai-ws-runtime", h.Admin.Ops.GetOpenAIWSRuntime)

		// Alerts (rules + events)
		ops.GET("/alert-rules", h.Admin.Ops.ListAlertRules)
		ops.POST("/alert-rules", h.Admin.Ops.CreateAlertRule)
		ops.PUT("/alert-rules/:id", h.Admin.Ops.UpdateAlertRule)
		ops.DELETE("/alert-rules/:id", h.Admin.Ops.DeleteAlertRule)
		ops.GET("/alert-events", h.Admin.Ops.ListAlertEvents)
		ops.GET("/alert-events/:id", h.Admin.Ops.GetAlertEvent)
		ops.PUT("/alert-events/:id/status", h.Admin.Ops.UpdateAlertEventStatus)
		ops.POST("/alert-silences", h.Admin.Ops.CreateAlertSilence)

		// Email notification config (DB-backed)
		ops.GET("/email-notification/config", h.Admin.Ops.GetEmailNotificationConfig)
		ops.PUT("/email-notification/config", h.Admin.Ops.UpdateEmailNotificationConfig)

		// Runtime settings (DB-backed)
		runtime := ops.Group("/runtime")
		{
			runtime.GET("/alert", h.Admin.Ops.GetAlertRuntimeSettings)
			runtime.PUT("/alert", h.Admin.Ops.UpdateAlertRuntimeSettings)
			runtime.GET("/logging", h.Admin.Ops.GetRuntimeLogConfig)
			runtime.PUT("/logging", h.Admin.Ops.UpdateRuntimeLogConfig)
			runtime.POST("/logging/reset", h.Admin.Ops.ResetRuntimeLogConfig)
		}

		// Advanced settings (DB-backed)
		ops.GET("/advanced-settings", h.Admin.Ops.GetAdvancedSettings)
		ops.PUT("/advanced-settings", h.Admin.Ops.UpdateAdvancedSettings)

		// Settings group (DB-backed)
		settings := ops.Group("/settings")
		{
			settings.GET("/metric-thresholds", h.Admin.Ops.GetMetricThresholds)
			settings.PUT("/metric-thresholds", h.Admin.Ops.UpdateMetricThresholds)
		}

		// WebSocket realtime (QPS/TPS)
		ws := ops.Group("/ws")
		{
			ws.GET("/qps", h.Admin.Ops.QPSWSHandler)
		}

		// Error logs (legacy)
		ops.GET("/errors", h.Admin.Ops.GetErrorLogs)
		ops.GET("/errors/:id", h.Admin.Ops.GetErrorLogByID)
		ops.GET("/errors/:id/retries", h.Admin.Ops.ListRetryAttempts)
		ops.POST("/errors/:id/retry", h.Admin.Ops.RetryErrorRequest)
		ops.PUT("/errors/:id/resolve", h.Admin.Ops.UpdateErrorResolution)

		// Request errors (client-visible failures)
		ops.GET("/request-errors", h.Admin.Ops.ListRequestErrors)
		ops.GET("/request-errors/:id", h.Admin.Ops.GetRequestError)
		ops.GET("/request-errors/:id/upstream-errors", h.Admin.Ops.ListRequestErrorUpstreamErrors)
		ops.POST("/request-errors/:id/retry-client", h.Admin.Ops.RetryRequestErrorClient)
		ops.POST("/request-errors/:id/upstream-errors/:idx/retry", h.Admin.Ops.RetryRequestErrorUpstreamEvent)
		ops.PUT("/request-errors/:id/resolve", h.Admin.Ops.ResolveRequestError)

		// Upstream errors (independent upstream failures)
		ops.GET("/upstream-errors", h.Admin.Ops.ListUpstreamErrors)
		ops.GET("/upstream-errors/:id", h.Admin.Ops.GetUpstreamError)
		ops.POST("/upstream-errors/:id/retry", h.Admin.Ops.RetryUpstreamError)
		ops.PUT("/upstream-errors/:id/resolve", h.Admin.Ops.ResolveUpstreamError)

		// Request drilldown (success + error)
		ops.GET("/requests", h.Admin.Ops.ListRequestDetails)

		// Indexed system logs
		ops.GET("/system-logs", h.Admin.Ops.ListSystemLogs)
		ops.POST("/system-logs/cleanup", h.Admin.Ops.CleanupSystemLogs)
		ops.GET("/system-logs/health", h.Admin.Ops.GetSystemLogIngestionHealth)

		// Dashboard (vNext - raw path for MVP)
		ops.GET("/dashboard/snapshot-v2", h.Admin.Ops.GetDashboardSnapshotV2)
		ops.GET("/dashboard/overview", h.Admin.Ops.GetDashboardOverview)
		ops.GET("/dashboard/throughput-trend", h.Admin.Ops.GetDashboardThroughputTrend)
		ops.GET("/dashboard/latency-histogram", h.Admin.Ops.GetDashboardLatencyHistogram)
		ops.GET("/dashboard/error-trend", h.Admin.Ops.GetDashboardErrorTrend)
		ops.GET("/dashboard/error-distribution", h.Admin.Ops.GetDashboardErrorDistribution)
		ops.GET("/dashboard/openai-token-stats", h.Admin.Ops.GetDashboardOpenAITokenStats)
	}
}

func registerDashboardRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	dashboard := admin.Group("/dashboard")
	{
		dashboard.GET("/snapshot-v2", h.Admin.Dashboard.GetSnapshotV2)
		dashboard.GET("/stats", h.Admin.Dashboard.GetStats)
		dashboard.GET("/realtime", h.Admin.Dashboard.GetRealtimeMetrics)
		dashboard.GET("/trend", h.Admin.Dashboard.GetUsageTrend)
		dashboard.GET("/models", h.Admin.Dashboard.GetModelStats)
		dashboard.GET("/groups", h.Admin.Dashboard.GetGroupStats)
		dashboard.GET("/api-keys-trend", h.Admin.Dashboard.GetAPIKeyUsageTrend)
		dashboard.GET("/users-trend", h.Admin.Dashboard.GetUserUsageTrend)
		dashboard.GET("/users-ranking", h.Admin.Dashboard.GetUserSpendingRanking)
		dashboard.POST("/users-usage", h.Admin.Dashboard.GetBatchUsersUsage)
		dashboard.POST("/api-keys-usage", h.Admin.Dashboard.GetBatchAPIKeysUsage)
		dashboard.GET("/user-breakdown", h.Admin.Dashboard.GetUserBreakdown)
		dashboard.POST("/aggregation/backfill", h.Admin.Dashboard.BackfillAggregation)
	}
}

func registerUserManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	users := admin.Group("/users")
	{
		users.GET("", h.Admin.User.List)
		users.GET("/:id", h.Admin.User.GetByID)
		users.POST("", h.Admin.User.Create)
		users.PUT("/:id", h.Admin.User.Update)
		users.DELETE("/:id", h.Admin.User.Delete)
		users.POST("/:id/balance", h.Admin.User.UpdateBalance)
		users.GET("/:id/api-keys", h.Admin.User.GetUserAPIKeys)
		users.GET("/:id/usage", h.Admin.User.GetUserUsage)
		users.GET("/:id/balance-history", h.Admin.User.GetBalanceHistory)
		users.POST("/:id/replace-group", h.Admin.User.ReplaceGroup)

		// User attribute values
		users.GET("/:id/attributes", h.Admin.UserAttribute.GetUserAttributes)
		users.PUT("/:id/attributes", h.Admin.UserAttribute.UpdateUserAttributes)
	}
}

func registerGroupRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	groups := admin.Group("/groups")
	{
		groups.GET("", h.Admin.Group.List)
		groups.GET("/all", h.Admin.Group.GetAll)
		groups.GET("/usage-summary", h.Admin.Group.GetUsageSummary)
		groups.GET("/capacity-summary", h.Admin.Group.GetCapacitySummary)
		groups.PUT("/sort-order", h.Admin.Group.UpdateSortOrder)
		groups.GET("/:id", h.Admin.Group.GetByID)
		groups.POST("", h.Admin.Group.Create)
		groups.PUT("/:id", h.Admin.Group.Update)
		groups.DELETE("/:id", h.Admin.Group.Delete)
		groups.GET("/:id/stats", h.Admin.Group.GetStats)
		groups.GET("/:id/rate-multipliers", h.Admin.Group.GetGroupRateMultipliers)
		groups.PUT("/:id/rate-multipliers", h.Admin.Group.BatchSetGroupRateMultipliers)
		groups.DELETE("/:id/rate-multipliers", h.Admin.Group.ClearGroupRateMultipliers)
		groups.GET("/:id/api-keys", h.Admin.Group.GetGroupAPIKeys)
	}
}

func registerAccountRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	accounts := admin.Group("/accounts")
	{
		accounts.GET("", h.Admin.Account.List)
		accounts.GET("/:id", h.Admin.Account.GetByID)
		accounts.POST("", h.Admin.Account.Create)
		accounts.POST("/check-mixed-channel", h.Admin.Account.CheckMixedChannel)
		accounts.POST("/sync/crs", h.Admin.Account.SyncFromCRS)
		accounts.POST("/sync/crs/preview", h.Admin.Account.PreviewFromCRS)
		accounts.PUT("/:id", h.Admin.Account.Update)
		accounts.DELETE("/:id", h.Admin.Account.Delete)
		accounts.POST("/:id/test", h.Admin.Account.Test)
		accounts.POST("/batch-test", h.Admin.Account.BatchTest)
		accounts.POST("/:id/recover-state", h.Admin.Account.RecoverState)
		accounts.POST("/:id/refresh", h.Admin.Account.Refresh)
		accounts.POST("/:id/refresh-tier", h.Admin.Account.RefreshTier)
		accounts.GET("/:id/stats", h.Admin.Account.GetStats)
		accounts.POST("/:id/clear-error", h.Admin.Account.ClearError)
		accounts.GET("/:id/usage", h.Admin.Account.GetUsage)
		accounts.GET("/:id/today-stats", h.Admin.Account.GetTodayStats)
		accounts.POST("/today-stats/batch", h.Admin.Account.GetBatchTodayStats)
		accounts.POST("/:id/clear-rate-limit", h.Admin.Account.ClearRateLimit)
		accounts.POST("/:id/reset-quota", h.Admin.Account.ResetQuota)
		accounts.GET("/:id/temp-unschedulable", h.Admin.Account.GetTempUnschedulable)
		accounts.DELETE("/:id/temp-unschedulable", h.Admin.Account.ClearTempUnschedulable)
		accounts.POST("/:id/schedulable", h.Admin.Account.SetSchedulable)
		accounts.GET("/:id/models", h.Admin.Account.GetAvailableModels)
		accounts.POST("/:id/models/refresh", h.Admin.Account.RefreshModels)
		accounts.POST("/batch", h.Admin.Account.BatchCreate)
		accounts.GET("/data", h.Admin.Account.ExportData)
		accounts.POST("/data", h.Admin.Account.ImportData)
		accounts.POST("/data/export-tasks", h.Admin.Account.CreateExportTask)
		accounts.GET("/data/export-tasks/:task_id", h.Admin.Account.GetExportTask)
		accounts.POST("/data/upload-sessions", h.Admin.Account.CreateImportUploadSession)
		accounts.GET("/data/upload-sessions/:session_id", h.Admin.Account.GetImportUploadSession)
		accounts.PUT("/data/upload-sessions/:session_id", h.Admin.Account.UploadImportChunk)
		accounts.POST("/data/upload-sessions/:session_id/finalize", h.Admin.Account.FinalizeImportUploadSession)
		accounts.POST("/data/tasks", h.Admin.Account.CreateImportTask)
		accounts.GET("/data/tasks/:task_id", h.Admin.Account.GetImportTask)
		accounts.POST("/batch-update-credentials", h.Admin.Account.BatchUpdateCredentials)
		accounts.POST("/batch-refresh-tier", h.Admin.Account.BatchRefreshTier)
		accounts.POST("/bulk-update", h.Admin.Account.BulkUpdate)
		accounts.POST("/batch-clear-error", h.Admin.Account.BatchClearError)
		accounts.POST("/batch-refresh", h.Admin.Account.BatchRefresh)

		// Antigravity 默认模型映射
		accounts.GET("/antigravity/default-model-mapping", h.Admin.Account.GetAntigravityDefaultModelMapping)

		// Claude OAuth routes
		accounts.POST("/generate-auth-url", h.Admin.OAuth.GenerateAuthURL)
		accounts.POST("/generate-setup-token-url", h.Admin.OAuth.GenerateSetupTokenURL)
		accounts.POST("/exchange-code", h.Admin.OAuth.ExchangeCode)
		accounts.POST("/exchange-setup-token-code", h.Admin.OAuth.ExchangeSetupTokenCode)
		accounts.POST("/cookie-auth", h.Admin.OAuth.CookieAuth)
		accounts.POST("/setup-token-cookie-auth", h.Admin.OAuth.SetupTokenCookieAuth)
	}
}

func registerAnnouncementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	announcements := admin.Group("/announcements")
	{
		announcements.GET("", h.Admin.Announcement.List)
		announcements.POST("", h.Admin.Announcement.Create)
		announcements.GET("/:id", h.Admin.Announcement.GetByID)
		announcements.PUT("/:id", h.Admin.Announcement.Update)
		announcements.DELETE("/:id", h.Admin.Announcement.Delete)
		announcements.GET("/:id/read-status", h.Admin.Announcement.ListReadStatus)
	}
}

func registerOpenAIOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	openai := admin.Group("/openai")
	{
		openai.POST("/generate-auth-url", h.Admin.OpenAIOAuth.GenerateAuthURL)
		openai.POST("/exchange-code", h.Admin.OpenAIOAuth.ExchangeCode)
		openai.POST("/refresh-token", h.Admin.OpenAIOAuth.RefreshToken)
		openai.POST("/st2at", h.Admin.OpenAIOAuth.ExchangeOpenAIChatWebSessionToken)
		openai.POST("/at2info", h.Admin.OpenAIOAuth.InspectOpenAIChatWebAccessToken)
		openai.POST("/accounts/:id/refresh", h.Admin.OpenAIOAuth.RefreshAccountToken)
		openai.POST("/create-from-oauth", h.Admin.OpenAIOAuth.CreateAccountFromOAuth)
	}
}

func registerSoraOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	sora := admin.Group("/sora")
	{
		sora.POST("/generate-auth-url", h.Admin.OpenAIOAuth.GenerateAuthURL)
		sora.POST("/exchange-code", h.Admin.OpenAIOAuth.ExchangeCode)
		sora.POST("/refresh-token", h.Admin.OpenAIOAuth.RefreshToken)
		sora.POST("/st2at", h.Admin.OpenAIOAuth.ExchangeSoraSessionToken)
		sora.POST("/rt2at", h.Admin.OpenAIOAuth.RefreshToken)
		sora.POST("/accounts/:id/refresh", h.Admin.OpenAIOAuth.RefreshAccountToken)
		sora.POST("/create-from-oauth", h.Admin.OpenAIOAuth.CreateAccountFromOAuth)
	}
}

func registerGeminiOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	gemini := admin.Group("/gemini")
	{
		gemini.POST("/oauth/auth-url", h.Admin.GeminiOAuth.GenerateAuthURL)
		gemini.POST("/oauth/exchange-code", h.Admin.GeminiOAuth.ExchangeCode)
		gemini.GET("/oauth/capabilities", h.Admin.GeminiOAuth.GetCapabilities)
	}
}

func registerAntigravityOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	antigravity := admin.Group("/antigravity")
	{
		antigravity.POST("/oauth/auth-url", h.Admin.AntigravityOAuth.GenerateAuthURL)
		antigravity.POST("/oauth/exchange-code", h.Admin.AntigravityOAuth.ExchangeCode)
		antigravity.POST("/oauth/refresh-token", h.Admin.AntigravityOAuth.RefreshToken)
	}
}

func registerProxyRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	proxies := admin.Group("/proxies")
	{
		proxies.GET("", h.Admin.Proxy.List)
		proxies.GET("/all", h.Admin.Proxy.GetAll)
		proxies.GET("/data", h.Admin.Proxy.ExportData)
		proxies.POST("/data", h.Admin.Proxy.ImportData)
		proxies.GET("/:id", h.Admin.Proxy.GetByID)
		proxies.POST("", h.Admin.Proxy.Create)
		proxies.PUT("/:id", h.Admin.Proxy.Update)
		proxies.DELETE("/:id", h.Admin.Proxy.Delete)
		proxies.POST("/:id/test", h.Admin.Proxy.Test)
		proxies.POST("/:id/quality-check", h.Admin.Proxy.CheckQuality)
		proxies.GET("/:id/stats", h.Admin.Proxy.GetStats)
		proxies.GET("/:id/accounts", h.Admin.Proxy.GetProxyAccounts)
		proxies.POST("/batch-delete", h.Admin.Proxy.BatchDelete)
		proxies.POST("/batch", h.Admin.Proxy.BatchCreate)
	}
}

func registerRedeemCodeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	codes := admin.Group("/redeem-codes")
	{
		codes.GET("", h.Admin.Redeem.List)
		codes.GET("/stats", h.Admin.Redeem.GetStats)
		codes.GET("/export", h.Admin.Redeem.Export)
		codes.GET("/:id", h.Admin.Redeem.GetByID)
		codes.POST("/create-and-redeem", h.Admin.Redeem.CreateAndRedeem)
		codes.POST("/generate", h.Admin.Redeem.Generate)
		codes.DELETE("/:id", h.Admin.Redeem.Delete)
		codes.POST("/batch-delete", h.Admin.Redeem.BatchDelete)
		codes.POST("/:id/expire", h.Admin.Redeem.Expire)
	}
}

func registerPromoCodeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	promoCodes := admin.Group("/promo-codes")
	{
		promoCodes.GET("", h.Admin.Promo.List)
		promoCodes.GET("/:id", h.Admin.Promo.GetByID)
		promoCodes.POST("", h.Admin.Promo.Create)
		promoCodes.PUT("/:id", h.Admin.Promo.Update)
		promoCodes.DELETE("/:id", h.Admin.Promo.Delete)
		promoCodes.GET("/:id/usages", h.Admin.Promo.GetUsages)
	}
}

func registerSettingsRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	adminSettings := admin.Group("/settings")
	{
		adminSettings.GET("", h.Admin.Setting.GetSettings)
		adminSettings.PUT("", h.Admin.Setting.UpdateSettings)
		adminSettings.POST("/test-smtp", h.Admin.Setting.TestSMTPConnection)
		adminSettings.POST("/send-test-email", h.Admin.Setting.SendTestEmail)
		// Admin API Key 管理
		adminSettings.GET("/admin-api-key", h.Admin.Setting.GetAdminAPIKey)
		adminSettings.POST("/admin-api-key/regenerate", h.Admin.Setting.RegenerateAdminAPIKey)
		adminSettings.DELETE("/admin-api-key", h.Admin.Setting.DeleteAdminAPIKey)
		// 529过载冷却配置
		adminSettings.GET("/overload-cooldown", h.Admin.Setting.GetOverloadCooldownSettings)
		adminSettings.PUT("/overload-cooldown", h.Admin.Setting.UpdateOverloadCooldownSettings)
		// 流超时处理配置
		adminSettings.GET("/stream-timeout", h.Admin.Setting.GetStreamTimeoutSettings)
		adminSettings.PUT("/stream-timeout", h.Admin.Setting.UpdateStreamTimeoutSettings)
		// 请求整流器配置
		adminSettings.GET("/rectifier", h.Admin.Setting.GetRectifierSettings)
		adminSettings.PUT("/rectifier", h.Admin.Setting.UpdateRectifierSettings)
		// Beta 策略配置
		adminSettings.GET("/beta-policy", h.Admin.Setting.GetBetaPolicySettings)
		adminSettings.PUT("/beta-policy", h.Admin.Setting.UpdateBetaPolicySettings)
		// TLS 指纹 Profile 配置
		adminSettings.GET("/tls-fingerprint", h.Admin.Setting.GetTLSFingerprintSettings)
		adminSettings.PUT("/tls-fingerprint", h.Admin.Setting.UpdateTLSFingerprintSettings)
		adminSettings.GET("/tls-fingerprint/profiles", h.Admin.Setting.ListTLSFingerprintProfiles)
		adminSettings.POST("/tls-fingerprint/profiles", h.Admin.Setting.CreateTLSFingerprintProfile)
		adminSettings.PUT("/tls-fingerprint/profiles/:profile_id", h.Admin.Setting.UpdateTLSFingerprintProfile)
		adminSettings.DELETE("/tls-fingerprint/profiles/:profile_id", h.Admin.Setting.DeleteTLSFingerprintProfile)
		// Sora S3 存储配置
		adminSettings.GET("/sora-s3", h.Admin.Setting.GetSoraS3Settings)
		adminSettings.PUT("/sora-s3", h.Admin.Setting.UpdateSoraS3Settings)
		adminSettings.POST("/sora-s3/test", h.Admin.Setting.TestSoraS3Connection)
		adminSettings.GET("/sora-s3/profiles", h.Admin.Setting.ListSoraS3Profiles)
		adminSettings.POST("/sora-s3/profiles", h.Admin.Setting.CreateSoraS3Profile)
		adminSettings.PUT("/sora-s3/profiles/:profile_id", h.Admin.Setting.UpdateSoraS3Profile)
		adminSettings.DELETE("/sora-s3/profiles/:profile_id", h.Admin.Setting.DeleteSoraS3Profile)
		adminSettings.POST("/sora-s3/profiles/:profile_id/activate", h.Admin.Setting.SetActiveSoraS3Profile)
	}
}

func registerDataManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	dataManagement := admin.Group("/data-management")
	{
		dataManagement.GET("/agent/health", h.Admin.DataManagement.GetAgentHealth)
		dataManagement.GET("/config", h.Admin.DataManagement.GetConfig)
		dataManagement.PUT("/config", h.Admin.DataManagement.UpdateConfig)
		dataManagement.GET("/sources/:source_type/profiles", h.Admin.DataManagement.ListSourceProfiles)
		dataManagement.POST("/sources/:source_type/profiles", h.Admin.DataManagement.CreateSourceProfile)
		dataManagement.PUT("/sources/:source_type/profiles/:profile_id", h.Admin.DataManagement.UpdateSourceProfile)
		dataManagement.DELETE("/sources/:source_type/profiles/:profile_id", h.Admin.DataManagement.DeleteSourceProfile)
		dataManagement.POST("/sources/:source_type/profiles/:profile_id/activate", h.Admin.DataManagement.SetActiveSourceProfile)
		dataManagement.POST("/s3/test", h.Admin.DataManagement.TestS3)
		dataManagement.GET("/s3/profiles", h.Admin.DataManagement.ListS3Profiles)
		dataManagement.POST("/s3/profiles", h.Admin.DataManagement.CreateS3Profile)
		dataManagement.PUT("/s3/profiles/:profile_id", h.Admin.DataManagement.UpdateS3Profile)
		dataManagement.DELETE("/s3/profiles/:profile_id", h.Admin.DataManagement.DeleteS3Profile)
		dataManagement.POST("/s3/profiles/:profile_id/activate", h.Admin.DataManagement.SetActiveS3Profile)
		dataManagement.POST("/backups", h.Admin.DataManagement.CreateBackupJob)
		dataManagement.GET("/backups", h.Admin.DataManagement.ListBackupJobs)
		dataManagement.GET("/backups/:job_id", h.Admin.DataManagement.GetBackupJob)
	}
}

func registerBackupRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	backup := admin.Group("/backups")
	{
		// S3 存储配置
		backup.GET("/s3-config", h.Admin.Backup.GetS3Config)
		backup.PUT("/s3-config", h.Admin.Backup.UpdateS3Config)
		backup.POST("/s3-config/test", h.Admin.Backup.TestS3Connection)

		// 定时备份配置
		backup.GET("/schedule", h.Admin.Backup.GetSchedule)
		backup.PUT("/schedule", h.Admin.Backup.UpdateSchedule)

		// 备份操作
		backup.POST("", h.Admin.Backup.CreateBackup)
		backup.GET("", h.Admin.Backup.ListBackups)
		backup.GET("/:id", h.Admin.Backup.GetBackup)
		backup.DELETE("/:id", h.Admin.Backup.DeleteBackup)
		backup.GET("/:id/download-url", h.Admin.Backup.GetDownloadURL)

		// 恢复操作
		backup.POST("/:id/restore", h.Admin.Backup.RestoreBackup)
	}
}

func registerSystemRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	system := admin.Group("/system")
	{
		system.GET("/version", h.Admin.System.GetVersion)
		system.GET("/check-updates", h.Admin.System.CheckUpdates)
		system.POST("/update", h.Admin.System.PerformUpdate)
		system.POST("/rollback", h.Admin.System.Rollback)
		system.POST("/restart", h.Admin.System.RestartService)
	}
}

func registerSubscriptionRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	subscriptions := admin.Group("/subscriptions")
	{
		subscriptions.GET("", h.Admin.Subscription.List)
		subscriptions.GET("/:id", h.Admin.Subscription.GetByID)
		subscriptions.GET("/:id/progress", h.Admin.Subscription.GetProgress)
		subscriptions.POST("/assign", h.Admin.Subscription.Assign)
		subscriptions.POST("/bulk-assign", h.Admin.Subscription.BulkAssign)
		subscriptions.POST("/:id/extend", h.Admin.Subscription.Extend)
		subscriptions.POST("/:id/reset-quota", h.Admin.Subscription.ResetQuota)
		subscriptions.DELETE("/:id", h.Admin.Subscription.Revoke)
	}

	// 分组下的订阅列表
	admin.GET("/groups/:id/subscriptions", h.Admin.Subscription.ListByGroup)

	// 用户下的订阅列表
	admin.GET("/users/:id/subscriptions", h.Admin.Subscription.ListByUser)
}

func registerUsageRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	usage := admin.Group("/usage")
	{
		usage.GET("", h.Admin.Usage.List)
		usage.GET("/stats", h.Admin.Usage.Stats)
		usage.GET("/search-users", h.Admin.Usage.SearchUsers)
		usage.GET("/search-api-keys", h.Admin.Usage.SearchAPIKeys)
		usage.GET("/cleanup-tasks", h.Admin.Usage.ListCleanupTasks)
		usage.POST("/cleanup-tasks", h.Admin.Usage.CreateCleanupTask)
		usage.POST("/cleanup-tasks/:id/cancel", h.Admin.Usage.CancelCleanupTask)
	}
}

func registerUserAttributeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	attrs := admin.Group("/user-attributes")
	{
		attrs.GET("", h.Admin.UserAttribute.ListDefinitions)
		attrs.POST("", h.Admin.UserAttribute.CreateDefinition)
		attrs.POST("/batch", h.Admin.UserAttribute.GetBatchUserAttributes)
		attrs.PUT("/reorder", h.Admin.UserAttribute.ReorderDefinitions)
		attrs.PUT("/:id", h.Admin.UserAttribute.UpdateDefinition)
		attrs.DELETE("/:id", h.Admin.UserAttribute.DeleteDefinition)
	}
}

func registerScheduledTestRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	plans := admin.Group("/scheduled-test-plans")
	{
		plans.POST("", h.Admin.ScheduledTest.Create)
		plans.PUT("/:id", h.Admin.ScheduledTest.Update)
		plans.DELETE("/:id", h.Admin.ScheduledTest.Delete)
		plans.GET("/:id/results", h.Admin.ScheduledTest.ListResults)
	}
	// Nested under accounts
	admin.GET("/accounts/:id/scheduled-test-plans", h.Admin.ScheduledTest.ListByAccount)
}

func registerProxyMaintenanceRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	plans := admin.Group("/proxy-maintenance-plans")
	{
		plans.GET("", h.Admin.ProxyMaintenance.List)
		plans.POST("", h.Admin.ProxyMaintenance.Create)
		plans.PUT("/:id", h.Admin.ProxyMaintenance.Update)
		plans.DELETE("/:id", h.Admin.ProxyMaintenance.Delete)
		plans.GET("/:id/results", h.Admin.ProxyMaintenance.ListResults)
	}
	admin.POST("/proxy-maintenance/run-now", h.Admin.ProxyMaintenance.RunNow)
	admin.GET("/proxy-maintenance/tasks/:task_id", h.Admin.ProxyMaintenance.GetTask)
}

func registerErrorPassthroughRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	rules := admin.Group("/error-passthrough-rules")
	{
		rules.GET("", h.Admin.ErrorPassthrough.List)
		rules.GET("/:id", h.Admin.ErrorPassthrough.GetByID)
		rules.POST("", h.Admin.ErrorPassthrough.Create)
		rules.PUT("/:id", h.Admin.ErrorPassthrough.Update)
		rules.DELETE("/:id", h.Admin.ErrorPassthrough.Delete)
	}
}
