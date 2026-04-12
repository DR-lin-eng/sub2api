package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/google/uuid"
)

const (
	openAIChatWebBaseURL                  = "https://chatgpt.com"
	openAIChatWebSentinelSDKURLDefault    = "https://sentinel.openai.com/sentinel/20260219f9f6/sdk.js"
	openAIChatWebClientBuildNumberDefault = "5674830"
	openAIChatWebClientVersionDefault     = "prod-6ee9d1a31e859a475cea92af39e34971bf5582c6"
	openAIChatWebLanguageDefault          = "zh-CN"
	openAIChatWebLanguagesJoinDefault     = "zh-CN,zh"
)

type openAIChatWebSentinelBundle struct {
	ProofToken            string
	ChatRequirementsToken string
	TurnstileToken        string
	SOToken               string
	PrepareToken          string
	ExtraData             string
	EchoLogs              string
}

type openAIChatWebConversationPrepareResult struct {
	ConversationID string
	Referer        string
	TurnTraceID    string
	ConduitToken   string
	Payload        map[string]any
	Sentinel       *openAIChatWebSentinelBundle
	Session        openAIChatWebSentinelSession
	SessionID      string
}

func buildOpenAIChatWebHeaders(
	targetPath string,
	token string,
	deviceID string,
	userAgent string,
	sessionID string,
	chatgptAccountID string,
	referer string,
	accept string,
) http.Header {
	headers := http.Header{}
	headers.Set("accept", accept)
	headers.Set("accept-language", openAIChatWebLanguagesJoinDefault+";q=0.9")
	headers.Set("accept-encoding", "gzip, deflate, br, zstd")
	headers.Set("user-agent", userAgent)
	headers.Set("sec-ch-ua", `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`)
	headers.Set("sec-ch-ua-mobile", "?0")
	headers.Set("sec-ch-ua-platform", `"macOS"`)
	headers.Set("sec-fetch-dest", "empty")
	headers.Set("sec-fetch-mode", "cors")
	headers.Set("sec-fetch-site", "same-origin")
	headers.Set("origin", openAIChatWebBaseURL)
	if strings.TrimSpace(referer) == "" {
		referer = openAIChatWebBaseURL + "/"
	}
	headers.Set("referer", referer)
	headers.Set("priority", "u=1, i")
	headers.Set("oai-language", openAIChatWebLanguageDefault)
	headers.Set("oai-device-id", deviceID)
	headers.Set("oai-client-build-number", openAIChatWebClientBuildNumberDefault)
	headers.Set("oai-client-version", openAIChatWebClientVersionDefault)
	headers.Set("x-openai-target-path", targetPath)
	headers.Set("x-openai-target-route", targetPath)
	headers.Set("authorization", "Bearer "+token)
	if strings.TrimSpace(sessionID) != "" {
		headers.Set("oai-session-id", sessionID)
	}
	if strings.TrimSpace(chatgptAccountID) != "" {
		headers.Set("chatgpt-account-id", chatgptAccountID)
	}
	return headers
}

func (s *OpenAIGatewayService) buildOpenAIChatWebSentinelSession(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
) openAIChatWebSentinelSession {
	userAgent := strings.TrimSpace(chatGPTWebUserAgent)
	if value := strings.TrimSpace(account.GetOpenAIUserAgent()); value != "" {
		userAgent = value
	} else if c != nil {
		if value := strings.TrimSpace(c.HeaderValue("User-Agent")); value != "" {
			userAgent = value
		}
	}

	deviceID := ""
	if c != nil {
		deviceID = strings.TrimSpace(c.HeaderValue("oai-device-id"))
	}
	if deviceID == "" {
		deviceID = strings.TrimSpace(account.GetCredential("device_id"))
	}
	if deviceID == "" {
		deviceID = uuid.NewString()
	}

	return openAIChatWebSentinelSession{
		DeviceID:            deviceID,
		UserAgent:           userAgent,
		ScreenWidth:         1512,
		ScreenHeight:        1457,
		HeapLimit:           4294967296,
		HardwareConcurrency: 10,
		Language:            openAIChatWebLanguageDefault,
		LanguagesJoin:       openAIChatWebLanguagesJoinDefault,
		Persona: openAIChatWebSentinelPersona{
			Platform:              "MacIntel",
			Vendor:                "Google Inc.",
			TimezoneOffsetMin:     -480,
			RequirementsScriptURL: openAIChatWebSentinelSDKURLDefault,
			NavigatorProbe:        "sendBeacon−function sendBeacon() { [native code] }",
			DocumentProbe:         "location",
			WindowProbe:           "__oai_so_hi",
		},
	}
}

