package server

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type RouteTransportHints struct {
	Streaming bool `json:"streaming,omitempty"`
	WebSocket bool `json:"websocket,omitempty"`
}

type RouteManifestEntry struct {
	Method     string              `json:"method"`
	Path       string              `json:"path"`
	Handler    string              `json:"handler"`
	Hints      RouteTransportHints `json:"hints,omitempty"`
	Middleware []string            `json:"middleware,omitempty"`
	Executable bool                `json:"executable,omitempty"`
}

type RouteManifest []RouteManifestEntry

func BuildRouteManifest(router *gin.Engine) RouteManifest {
	if router == nil {
		return nil
	}

	routes := router.Routes()
	manifest := make(RouteManifest, 0, len(routes))
	for _, route := range routes {
		entry := RouteManifestEntry{
			Method:  strings.ToUpper(strings.TrimSpace(route.Method)),
			Path:    strings.TrimSpace(route.Path),
			Handler: strings.TrimSpace(route.Handler),
		}
		entry.Hints = inferRouteTransportHints(entry)
		manifest = append(manifest, entry)
	}
	runtimeCfg := buildExecutableRuntimeConfig(nil, nil, nil, nil, nil)
	overlayExecutableRouteMetadata(manifest, runtimeCfg.routes)

	sort.Slice(manifest, func(i, j int) bool {
		if manifest[i].Path == manifest[j].Path {
			return manifest[i].Method < manifest[j].Method
		}
		return manifest[i].Path < manifest[j].Path
	})
	return manifest
}

func overlayExecutableRouteMetadata(manifest RouteManifest, defs []executableRoute) {
	if len(manifest) == 0 || len(defs) == 0 {
		return
	}
	for i := range manifest {
		for _, def := range defs {
			if manifest[i].Method == def.method && manifest[i].Path == def.path {
				manifest[i].Executable = true
				if len(def.middleware) > 0 {
					manifest[i].Middleware = append([]string(nil), def.middleware...)
				}
				break
			}
		}
	}
}

func inferRouteTransportHints(entry RouteManifestEntry) RouteTransportHints {
	lowerPath := strings.ToLower(entry.Path)
	lowerHandler := strings.ToLower(entry.Handler)
	return RouteTransportHints{
		Streaming: strings.Contains(lowerPath, "responses") ||
			strings.Contains(lowerPath, "stream") ||
			strings.Contains(lowerHandler, "stream") ||
			strings.Contains(lowerHandler, "sse"),
		WebSocket: strings.Contains(lowerPath, "responses") && strings.EqualFold(entry.Method, "GET") ||
			strings.Contains(lowerPath, "/ws") ||
			strings.Contains(lowerHandler, "websocket") ||
			strings.Contains(lowerHandler, ".responseswebsocket"),
	}
}

func cloneRouteManifest(in RouteManifest) RouteManifest {
	if len(in) == 0 {
		return nil
	}
	out := make(RouteManifest, len(in))
	copy(out, in)
	return out
}
