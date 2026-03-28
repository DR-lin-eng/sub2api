package middleware

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

var corsWarningOnce sync.Once

type CORSRuntimeConfig struct {
	allowAll         bool
	allowCredentials bool
	allowedSet       map[string]struct{}
	allowHeaders     string
}

// CORS 跨域中间件
func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	runtimeCfg := PrepareCORSRuntimeConfig(cfg)
	return func(c *gin.Context) {
		if ApplyCORSContext(gatewayctx.FromGin(c), runtimeCfg) {
			c.Next()
		}
	}
}

func PrepareCORSRuntimeConfig(cfg config.CORSConfig) CORSRuntimeConfig {
	allowedOrigins := normalizeOrigins(cfg.AllowedOrigins)
	allowAll := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
	}
	wildcardWithSpecific := allowAll && len(allowedOrigins) > 1
	if wildcardWithSpecific {
		allowedOrigins = []string{"*"}
	}
	allowCredentials := cfg.AllowCredentials

	corsWarningOnce.Do(func() {
		if len(allowedOrigins) == 0 {
			log.Println("Warning: CORS allowed_origins not configured; cross-origin requests will be rejected.")
		}
		if wildcardWithSpecific {
			log.Println("Warning: CORS allowed_origins includes '*'; wildcard will take precedence over explicit origins.")
		}
		if allowAll && allowCredentials {
			log.Println("Warning: CORS allowed_origins set to '*', disabling allow_credentials.")
		}
	})
	if allowAll && allowCredentials {
		allowCredentials = false
	}

	allowedSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "" || origin == "*" {
			continue
		}
		allowedSet[origin] = struct{}{}
	}
	allowHeaders := []string{
		"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization",
		"accept", "origin", "Cache-Control", "X-Requested-With", "X-API-Key",
	}
	// OpenAI Node SDK 会发送 x-stainless-* 请求头，需在 CORS 中显式放行。
	openAIProperties := []string{
		"lang", "package-version", "os", "arch", "retry-count", "runtime",
		"runtime-version", "async", "helper-method", "poll-helper", "custom-poll-interval", "timeout",
	}
	for _, prop := range openAIProperties {
		allowHeaders = append(allowHeaders, "x-stainless-"+prop)
	}
	allowHeadersValue := strings.Join(allowHeaders, ", ")
	return CORSRuntimeConfig{
		allowAll:         allowAll,
		allowCredentials: allowCredentials,
		allowedSet:       allowedSet,
		allowHeaders:     allowHeadersValue,
	}
}

func ApplyCORSContext(c gatewayctx.GatewayContext, cfg CORSRuntimeConfig) bool {
	if c == nil {
		return false
	}
	origin := strings.TrimSpace(c.HeaderValue("Origin"))
	originAllowed := cfg.allowAll
	if origin != "" && !cfg.allowAll {
		_, originAllowed = cfg.allowedSet[origin]
	}

	headers := c.Header()
	if originAllowed {
		if cfg.allowAll {
			headers.Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			headers.Set("Access-Control-Allow-Origin", origin)
			headers.Add("Vary", "Origin")
		}
		if cfg.allowCredentials {
			headers.Set("Access-Control-Allow-Credentials", "true")
		}
		headers.Set("Access-Control-Allow-Headers", cfg.allowHeaders)
		headers.Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		headers.Set("Access-Control-Expose-Headers", "ETag")
		headers.Set("Access-Control-Max-Age", "86400")
	}

	if strings.EqualFold(c.Method(), http.MethodOptions) {
		if originAllowed {
			c.Abort()
			c.SetStatus(http.StatusNoContent)
		} else {
			c.Abort()
			c.SetStatus(http.StatusForbidden)
		}
		return false
	}

	return true
}

func normalizeOrigins(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
