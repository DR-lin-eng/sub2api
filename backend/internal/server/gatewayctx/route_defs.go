package gatewayctx

type RouteDef struct {
	Method     string
	Path       string
	Handler    HandlerFunc
	Middleware []string
}
