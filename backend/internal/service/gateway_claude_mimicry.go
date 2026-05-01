package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

type claudeMimicMetadataBuilder func(fp *Fingerprint, body []byte) string

func (s *GatewayService) applyClaudeCodeOAuthMimicryToParsedRequestBody(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	parsed *ParsedRequest,
	body []byte,
	model string,
) ([]byte, string) {
	var systemRaw any
	var metadataBuilder claudeMimicMetadataBuilder
	if parsed != nil {
		systemRaw = parsed.System
		metadataBuilder = func(fp *Fingerprint, _ []byte) string {
			return s.buildOAuthMetadataUserID(parsed, account, fp)
		}
	}
	return s.applyClaudeCodeOAuthMimicryToBodyContext(ctx, c, account, body, systemRaw, model, metadataBuilder)
}

// applyClaudeCodeOAuthMimicryToBodyContext applies the full Claude Code mimicry
// pipeline for non-Claude-Code clients using Anthropic OAuth accounts.
func (s *GatewayService) applyClaudeCodeOAuthMimicryToBodyContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	systemRaw any,
	model string,
	metadataBuilder claudeMimicMetadataBuilder,
) ([]byte, string) {
	if account == nil || !account.IsOAuth() || len(body) == 0 {
		return body, model
	}

	systemRewritten := false
	if !strings.Contains(strings.ToLower(model), "haiku") && !systemIncludesClaudeCodePrompt(systemRaw) {
		body = injectClaudeCodePrompt(body, systemRaw)
		systemRewritten = true
	}

	normalizeOpts := claudeOAuthNormalizeOptions{stripSystemCacheControl: !systemRewritten}
	var fp *Fingerprint
	if s.identityService != nil {
		requestHeaders := http.Header{}
		if c != nil {
			if req := c.Request(); req != nil {
				requestHeaders = req.Header
			}
		}
		if got, err := s.identityService.GetOrCreateFingerprint(ctx, account.ID, requestHeaders); err == nil {
			fp = got
		}
	}
	metadataUserID := ""
	if metadataBuilder != nil {
		metadataUserID = metadataBuilder(fp, body)
	}
	if metadataUserID == "" {
		metadataUserID = s.buildOAuthMetadataUserIDFromBody(ctx, account, fp, body)
	}
	if metadataUserID != "" {
		normalizeOpts.injectMetadata = true
		normalizeOpts.metadataUserID = metadataUserID
	}

	body, model = normalizeClaudeOAuthRequestBody(body, model, normalizeOpts)

	body = stripMessageCacheControl(body)
	body = addMessageCacheBreakpoints(body)

	if rw := buildToolNameRewriteFromBody(body); rw != nil {
		body = applyToolNameRewriteToBody(body, rw)
		if c != nil {
			c.SetValue(toolNameRewriteKey, rw)
		}
	} else {
		body = applyToolsLastCacheBreakpoint(body)
	}

	return body, model
}

func (s *GatewayService) buildOAuthMetadataUserIDFromBody(
	ctx context.Context,
	account *Account,
	fp *Fingerprint,
	body []byte,
) string {
	_ = ctx
	if account == nil {
		return ""
	}
	if existing := strings.TrimSpace(gjson.GetBytes(body, "metadata.user_id").String()); existing != "" {
		return ""
	}

	userID := strings.TrimSpace(account.GetClaudeUserID())
	if userID == "" && fp != nil {
		userID = strings.TrimSpace(fp.ClientID)
	}
	if userID == "" {
		userID = generateClientID()
	}

	sessionID := uuid.NewString()
	if hash := hashBodyForSessionSeed(body); hash != "" {
		sessionID = generateSessionUUID(fmt.Sprintf("%d::%s", account.ID, hash))
	}

	uaVersion := ""
	if fp != nil {
		uaVersion = ExtractCLIVersion(fp.UserAgent)
	}
	accountUUID := strings.TrimSpace(account.GetExtraString("account_uuid"))
	return FormatMetadataUserID(userID, accountUUID, sessionID, uaVersion)
}

func hashBodyForSessionSeed(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%x", sum[:16])
}
