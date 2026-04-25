package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

func ExecutablePaymentRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil {
		return nil
	}

	userMW := []string{
		"request_logger",
		"cors",
		"security_headers",
		"client_request_id",
		"jwt_auth",
		"backend_mode_user_guard",
	}
	publicMW := []string{
		"request_logger",
		"cors",
		"security_headers",
		"client_request_id",
	}
	adminMW := []string{
		"request_logger",
		"cors",
		"security_headers",
		"client_request_id",
		"admin_auth",
	}

	defs := make([]gatewayctx.RouteDef, 0, 32)
	if h.Payment != nil {
		defs = append(defs,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/payment/config", Handler: h.Payment.GetPaymentConfigGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/payment/checkout-info", Handler: h.Payment.GetCheckoutInfoGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/payment/plans", Handler: h.Payment.GetPlansGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/payment/channels", Handler: h.Payment.GetChannelsGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/payment/limits", Handler: h.Payment.GetLimitsGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/payment/orders", Handler: h.Payment.CreateOrderGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/payment/orders/verify", Handler: h.Payment.VerifyOrderGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/payment/orders/my", Handler: h.Payment.GetMyOrdersGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/payment/orders/refund-eligible-providers", Handler: h.Payment.GetRefundEligibleProvidersGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/payment/orders/:id", Handler: h.Payment.GetOrderGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/payment/orders/:id/cancel", Handler: h.Payment.CancelOrderGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/payment/orders/:id/refund-request", Handler: h.Payment.RequestRefundGateway, Middleware: userMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/payment/public/orders/verify", Handler: h.Payment.VerifyOrderPublicGateway, Middleware: publicMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/payment/public/orders/resolve", Handler: h.Payment.ResolveOrderPublicByResumeTokenGateway, Middleware: publicMW},
		)
	}

	if h.PaymentWebhook != nil {
		defs = append(defs,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/payment/webhook/easypay", Handler: h.PaymentWebhook.EasyPayNotifyGateway, Middleware: publicMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/payment/webhook/easypay", Handler: h.PaymentWebhook.EasyPayNotifyGateway, Middleware: publicMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/payment/webhook/alipay", Handler: h.PaymentWebhook.AlipayNotifyGateway, Middleware: publicMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/payment/webhook/wxpay", Handler: h.PaymentWebhook.WxpayNotifyGateway, Middleware: publicMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/payment/webhook/stripe", Handler: h.PaymentWebhook.StripeWebhookGateway, Middleware: publicMW},
		)
	}

	if h.Admin != nil && h.Admin.Payment != nil {
		defs = append(defs,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/payment/dashboard", Handler: h.Admin.Payment.GetDashboardGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/payment/config", Handler: h.Admin.Payment.GetConfigGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/payment/config", Handler: h.Admin.Payment.UpdateConfigGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/payment/orders", Handler: h.Admin.Payment.ListOrdersGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/payment/orders/:id", Handler: h.Admin.Payment.GetOrderDetailGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/payment/orders/:id/cancel", Handler: h.Admin.Payment.CancelOrderGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/payment/orders/:id/retry", Handler: h.Admin.Payment.RetryFulfillmentGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/payment/orders/:id/refund", Handler: h.Admin.Payment.ProcessRefundGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/payment/plans", Handler: h.Admin.Payment.ListPlansGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/payment/plans", Handler: h.Admin.Payment.CreatePlanGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/payment/plans/:id", Handler: h.Admin.Payment.UpdatePlanGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/payment/plans/:id", Handler: h.Admin.Payment.DeletePlanGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/admin/payment/providers", Handler: h.Admin.Payment.ListProvidersGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/admin/payment/providers", Handler: h.Admin.Payment.CreateProviderGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/admin/payment/providers/:id", Handler: h.Admin.Payment.UpdateProviderGateway, Middleware: adminMW},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/admin/payment/providers/:id", Handler: h.Admin.Payment.DeleteProviderGateway, Middleware: adminMW},
		)
	}

	return defs
}