func openAIChatWebFindString(payload any, keys map[string]struct{}) string {
	switch value := payload.(type) {
	case map[string]any:
		for key, child := range value {
			if _, ok := keys[strings.ToLower(strings.TrimSpace(key))]; ok {
				if text, ok := child.(string); ok && strings.TrimSpace(text) != "" {
					return strings.TrimSpace(text)
				}
			}
		}
		for _, child := range value {
			if result := openAIChatWebFindString(child, keys); result != "" {
				return result
			}
		}
	case []any:
		for _, child := range value {
			if result := openAIChatWebFindString(child, keys); result != "" {
				return result
			}
		}
	}
	return ""
}

func openAIChatWebFindMap(payload any, keys map[string]struct{}) map[string]any {
	switch value := payload.(type) {
	case map[string]any:
		for key, child := range value {
			if _, ok := keys[strings.ToLower(strings.TrimSpace(key))]; ok {
				if object, ok := child.(map[string]any); ok {
					return object
				}
			}
		}
		for _, child := range value {
			if result := openAIChatWebFindMap(child, keys); result != nil {
				return result
			}
		}
	case []any:
		for _, child := range value {
			if result := openAIChatWebFindMap(child, keys); result != nil {
				return result
			}
		}
	}
	return nil
}

func openAIChatWebJSONStringify(v any) string {
	if v == nil {
		return ""
	}
	if text, ok := v.(string); ok {
		return text
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(raw)
}

func openAIChatWebExtractTextContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var parts []apicompat.ResponsesContentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		var builder strings.Builder
		for _, part := range parts {
			switch part.Type {
			case "input_text", "output_text":
				builder.WriteString(part.Text)
			case "input_image":
				if builder.Len() > 0 {
					builder.WriteByte('\n')
				}
				builder.WriteString("[image]")
			}
		}
		return builder.String()
	}
	return string(raw)
}

func openAIChatWebConvertInputToMessages(input json.RawMessage) ([]map[string]any, error) {
	if len(input) == 0 {
		return nil, nil
	}
	var text string
	if err := json.Unmarshal(input, &text); err == nil {
		return []map[string]any{
			{
				"id": uuid.NewString(),
				"author": map[string]any{
					"role": "user",
				},
				"content": map[string]any{
					"content_type": "text",
					"parts":        []string{text},
				},
				"metadata": map[string]any{},
			},
		}, nil
	}

	var items []apicompat.ResponsesInputItem
	if err := json.Unmarshal(input, &items); err != nil {
		return nil, fmt.Errorf("parse chatweb responses input: %w", err)
	}

	messages := make([]map[string]any, 0, len(items))
	for _, item := range items {
		role := strings.TrimSpace(item.Role)
		switch {
		case role != "":
			text := openAIChatWebExtractTextContent(item.Content)
			messages = append(messages, map[string]any{
				"id": uuid.NewString(),
				"author": map[string]any{
					"role": role,
				},
				"content": map[string]any{
					"content_type": "text",
					"parts":        []string{text},
				},
				"metadata": map[string]any{},
			})
		case item.Type == "function_call":
			messages = append(messages, map[string]any{
				"id": uuid.NewString(),
				"author": map[string]any{
					"role": "assistant",
				},
				"content": map[string]any{
					"content_type": "text",
					"parts":        []string{""},
				},
				"metadata": map[string]any{
					"tool_call_id": item.CallID,
					"tool_name":    item.Name,
					"tool_args":    item.Arguments,
				},
			})
		case item.Type == "function_call_output":
			messages = append(messages, map[string]any{
				"id": uuid.NewString(),
				"author": map[string]any{
					"role": "tool",
				},
				"content": map[string]any{
					"content_type": "text",
					"parts":        []string{item.Output},
				},
				"metadata": map[string]any{
					"tool_call_id": item.CallID,
				},
			})
		}
	}
	return messages, nil
}

