package server

import "net/http"

// BuildExecutablePreferredHandler builds a handler for internal sidecar ingress:
// it prefers the executable route pipeline and falls back to the original base
// handler only when no executable route matches.
func BuildExecutablePreferredHandler(base *http.Server) http.Handler {
	if base == nil || base.Handler == nil {
		return http.NewServeMux()
	}
	execRuntime := executableRuntimeForHTTPServer(base)
	if execRuntime == nil {
		return base.Handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dispatchExecutableRoute(execRuntime, r, w, "") {
			return
		}
		base.Handler.ServeHTTP(w, r)
	})
}