func openAIChatWebBuildConversationPayload(body []byte) (map[string]any, error) {
	var req apicompat.ResponsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("parse chatweb request body: %w", err)
	}
	messages, err := openAIChatWebConvertInputToMessages(req.Input)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"action":                        "next",
		"conversation_mode":             map[string]any{"kind": "primary_assistant"},
		"force_paragen":                 false,
		"force_rate_limit":              false,
		"history_and_training_disabled": false,
		"messages":                      messages,
		"model":                         strings.TrimSpace(req.Model),
		"parent_message_id":             uuid.NewString(),
		"supports_buffering":            true,
		"timezone_offset_min":           -480,
		"websocket_request_id":          uuid.NewString(),
	}
	if payload["model"] == "" {
		payload["model"] = "auto"
	}

	var original map[string]any
	if err := json.Unmarshal(body, &original); err == nil {
		for _, key := range []string{
			"tools",
			"tool_choice",
			"include",
			"store",
			"reasoning",
			"temperature",
			"top_p",
			"service_tier",
			"max_output_tokens",
			"previous_response_id",
			"instructions",
			"text",
		} {
			if value, ok := original[key]; ok {
				payload[key] = value
			}
		}
	}

	return payload, nil
}

func (s *OpenAIGatewayService) doOpenAIChatWebJSONRequest(
	ctx context.Context,
	account *Account,
	method string,
	url string,
	headers http.Header,
	body []byte,
) (map[string]any, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Host = "chatgpt.com"
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if req.Header.Get("content-type") == "" && len(body) > 0 {
		req.Header.Set("content-type", "application/json")
	}

	proxyURL := ""
	if account != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Header, fmt.Errorf("chatweb request failed: status=%d body=%s", resp.StatusCode, truncateString(string(respBody), 512))
	}

	var payload map[string]any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &payload); err != nil {
			return nil, resp.Header, fmt.Errorf("decode chatweb response: %w", err)
		}
	}
	return payload, resp.Header, nil
}

func (s *OpenAIGatewayService) prepareOpenAIChatWebSentinel(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	token string,
	session openAIChatWebSentinelSession,
	sessionID string,
	referer string,
) (*openAIChatWebSentinelBundle, error) {
	helper := newOpenAIChatWebSentinelHelperRunner()
	requirementsToken, err := helper.requirementsToken(ctx, session)
	if err != nil {
		return nil, err
	}

	headers := buildOpenAIChatWebHeaders(
		"/backend-api/sentinel/chat-requirements/prepare",
		token,
		session.DeviceID,
		session.UserAgent,
		sessionID,
		account.GetChatGPTAccountID(),
		referer,
		"*/*",
	)
	headers.Set("content-type", "application/json")
	payload := map[string]any{
		"p":                             requirementsToken,
		"conversation_mode_kind":        "primary_assistant",
		"history_and_training_disabled": false,
		"model":                         "auto",
		"supported_encodings":           []string{"v1"},
	}

	raw, _ := json.Marshal(payload)
	preparePayload, _, err := s.doOpenAIChatWebJSONRequest(
		ctx,
		account,
		http.MethodPost,
		openAIChatWebBaseURL+"/backend-api/sentinel/chat-requirements/prepare",
		headers,
		raw,
	)
	if err != nil {
		return nil, err
	}

	powPayload := openAIChatWebFindMap(preparePayload, map[string]struct{}{"proofofwork": {}})
	required := false
	seed := ""
	difficulty := ""
	if powPayload != nil {
		if value, ok := powPayload["required"].(bool); ok {
			required = value
		}
		seed, _ = powPayload["seed"].(string)
		difficulty, _ = powPayload["difficulty"].(string)
	}
	proofToken, err := helper.enforcementToken(ctx, session, required, seed, difficulty)
	if err != nil {
		return nil, err
	}

	bundle := &openAIChatWebSentinelBundle{
		ProofToken:            proofToken,
		ChatRequirementsToken: openAIChatWebFindString(preparePayload, map[string]struct{}{"chatrequirementstoken": {}, "openaisentinelchatrequirementstoken": {}, "requirementstoken": {}, "token": {}}),
		TurnstileToken:        openAIChatWebFindString(preparePayload, map[string]struct{}{"turnstiletoken": {}, "openaisentinelturnstiletoken": {}}),
		SOToken:               openAIChatWebFindString(preparePayload, map[string]struct{}{"sotoken": {}, "openaisentinelsotoken": {}}),
		PrepareToken:          requirementsToken,
		ExtraData:             openAIChatWebFindString(preparePayload, map[string]struct{}{"extradata": {}, "sentinelextradata": {}, "openaisentinelextradata": {}}),
		EchoLogs:              openAIChatWebFindString(preparePayload, map[string]struct{}{"echologs": {}, "oaiechologs": {}}),
	}

	turnstilePayload := openAIChatWebFindMap(preparePayload, map[string]struct{}{"turnstile": {}})
	if turnstilePayload != nil && bundle.TurnstileToken == "" {
		dx, _ := turnstilePayload["dx"].(string)
		if strings.TrimSpace(dx) != "" {
			bundle.TurnstileToken, _ = helper.solveTurnstile(ctx, session, requirementsToken, dx)
		}
	}

	soPayload := openAIChatWebFindMap(preparePayload, map[string]struct{}{"so": {}, "sessionobserver": {}})
	if soPayload != nil && bundle.SOToken == "" {
		collectorDX, _ := soPayload["collector_dx"].(string)
		if strings.TrimSpace(collectorDX) != "" {
			bundle.SOToken, _ = helper.solveSessionObserver(ctx, session, proofToken, collectorDX)
		}
	}

	finalizeHeaders := buildOpenAIChatWebHeaders(
		"/backend-api/sentinel/chat-requirements/finalize",
		token,
		session.DeviceID,
		session.UserAgent,
		sessionID,
		account.GetChatGPTAccountID(),
		openAIChatWebBaseURL+"/",
		"*/*",
	)
	finalizeHeaders.Set("content-type", "application/json")
	finalizePayload := map[string]any{
		"conversation_mode_kind":        "primary_assistant",
		"history_and_training_disabled": false,
		"model":                         "auto",
		"supported_encodings":           []string{"v1"},
		"proof_token":                   bundle.ProofToken,
		"chat_requirements_token":       bundle.ChatRequirementsToken,
		"turnstile_token":               bundle.TurnstileToken,
		"so_token":                      bundle.SOToken,
		"prepare_token":                 bundle.PrepareToken,
		"echo_logs":                     bundle.EchoLogs,
		"device_id":                     session.DeviceID,
		"session_id":                    sessionID,
	}
	finalizeRaw, _ := json.Marshal(finalizePayload)
	finalizeResp, _, err := s.doOpenAIChatWebJSONRequest(
		ctx,
		account,
		http.MethodPost,
		openAIChatWebBaseURL+"/backend-api/sentinel/chat-requirements/finalize",
		finalizeHeaders,
		finalizeRaw,
	)
	if err == nil {
		if bundle.ProofToken == "" {
			bundle.ProofToken = openAIChatWebFindString(finalizeResp, map[string]struct{}{"prooftoken": {}, "openaisentinelprooftoken": {}})
		}
		if bundle.ChatRequirementsToken == "" {
			bundle.ChatRequirementsToken = openAIChatWebFindString(finalizeResp, map[string]struct{}{"chatrequirementstoken": {}, "openaisentinelchatrequirementstoken": {}, "requirementstoken": {}, "token": {}})
		}
		if bundle.TurnstileToken == "" {
			bundle.TurnstileToken = openAIChatWebFindString(finalizeResp, map[string]struct{}{"turnstiletoken": {}, "openaisentinelturnstiletoken": {}})
		}
		if bundle.SOToken == "" {
			bundle.SOToken = openAIChatWebFindString(finalizeResp, map[string]struct{}{"sotoken": {}, "openaisentinelsotoken": {}})
		}
		if bundle.PrepareToken == "" {
			bundle.PrepareToken = openAIChatWebFindString(finalizeResp, map[string]struct{}{"preparetoken": {}, "chatrequirementspreparetoken": {}, "openaisentinelchatrequirementspreparetoken": {}})
		}
		if bundle.ExtraData == "" {
			bundle.ExtraData = openAIChatWebFindString(finalizeResp, map[string]struct{}{"extradata": {}, "sentinelextradata": {}, "openaisentinelextradata": {}})
		}
		if bundle.EchoLogs == "" {
			bundle.EchoLogs = openAIChatWebFindString(finalizeResp, map[string]struct{}{"echologs": {}, "oaiechologs": {}})
		}
	}

	return bundle, nil
}

func (s *OpenAIGatewayService) initOpenAIChatWebConversation(
	ctx context.Context,
	account *Account,
	token string,
	session openAIChatWebSentinelSession,
	sessionID string,
) (string, string, error) {
	headers := buildOpenAIChatWebHeaders(
		"/backend-api/conversation/init",
		token,
		session.DeviceID,
		session.UserAgent,
		sessionID,
		account.GetChatGPTAccountID(),
		openAIChatWebBaseURL+"/",
		"*/*",
	)
	headers.Set("content-type", "application/json")
	payload := map[string]any{
		"history_and_training_disabled": false,
		"model":                         "auto",
		"supports_buffering":            true,
	}
	raw, _ := json.Marshal(payload)
	initResp, _, err := s.doOpenAIChatWebJSONRequest(
		ctx,
		account,
		http.MethodPost,
		openAIChatWebBaseURL+"/backend-api/conversation/init",
		headers,
		raw,
	)
	if err != nil {
		return "", openAIChatWebBaseURL + "/", err
	}
	conversationID := openAIChatWebFindString(initResp, map[string]struct{}{"conversation_id": {}, "conversationid": {}})
	if conversationID == "" {
		return "", openAIChatWebBaseURL + "/", nil
	}
	return conversationID, openAIChatWebBaseURL + "/c/" + conversationID, nil
}

func (s *OpenAIGatewayService) prepareOpenAIChatWebConversationRequest(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	token string,
) (*openAIChatWebConversationPrepareResult, error) {
	session := s.buildOpenAIChatWebSentinelSession(ctx, c, account)
	sessionID := strings.TrimSpace(session.Persona.SessionID)
	if sessionID == "" {
		sessionID = uuid.NewString()
		session.Persona.SessionID = sessionID
	}

	_, referer, err := s.initOpenAIChatWebConversation(ctx, account, token, session, sessionID)
	if err != nil {
		return nil, err
	}
	sentinelBundle, err := s.prepareOpenAIChatWebSentinel(ctx, c, account, token, session, sessionID, referer)
	if err != nil {
		return nil, err
	}

	payload, err := openAIChatWebBuildConversationPayload(body)
	if err != nil {
		return nil, err
	}

	turnTraceID := uuid.NewString()
	headers := buildOpenAIChatWebHeaders(
		"/backend-api/f/conversation/prepare",
		token,
		session.DeviceID,
		session.UserAgent,
		sessionID,
		account.GetChatGPTAccountID(),
		referer,
		"*/*",
	)
	headers.Set("content-type", "application/json")
	headers.Set("x-conduit-token", "no-token")
	headers.Set("x-oai-turn-trace-id", turnTraceID)

	raw, _ := json.Marshal(payload)
	prepareResp, respHeaders, err := s.doOpenAIChatWebJSONRequest(
		ctx,
		account,
		http.MethodPost,
		openAIChatWebBaseURL+"/backend-api/f/conversation/prepare",
		headers,
		raw,
	)
	if err != nil {
		return nil, err
	}

	if conversationID := openAIChatWebFindString(prepareResp, map[string]struct{}{"conversation_id": {}, "conversationid": {}}); conversationID != "" {
		referer = openAIChatWebBaseURL + "/c/" + conversationID
	}

	finalPayload := payload
	for _, key := range []string{"body", "payload", "request"} {
		if candidate, ok := prepareResp[key].(map[string]any); ok && candidate != nil {
			finalPayload = candidate
			break
		}
	}

	return &openAIChatWebConversationPrepareResult{
		Referer:      referer,
		TurnTraceID:  turnTraceID,
		ConduitToken: strings.TrimSpace(respHeaders.Get("x-conduit-token")),
		Payload:      finalPayload,
		Sentinel:     sentinelBundle,
		Session:      session,
		SessionID:    sessionID,
	}, nil
}

func (s *OpenAIGatewayService) pingOpenAIChatWebSentinel(
	ctx context.Context,
	account *Account,
	token string,
	prepared *openAIChatWebConversationPrepareResult,
) {
	if prepared == nil || prepared.Sentinel == nil {
		return
	}
	headers := buildOpenAIChatWebHeaders(
		"/backend-api/sentinel/ping",
		token,
		prepared.Session.DeviceID,
		prepared.Session.UserAgent,
		prepared.SessionID,
		account.GetChatGPTAccountID(),
		prepared.Referer,
		"*/*",
	)
	headers.Del("content-type")
	if prepared.Sentinel.ProofToken != "" {
		headers.Set("openai-sentinel-proof-token", prepared.Sentinel.ProofToken)
	}
	if prepared.Sentinel.ChatRequirementsToken != "" {
		headers.Set("openai-sentinel-chat-requirements-token", prepared.Sentinel.ChatRequirementsToken)
	}
	if prepared.Sentinel.TurnstileToken != "" {
		headers.Set("openai-sentinel-turnstile-token", prepared.Sentinel.TurnstileToken)
	}
	if prepared.Sentinel.SOToken != "" {
		headers.Set("openai-sentinel-so-token", prepared.Sentinel.SOToken)
	}
	if prepared.Sentinel.PrepareToken != "" {
		headers.Set("openai-sentinel-chat-requirements-prepare-token", prepared.Sentinel.PrepareToken)
	}
	if prepared.Sentinel.ExtraData != "" {
		headers.Set("openai-sentinel-extra-data", prepared.Sentinel.ExtraData)
	}
	if prepared.Sentinel.EchoLogs != "" {
		headers.Set("oai-echo-logs", prepared.Sentinel.EchoLogs)
	}
	_, _, _ = s.doOpenAIChatWebJSONRequest(
		ctx,
		account,
		http.MethodPost,
		openAIChatWebBaseURL+"/backend-api/sentinel/ping",
		headers,
		nil,
	)
}

func (s *OpenAIGatewayService) beginOpenAIChatWebConversationRequest(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
) (*http.Response, *openAIChatWebConversationPrepareResult, string, error) {
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, nil, "", err
	}

	prepared, err := s.prepareOpenAIChatWebConversationRequest(ctx, c, account, body, token)
	if err != nil {
		return nil, nil, "", err
	}

	payloadRaw, err := json.Marshal(prepared.Payload)
	if err != nil {
		return nil, nil, "", fmt.Errorf("marshal chatweb conversation payload: %w", err)
	}
	setOpsUpstreamRequestBodyContext(c, payloadRaw)

	headers := buildOpenAIChatWebHeaders(
		"/backend-api/f/conversation",
		token,
		prepared.Session.DeviceID,
		prepared.Session.UserAgent,
		prepared.SessionID,
		account.GetChatGPTAccountID(),
		prepared.Referer,
		"text/event-stream",
	)
	headers.Set("content-type", "application/json")
	headers.Set("x-oai-turn-trace-id", prepared.TurnTraceID)
	if strings.TrimSpace(prepared.ConduitToken) != "" {
		headers.Set("x-conduit-token", prepared.ConduitToken)
	}
	if prepared.Sentinel != nil {
		if prepared.Sentinel.ProofToken != "" {
			headers.Set("openai-sentinel-proof-token", prepared.Sentinel.ProofToken)
		}
		if prepared.Sentinel.ChatRequirementsToken != "" {
			headers.Set("openai-sentinel-chat-requirements-token", prepared.Sentinel.ChatRequirementsToken)
		}
		if prepared.Sentinel.TurnstileToken != "" {
			headers.Set("openai-sentinel-turnstile-token", prepared.Sentinel.TurnstileToken)
		}
		if prepared.Sentinel.SOToken != "" {
			headers.Set("openai-sentinel-so-token", prepared.Sentinel.SOToken)
		}
		if prepared.Sentinel.PrepareToken != "" {
			headers.Set("openai-sentinel-chat-requirements-prepare-token", prepared.Sentinel.PrepareToken)
		}
		if prepared.Sentinel.ExtraData != "" {
			headers.Set("openai-sentinel-extra-data", prepared.Sentinel.ExtraData)
		}
		if prepared.Sentinel.EchoLogs != "" {
			headers.Set("oai-echo-logs", prepared.Sentinel.EchoLogs)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIChatWebBaseURL+"/backend-api/f/conversation", bytes.NewReader(payloadRaw))
	if err != nil {
		return nil, nil, "", err
	}
	req.Host = "chatgpt.com"
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamErrorContext(c, 0, safeErr, "")
		appendOpsUpstreamErrorContext(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Passthrough:        true,
			Kind:               "request_error",
			Message:            safeErr,
		})
		return nil, nil, "", newProxyRequestFailoverError(account, proxyURL, err)
	}

	return resp, prepared, token, nil
}

func (s *OpenAIGatewayService) forwardOpenAIChatWebConversationContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	reqModel string,
	reasoningEffort *string,
	reqStream bool,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	resp, prepared, token, err := s.beginOpenAIChatWebConversationRequest(ctx, c, account, body)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	defer s.pingOpenAIChatWebSentinel(context.Background(), account, token, prepared)

	if resp.StatusCode >= 400 {
		return nil, s.handleErrorResponsePassthroughContext(ctx, resp, c, account, body)
	}

	if reqStream {
		streamCtx := withOpenAIReasoningEffort(ctx, reasoningEffort)
		streamResult, err := s.handleStreamingResponseContext(streamCtx, resp, c, account, startTime, reqModel, reqModel)
		if err != nil {
			return nil, err
		}
		usageValue := OpenAIUsage{}
		if streamResult != nil && streamResult.usage != nil {
			usageValue = *streamResult.usage
		}
		firstTokenMs := (*int)(nil)
		if streamResult != nil {
			firstTokenMs = streamResult.firstTokenMs
		}
		return &OpenAIForwardResult{
			RequestID:       resp.Header.Get("x-request-id"),
			Usage:           usageValue,
			Model:           reqModel,
			UpstreamModel:   reqModel,
			ReasoningEffort: reasoningEffort,
			Stream:          true,
			OpenAIWSMode:    false,
			ResponseHeaders: resp.Header.Clone(),
			Duration:        time.Since(startTime),
			FirstTokenMs:    firstTokenMs,
		}, nil
	}

	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	respBody, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		return nil, err
	}
	usage, err := s.handleOAuthSSEToJSONContext(resp, c, respBody, reqModel, reqModel)
	if err != nil {
		return nil, err
	}
	usageValue := OpenAIUsage{}
	if usage != nil {
		usageValue = *usage
	}
	return &OpenAIForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		Usage:           usageValue,
		Model:           reqModel,
		UpstreamModel:   reqModel,
		ReasoningEffort: reasoningEffort,
		Stream:          false,
		OpenAIWSMode:    false,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
	}, nil
}
