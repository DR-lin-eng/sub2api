package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	openAIChatWebImagesUpstreamModelDefault = "auto"
	openAIChatWebImagesTimeout              = 180 * time.Second
)

type openAIChatWebImageItem struct {
	B64JSON       string
	URL           string
	RevisedPrompt string
}

type openAIChatWebImageResult struct {
	Created       int64
	Data          []openAIChatWebImageItem
	Usage         OpenAIUsage
	RequestID     string
	UpstreamModel string
}

type openAIChatWebCompatImageRequest struct {
	RequestModel   string
	UpstreamModel  string
	Prompt         string
	N              int
	ResponseFormat string
	Uploads        []OpenAIImagesUpload
}

type openAIChatWebUploadedImage struct {
	FileID string
	Upload OpenAIImagesUpload
}

type openAIChatWebImageStreamSummary struct {
	ConversationID string
	FileIDs        []string
	Text           string
}

func (s *OpenAIGatewayService) forwardOpenAIImagesChatWebContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	parsed *OpenAIImagesRequest,
	channelMappedModel string,
) (*OpenAIForwardResult, error) {
	if c == nil {
		return nil, fmt.Errorf("gateway context is nil")
	}
	if parsed == nil {
		return nil, fmt.Errorf("parsed images request is required")
	}
	if parsed.Stream {
		return nil, writeOpenAIChatWebCompatError(c, http.StatusBadRequest, "invalid_request_error", "ChatWeb Images API does not support stream=true")
	}
	if strings.TrimSpace(parsed.Prompt) == "" {
		return nil, writeOpenAIChatWebCompatError(c, http.StatusBadRequest, "invalid_request_error", "prompt is required")
	}

	startTime := time.Now()
	requestModel := strings.TrimSpace(parsed.Model)
	if requestModel == "" {
		requestModel = "gpt-image-1"
	}

	req := &openAIChatWebCompatImageRequest{
		RequestModel:   requestModel,
		UpstreamModel:  resolveOpenAIChatWebImageUpstreamModel(account, requestModel, channelMappedModel),
		Prompt:         strings.TrimSpace(parsed.Prompt),
		N:              parsed.N,
		ResponseFormat: strings.TrimSpace(parsed.ResponseFormat),
		Uploads:        cloneOpenAIImageUploads(parsed.Uploads),
	}
	if req.N <= 0 {
		req.N = 1
	}

	result, err := s.executeOpenAIChatWebImageRequestContext(ctx, c, account, req)
	if err != nil {
		var failoverErr *UpstreamFailoverError
		if errors.As(err, &failoverErr) {
			return nil, err
		}
		if !c.ResponseWritten() {
			_ = writeOpenAIChatWebCompatError(c, http.StatusBadGateway, "upstream_error", err.Error())
		}
		return nil, err
	}

	c.WriteJSON(http.StatusOK, buildOpenAIChatWebImagesAPIResponse(result, req.ResponseFormat))

	return &OpenAIForwardResult{
		RequestID:     result.RequestID,
		Usage:         result.Usage,
		Model:         requestModel,
		UpstreamModel: req.UpstreamModel,
		Stream:        false,
		Duration:      time.Since(startTime),
		ImageCount:    len(result.Data),
		ImageSize:     parsed.SizeTier,
	}, nil
}

func (s *OpenAIGatewayService) tryForwardOpenAIChatWebImageChatCompletionsContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	reqModel string,
	startTime time.Time,
) (bool, *OpenAIForwardResult, error) {
	if c == nil || account == nil || !account.IsOpenAIChatWebMode() {
		return false, nil, nil
	}

	req, handled, err := parseOpenAIChatWebChatCompletionsImageRequest(body)
	if err != nil || !handled {
		return handled, nil, err
	}
	if strings.TrimSpace(req.RequestModel) == "" {
		req.RequestModel = strings.TrimSpace(reqModel)
	}
	req.UpstreamModel = resolveOpenAIChatWebImageUpstreamModel(account, req.RequestModel, req.UpstreamModel)

	result, execErr := s.executeOpenAIChatWebImageRequestContext(ctx, c, account, req)
	if execErr != nil {
		var failoverErr *UpstreamFailoverError
		if errors.As(execErr, &failoverErr) {
			return true, nil, execErr
		}
		if !c.ResponseWritten() {
			_ = writeOpenAIChatWebCompatError(c, http.StatusBadGateway, "upstream_error", execErr.Error())
		}
		return true, nil, execErr
	}

	c.WriteJSON(http.StatusOK, buildOpenAIChatWebChatCompletionResponse(result, req.RequestModel))
	return true, &OpenAIForwardResult{
		RequestID:     result.RequestID,
		Usage:         result.Usage,
		Model:         req.RequestModel,
		UpstreamModel: req.UpstreamModel,
		Stream:        false,
		Duration:      time.Since(startTime),
		ImageCount:    len(result.Data),
		ImageSize:     "2K",
	}, nil
}

func (s *OpenAIGatewayService) tryForwardOpenAIChatWebImageResponsesContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	reqModel string,
	startTime time.Time,
) (bool, *OpenAIForwardResult, error) {
	if c == nil || account == nil || !account.IsOpenAIChatWebMode() {
		return false, nil, nil
	}

	req, handled, err := parseOpenAIChatWebResponsesImageRequest(body)
	if err != nil || !handled {
		return handled, nil, err
	}
	if strings.TrimSpace(reqModel) != "" && strings.TrimSpace(req.RequestModel) == "" {
		req.RequestModel = strings.TrimSpace(reqModel)
	}
	req.UpstreamModel = resolveOpenAIChatWebImageUpstreamModel(account, req.RequestModel, req.UpstreamModel)

	result, execErr := s.executeOpenAIChatWebImageRequestContext(ctx, c, account, req)
	if execErr != nil {
		var failoverErr *UpstreamFailoverError
		if errors.As(execErr, &failoverErr) {
			return true, nil, execErr
		}
		if !c.ResponseWritten() {
			_ = writeOpenAIChatWebCompatError(c, http.StatusBadGateway, "upstream_error", execErr.Error())
		}
		return true, nil, execErr
	}

	c.WriteJSON(http.StatusOK, buildOpenAIChatWebResponsesImageResponse(result, req.RequestModel))
	return true, &OpenAIForwardResult{
		RequestID:     result.RequestID,
		Usage:         result.Usage,
		Model:         req.RequestModel,
		UpstreamModel: req.UpstreamModel,
		Stream:        false,
		Duration:      time.Since(startTime),
		ImageCount:    len(result.Data),
		ImageSize:     "2K",
	}, nil
}

func (s *OpenAIGatewayService) executeOpenAIChatWebImageRequestContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	req *openAIChatWebCompatImageRequest,
) (*openAIChatWebImageResult, error) {
	if req == nil {
		return nil, fmt.Errorf("chatweb image request is required")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, writeOpenAIChatWebCompatError(c, http.StatusBadRequest, "invalid_request_error", "prompt is required")
	}
	if req.N <= 0 {
		req.N = 1
	}

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	result := &openAIChatWebImageResult{
		Created:       time.Now().Unix(),
		UpstreamModel: req.UpstreamModel,
	}

	var lastErr error
	for i := 0; i < req.N; i++ {
		turnResult, turnErr := s.executeSingleOpenAIChatWebImageTurnContext(ctx, c, account, token, req)
		if turnErr != nil {
			lastErr = turnErr
			break
		}
		if turnResult == nil {
			lastErr = fmt.Errorf("chatweb image turn returned no result")
			break
		}
		if result.RequestID == "" {
			result.RequestID = turnResult.RequestID
		}
		result.Data = append(result.Data, turnResult.Data...)
		mergeOpenAIImageUsage(&result.Usage, turnResult.Usage)
	}

	if len(result.Data) == 0 && lastErr != nil {
		return nil, lastErr
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no image returned from ChatWeb")
	}
	return result, nil
}

func (s *OpenAIGatewayService) executeSingleOpenAIChatWebImageTurnContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	token string,
	req *openAIChatWebCompatImageRequest,
) (*openAIChatWebImageResult, error) {
	session := s.buildOpenAIChatWebSentinelSession(ctx, c, account)
	sessionID := strings.TrimSpace(session.Persona.SessionID)
	if sessionID == "" {
		sessionID = uuid.NewString()
		session.Persona.SessionID = sessionID
	}
	referer := openAIChatWebBaseURL + "/"

	sentinelBundle, err := s.prepareOpenAIChatWebSentinel(ctx, c, account, token, session, sessionID, referer)
	if err != nil {
		return nil, fmt.Errorf("prepare chatweb sentinel: %w", err)
	}

	uploadedImages := make([]openAIChatWebUploadedImage, 0, len(req.Uploads))
	for _, upload := range req.Uploads {
		fileID, uploadErr := s.uploadOpenAIChatWebImageContext(ctx, account, token, session, sessionID, referer, upload)
		if uploadErr != nil {
			return nil, uploadErr
		}
		uploadedImages = append(uploadedImages, openAIChatWebUploadedImage{
			FileID: fileID,
			Upload: upload,
		})
	}

	payload := buildOpenAIChatWebImageConversationPayload(req.Prompt, req.UpstreamModel, uploadedImages, session)
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal chatweb image payload: %w", err)
	}
	setOpsUpstreamRequestBodyContext(c, payloadRaw)

	headers := buildOpenAIChatWebHeaders(
		"/backend-api/conversation",
		token,
		session.DeviceID,
		session.UserAgent,
		sessionID,
		account.GetChatGPTAccountID(),
		referer,
		"text/event-stream",
	)
	headers.Set("content-type", "application/json")
	applyOpenAIChatWebSentinelHeaders(headers, sentinelBundle)

	conversationURL := openAIChatWebBaseURL + "/backend-api/conversation"
	httpReq, err := http.NewRequestWithContext(withOpenAIChatWebImageTimeout(ctx), http.MethodPost, conversationURL, bytes.NewReader(payloadRaw))
	if err != nil {
		return nil, err
	}
	httpReq.Host = "chatgpt.com"
	for key, values := range headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(httpReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, newProxyRequestFailoverError(account, proxyURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode),
			}
		}
		if upstreamMsg == "" {
			upstreamMsg = fmt.Sprintf("ChatWeb image conversation failed: status=%d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", upstreamMsg)
	}

	respBody, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, err
	}

	summary := parseOpenAIChatWebImageStreamBody(respBody)
	outputIDs := filterOpenAIChatWebOutputFileIDs(summary.FileIDs, uploadedImages)
	if summary.ConversationID != "" && len(outputIDs) == 0 {
		polledIDs, pollErr := s.pollOpenAIChatWebImageOutputIDsContext(ctx, account, token, session, sessionID, referer, summary.ConversationID, uploadedImages)
		if pollErr == nil {
			outputIDs = polledIDs
		}
	}
	if len(outputIDs) == 0 {
		msg := strings.TrimSpace(summary.Text)
		if msg == "" {
			msg = "no image returned from ChatWeb"
		}
		return nil, fmt.Errorf("%s", msg)
	}

	firstID := outputIDs[0]
	downloadURL, err := s.fetchOpenAIChatWebDownloadURLContext(ctx, account, token, session, sessionID, referer, summary.ConversationID, firstID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(downloadURL) == "" {
		return nil, fmt.Errorf("failed to get ChatWeb image download url")
	}

	b64JSON, err := s.downloadOpenAIChatWebImageBase64Context(ctx, account, downloadURL)
	if err != nil {
		return nil, err
	}

	usage := OpenAIUsage{}
	if parsed := s.parseSSEUsageFromBody(string(respBody)); parsed != nil {
		usage = *parsed
	}

	return &openAIChatWebImageResult{
		Created:       time.Now().Unix(),
		Usage:         usage,
		RequestID:     resp.Header.Get("x-request-id"),
		UpstreamModel: req.UpstreamModel,
		Data: []openAIChatWebImageItem{{
			B64JSON:       b64JSON,
			URL:           downloadURL,
			RevisedPrompt: req.Prompt,
		}},
	}, nil
}

func (s *OpenAIGatewayService) uploadOpenAIChatWebImageContext(
	ctx context.Context,
	account *Account,
	token string,
	session openAIChatWebSentinelSession,
	sessionID string,
	referer string,
	upload OpenAIImagesUpload,
) (string, error) {
	fileName := strings.TrimSpace(upload.FileName)
	if fileName == "" {
		fileName = "image.png"
	}
	contentType := strings.TrimSpace(upload.ContentType)
	if contentType == "" {
		contentType = http.DetectContentType(upload.Data)
	}
	if contentType == "" {
		contentType = "image/png"
	}

	initHeaders := buildOpenAIChatWebHeaders(
		"/backend-api/files",
		token,
		session.DeviceID,
		session.UserAgent,
		sessionID,
		account.GetChatGPTAccountID(),
		referer,
		"*/*",
	)
	initHeaders.Set("content-type", "application/json")
	initPayload := map[string]any{
		"file_name":           fileName,
		"file_size":           len(upload.Data),
		"use_case":            "multimodal",
		"timezone_offset_min": -480,
		"reset_rate_limits":   false,
	}
	initRaw, _ := json.Marshal(initPayload)
	initResp, _, err := s.doOpenAIChatWebJSONRequest(
		withOpenAIChatWebImageTimeout(ctx),
		account,
		http.MethodPost,
		openAIChatWebBaseURL+"/backend-api/files",
		initHeaders,
		initRaw,
	)
	if err != nil {
		return "", fmt.Errorf("chatweb file upload init failed: %w", err)
	}

	uploadURL := strings.TrimSpace(openAIChatWebFindString(initResp, map[string]struct{}{"upload_url": {}, "uploadurl": {}}))
	fileID := strings.TrimSpace(openAIChatWebFindString(initResp, map[string]struct{}{"file_id": {}, "fileid": {}}))
	if uploadURL == "" || fileID == "" {
		return "", fmt.Errorf("chatweb file upload init returned no upload_url or file_id")
	}

	putReq, err := http.NewRequestWithContext(withOpenAIChatWebImageTimeout(ctx), http.MethodPut, uploadURL, bytes.NewReader(upload.Data))
	if err != nil {
		return "", err
	}
	putReq.Header.Set("Content-Type", contentType)
	putReq.Header.Set("x-ms-blob-type", "BlockBlob")
	putReq.Header.Set("x-ms-version", "2020-04-08")

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	putResp, err := s.httpUpstream.Do(putReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return "", newProxyRequestFailoverError(account, proxyURL, err)
	}
	defer func() { _ = putResp.Body.Close() }()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		return "", fmt.Errorf("chatweb file upload PUT failed: status=%d", putResp.StatusCode)
	}

	processHeaders := buildOpenAIChatWebHeaders(
		"/backend-api/files/process_upload_stream",
		token,
		session.DeviceID,
		session.UserAgent,
		sessionID,
		account.GetChatGPTAccountID(),
		referer,
		"*/*",
	)
	processHeaders.Set("content-type", "application/json")
	processPayload := map[string]any{
		"file_id":             fileID,
		"use_case":            "multimodal",
		"index_for_retrieval": false,
		"file_name":           fileName,
	}
	processRaw, _ := json.Marshal(processPayload)
	if _, _, err := s.doOpenAIChatWebJSONRequest(
		withOpenAIChatWebImageTimeout(ctx),
		account,
		http.MethodPost,
		openAIChatWebBaseURL+"/backend-api/files/process_upload_stream",
		processHeaders,
		processRaw,
	); err != nil {
		return "", fmt.Errorf("chatweb file process failed: %w", err)
	}

	return fileID, nil
}

func (s *OpenAIGatewayService) pollOpenAIChatWebImageOutputIDsContext(
	ctx context.Context,
	account *Account,
	token string,
	session openAIChatWebSentinelSession,
	sessionID string,
	referer string,
	conversationID string,
	uploaded []openAIChatWebUploadedImage,
) ([]string, error) {
	deadline := time.Now().Add(openAIChatWebImagesTimeout)
	for time.Now().Before(deadline) {
		fileIDs, err := s.fetchOpenAIChatWebConversationOutputIDsContext(ctx, account, token, session, sessionID, referer, conversationID)
		if err == nil {
			filtered := filterOpenAIChatWebOutputFileIDs(fileIDs, uploaded)
			if len(filtered) > 0 {
				return filtered, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
	return nil, fmt.Errorf("chatweb image poll timed out")
}

func (s *OpenAIGatewayService) fetchOpenAIChatWebConversationOutputIDsContext(
	ctx context.Context,
	account *Account,
	token string,
	session openAIChatWebSentinelSession,
	sessionID string,
	referer string,
	conversationID string,
) ([]string, error) {
	headers := buildOpenAIChatWebHeaders(
		"/backend-api/conversation/"+conversationID,
		token,
		session.DeviceID,
		session.UserAgent,
		sessionID,
		account.GetChatGPTAccountID(),
		referer,
		"*/*",
	)
	payload, _, err := s.doOpenAIChatWebJSONRequest(
		withOpenAIChatWebImageTimeout(ctx),
		account,
		http.MethodGet,
		openAIChatWebBaseURL+"/backend-api/conversation/"+conversationID,
		headers,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return extractOpenAIChatWebConversationFileIDs(payload), nil
}

func (s *OpenAIGatewayService) fetchOpenAIChatWebDownloadURLContext(
	ctx context.Context,
	account *Account,
	token string,
	session openAIChatWebSentinelSession,
	sessionID string,
	referer string,
	conversationID string,
	fileID string,
) (string, error) {
	rawID := strings.TrimSpace(strings.TrimPrefix(fileID, "sed:"))
	targetPath := "/backend-api/files/" + rawID + "/download"
	url := openAIChatWebBaseURL + targetPath
	if strings.HasPrefix(strings.TrimSpace(fileID), "sed:") {
		targetPath = "/backend-api/conversation/" + strings.TrimSpace(conversationID) + "/attachment/" + rawID + "/download"
		url = openAIChatWebBaseURL + targetPath
	}

	headers := buildOpenAIChatWebHeaders(
		targetPath,
		token,
		session.DeviceID,
		session.UserAgent,
		sessionID,
		account.GetChatGPTAccountID(),
		referer,
		"*/*",
	)
	payload, _, err := s.doOpenAIChatWebJSONRequest(
		withOpenAIChatWebImageTimeout(ctx),
		account,
		http.MethodGet,
		url,
		headers,
		nil,
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(openAIChatWebFindString(payload, map[string]struct{}{"download_url": {}, "downloadurl": {}})), nil
}

func (s *OpenAIGatewayService) downloadOpenAIChatWebImageBase64Context(
	ctx context.Context,
	account *Account,
	downloadURL string,
) (string, error) {
	req, err := http.NewRequestWithContext(withOpenAIChatWebImageTimeout(ctx), http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return "", newProxyRequestFailoverError(account, proxyURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("chatweb image download failed: status=%d", resp.StatusCode)
	}
	body, err := readUpstreamResponseBodyLimited(resp.Body, 64<<20)
	if err != nil {
		return "", err
	}
	if len(body) == 0 {
		return "", fmt.Errorf("chatweb image download returned empty body")
	}
	return base64.StdEncoding.EncodeToString(body), nil
}

func buildOpenAIChatWebImageConversationPayload(
	prompt string,
	upstreamModel string,
	uploaded []openAIChatWebUploadedImage,
	session openAIChatWebSentinelSession,
) map[string]any {
	upstreamModel = strings.TrimSpace(upstreamModel)
	if upstreamModel == "" {
		upstreamModel = openAIChatWebImagesUpstreamModelDefault
	}

	messageID := uuid.NewString()
	parentMessageID := uuid.NewString()
	websocketRequestID := uuid.NewString()

	content := map[string]any{
		"content_type": "text",
		"parts":        []any{prompt},
	}
	metadata := map[string]any{
		"attachments": []any{},
	}

	if len(uploaded) > 0 {
		parts := make([]any, 0, len(uploaded)+1)
		attachments := make([]any, 0, len(uploaded))
		for _, item := range uploaded {
			width := item.Upload.Width
			height := item.Upload.Height
			if width <= 0 || height <= 0 {
				width, height = parseOpenAIImageDimensions(item.Upload.Data)
			}
			parts = append(parts, map[string]any{
				"content_type":  "image_asset_pointer",
				"asset_pointer": "sediment://" + item.FileID,
				"size_bytes":    len(item.Upload.Data),
				"width":         width,
				"height":        height,
			})
			attachments = append(attachments, map[string]any{
				"id":           item.FileID,
				"size":         len(item.Upload.Data),
				"name":         item.Upload.FileName,
				"mime_type":    item.Upload.ContentType,
				"width":        width,
				"height":       height,
				"source":       "local",
				"is_big_paste": false,
			})
		}
		parts = append(parts, prompt)
		content = map[string]any{
			"content_type": "multimodal_text",
			"parts":        parts,
		}
		metadata["attachments"] = attachments
	}

	return map[string]any{
		"action": "next",
		"messages": []any{
			map[string]any{
				"id": messageID,
				"author": map[string]any{
					"role": "user",
				},
				"content":  content,
				"metadata": metadata,
			},
		},
		"parent_message_id":                    parentMessageID,
		"model":                                upstreamModel,
		"history_and_training_disabled":        false,
		"timezone_offset_min":                  -480,
		"timezone":                             "America/Los_Angeles",
		"conversation_mode":                    map[string]any{"kind": "primary_assistant"},
		"force_paragen":                        false,
		"force_paragen_model_slug":             "",
		"force_rate_limit":                     false,
		"force_use_sse":                        true,
		"paragen_cot_summary_display_override": "allow",
		"reset_rate_limits":                    false,
		"suggestions":                          []any{},
		"supported_encodings":                  []any{},
		"system_hints":                         []string{"picture_v2"},
		"variant_purpose":                      "comparison_implicit",
		"websocket_request_id":                 websocketRequestID,
		"client_contextual_info": map[string]any{
			"is_dark_mode":      false,
			"time_since_loaded": 120,
			"page_height":       session.ScreenHeight,
			"page_width":        session.ScreenWidth,
			"pixel_ratio":       2,
			"screen_height":     session.ScreenHeight,
			"screen_width":      session.ScreenWidth,
		},
	}
}

func applyOpenAIChatWebSentinelHeaders(headers http.Header, bundle *openAIChatWebSentinelBundle) {
	if headers == nil || bundle == nil {
		return
	}
	if bundle.ProofToken != "" {
		headers.Set("openai-sentinel-proof-token", bundle.ProofToken)
	}
	if bundle.ChatRequirementsToken != "" {
		headers.Set("openai-sentinel-chat-requirements-token", bundle.ChatRequirementsToken)
	}
	if bundle.TurnstileToken != "" {
		headers.Set("openai-sentinel-turnstile-token", bundle.TurnstileToken)
	}
	if bundle.SOToken != "" {
		headers.Set("openai-sentinel-so-token", bundle.SOToken)
	}
	if bundle.PrepareToken != "" {
		headers.Set("openai-sentinel-chat-requirements-prepare-token", bundle.PrepareToken)
	}
	if bundle.ExtraData != "" {
		headers.Set("openai-sentinel-extra-data", bundle.ExtraData)
	}
	if bundle.EchoLogs != "" {
		headers.Set("oai-echo-logs", bundle.EchoLogs)
	}
}

func buildOpenAIChatWebImagesAPIResponse(result *openAIChatWebImageResult, responseFormat string) map[string]any {
	format := strings.ToLower(strings.TrimSpace(responseFormat))
	if format == "" {
		format = "b64_json"
	}
	data := make([]map[string]any, 0, len(result.Data))
	for _, item := range result.Data {
		entry := map[string]any{
			"revised_prompt": item.RevisedPrompt,
		}
		if format == "url" && item.URL != "" {
			entry["url"] = item.URL
		} else {
			entry["b64_json"] = item.B64JSON
			if item.URL != "" {
				entry["url"] = item.URL
			}
		}
		data = append(data, entry)
	}
	return map[string]any{
		"created": result.Created,
		"data":    data,
	}
}

func buildOpenAIChatWebChatCompletionResponse(result *openAIChatWebImageResult, model string) map[string]any {
	images := make([]string, 0, len(result.Data))
	for i, item := range result.Data {
		if strings.TrimSpace(item.B64JSON) == "" {
			continue
		}
		images = append(images, fmt.Sprintf("![image_%d](data:image/png;base64,%s)", i+1, item.B64JSON))
	}
	content := "Image generation completed."
	if len(images) > 0 {
		content = strings.Join(images, "\n\n")
	}
	return map[string]any{
		"id":      "chatcmpl-" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		"object":  "chat.completion",
		"created": result.Created,
		"model":   model,
		"choices": []any{
			map[string]any{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": buildOpenAIUsageResponseMap(result.Usage),
	}
}

func buildOpenAIChatWebResponsesImageResponse(result *openAIChatWebImageResult, model string) map[string]any {
	output := make([]map[string]any, 0, len(result.Data))
	for i, item := range result.Data {
		output = append(output, map[string]any{
			"id":             fmt.Sprintf("ig_%d", i+1),
			"type":           "image_generation_call",
			"status":         "completed",
			"result":         item.B64JSON,
			"revised_prompt": item.RevisedPrompt,
		})
	}
	resp := map[string]any{
		"id":                  fmt.Sprintf("resp_%d", result.Created),
		"object":              "response",
		"created_at":          result.Created,
		"status":              "completed",
		"error":               nil,
		"incomplete_details":  nil,
		"model":               model,
		"output":              output,
		"parallel_tool_calls": false,
	}
	if usage := buildOpenAIUsageResponseMap(result.Usage); len(usage) > 0 {
		resp["usage"] = usage
	}
	return resp
}

func buildOpenAIUsageResponseMap(usage OpenAIUsage) map[string]any {
	return map[string]any{
		"input_tokens":  usage.InputTokens,
		"output_tokens": usage.OutputTokens,
		"total_tokens":  usage.InputTokens + usage.OutputTokens,
		"input_tokens_details": map[string]any{
			"cached_tokens": usage.CacheReadInputTokens,
		},
	}
}

func writeOpenAIChatWebCompatError(c gatewayctx.GatewayContext, status int, errType, message string) error {
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "OpenAI ChatWeb request failed"
	}
	if c != nil && !c.ResponseWritten() {
		c.WriteJSON(status, gin.H{
			"error": gin.H{
				"type":    errType,
				"message": message,
			},
		})
	}
	return fmt.Errorf("%s", message)
}

func resolveOpenAIChatWebImageUpstreamModel(account *Account, requestedModel, defaultMappedModel string) string {
	if mapped := strings.TrimSpace(defaultMappedModel); mapped != "" {
		return mapped
	}
	if account != nil {
		if mapped, matched := account.ResolveMappedModel(requestedModel); matched && strings.TrimSpace(mapped) != "" {
			return strings.TrimSpace(mapped)
		}
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(requestedModel)), "gpt-image-") {
		return openAIChatWebImagesUpstreamModelDefault
	}
	if strings.TrimSpace(requestedModel) == "" {
		return openAIChatWebImagesUpstreamModelDefault
	}
	return strings.TrimSpace(requestedModel)
}

func parseOpenAIChatWebChatCompletionsImageRequest(body []byte) (*openAIChatWebCompatImageRequest, bool, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false, fmt.Errorf("parse chat completions image request: %w", err)
	}

	model := strings.TrimSpace(chatWebString(payload["model"]))
	if !isOpenAIChatWebImageChatRequest(payload) {
		return nil, false, nil
	}
	if chatWebBool(payload["stream"]) {
		return nil, true, fmt.Errorf("chatweb image chat completions does not support stream=true")
	}

	prompt := strings.TrimSpace(chatWebString(payload["prompt"]))
	if prompt == "" {
		messages, _ := payload["messages"].([]any)
		prompt = extractOpenAIChatWebChatPrompt(messages)
	}
	if prompt == "" {
		return nil, true, fmt.Errorf("prompt is required")
	}

	messages, _ := payload["messages"].([]any)
	uploads := extractOpenAIChatWebChatImages(messages)

	n := chatWebInt(payload["n"], 1)
	if n <= 0 {
		n = 1
	}
	if model == "" {
		model = "gpt-image-1"
	}

	return &openAIChatWebCompatImageRequest{
		RequestModel:   model,
		UpstreamModel:  resolveOpenAIChatWebImageUpstreamModel(nil, model, ""),
		Prompt:         prompt,
		N:              n,
		ResponseFormat: "b64_json",
		Uploads:        uploads,
	}, true, nil
}

func parseOpenAIChatWebResponsesImageRequest(body []byte) (*openAIChatWebCompatImageRequest, bool, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false, fmt.Errorf("parse responses image request: %w", err)
	}
	if !hasOpenAIChatWebImageGenerationTool(payload) {
		return nil, false, nil
	}
	if chatWebBool(payload["stream"]) {
		return nil, true, fmt.Errorf("chatweb image responses does not support stream=true")
	}

	model := strings.TrimSpace(chatWebString(payload["model"]))
	if model == "" {
		model = "gpt-5"
	}
	input := payload["input"]
	prompt := extractOpenAIChatWebResponsesPrompt(input)
	if prompt == "" {
		return nil, true, fmt.Errorf("input text is required")
	}

	return &openAIChatWebCompatImageRequest{
		RequestModel:   model,
		UpstreamModel:  resolveOpenAIChatWebImageUpstreamModel(nil, model, ""),
		Prompt:         prompt,
		N:              1,
		ResponseFormat: "b64_json",
		Uploads:        extractOpenAIChatWebResponsesImages(input),
	}, true, nil
}

func hasOpenAIChatWebImageGenerationTool(payload map[string]any) bool {
	tools, _ := payload["tools"].([]any)
	for _, item := range tools {
		if tool, ok := item.(map[string]any); ok && strings.TrimSpace(chatWebString(tool["type"])) == "image_generation" {
			return true
		}
	}
	if toolChoice, ok := payload["tool_choice"].(map[string]any); ok {
		return strings.TrimSpace(chatWebString(toolChoice["type"])) == "image_generation"
	}
	return false
}

func isOpenAIChatWebImageChatRequest(payload map[string]any) bool {
	model := strings.TrimSpace(chatWebString(payload["model"]))
	if strings.HasPrefix(strings.ToLower(model), "gpt-image-") {
		return true
	}
	modalities, _ := payload["modalities"].([]any)
	for _, item := range modalities {
		if strings.EqualFold(strings.TrimSpace(chatWebString(item)), "image") {
			return true
		}
	}
	return false
}

func extractOpenAIChatWebResponsesPrompt(input any) string {
	switch value := input.(type) {
	case string:
		return strings.TrimSpace(value)
	case map[string]any:
		role := strings.ToLower(strings.TrimSpace(chatWebString(value["role"])))
		if role != "" && role != "user" {
			return ""
		}
		return extractOpenAIChatWebPromptFromContent(value["content"])
	case []any:
		var parts []string
		for _, item := range value {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if strings.TrimSpace(chatWebString(obj["type"])) == "input_text" {
				if text := strings.TrimSpace(chatWebString(obj["text"])); text != "" {
					parts = append(parts, text)
				}
				continue
			}
			role := strings.ToLower(strings.TrimSpace(chatWebString(obj["role"])))
			if role != "" && role != "user" {
				continue
			}
			if prompt := extractOpenAIChatWebPromptFromContent(obj["content"]); prompt != "" {
				parts = append(parts, prompt)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func extractOpenAIChatWebResponsesImages(input any) []OpenAIImagesUpload {
	switch value := input.(type) {
	case map[string]any:
		return extractOpenAIChatWebImagesFromContent(value["content"])
	case []any:
		var uploads []OpenAIImagesUpload
		for _, item := range value {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			switch strings.TrimSpace(chatWebString(obj["type"])) {
			case "input_image":
				if upload, ok := openAIChatWebUploadFromImageURL(chatWebString(obj["image_url"]), len(uploads)+1); ok {
					uploads = append(uploads, upload)
				}
			default:
				uploads = append(uploads, extractOpenAIChatWebImagesFromContent(obj["content"])...)
			}
		}
		return uploads
	default:
		return nil
	}
}

func extractOpenAIChatWebChatPrompt(messages []any) string {
	var parts []string
	for _, item := range messages {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.ToLower(strings.TrimSpace(chatWebString(obj["role"]))) != "user" {
			continue
		}
		if prompt := extractOpenAIChatWebPromptFromContent(obj["content"]); prompt != "" {
			parts = append(parts, prompt)
		}
	}
	return strings.Join(parts, "\n")
}

func extractOpenAIChatWebChatImages(messages []any) []OpenAIImagesUpload {
	var uploads []OpenAIImagesUpload
	for _, item := range messages {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.ToLower(strings.TrimSpace(chatWebString(obj["role"]))) != "user" {
			continue
		}
		uploads = append(uploads, extractOpenAIChatWebImagesFromContent(obj["content"])...)
	}
	return uploads
}

func extractOpenAIChatWebPromptFromContent(content any) string {
	parts, ok := content.([]any)
	if !ok {
		if text, ok := content.(string); ok {
			return strings.TrimSpace(text)
		}
		return ""
	}
	var texts []string
	for _, item := range parts {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		switch strings.TrimSpace(chatWebString(obj["type"])) {
		case "text", "input_text":
			text := strings.TrimSpace(chatWebString(obj["text"]))
			if text != "" {
				texts = append(texts, text)
			}
		}
	}
	return strings.Join(texts, "\n")
}

func extractOpenAIChatWebImagesFromContent(content any) []OpenAIImagesUpload {
	parts, ok := content.([]any)
	if !ok {
		return nil
	}
	uploads := make([]OpenAIImagesUpload, 0, len(parts))
	for _, item := range parts {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		switch strings.TrimSpace(chatWebString(obj["type"])) {
		case "image_url":
			imageURL := ""
			switch value := obj["image_url"].(type) {
			case string:
				imageURL = value
			case map[string]any:
				imageURL = chatWebString(value["url"])
			}
			if upload, ok := openAIChatWebUploadFromImageURL(imageURL, len(uploads)+1); ok {
				uploads = append(uploads, upload)
			}
		case "input_image":
			if upload, ok := openAIChatWebUploadFromImageURL(chatWebString(obj["image_url"]), len(uploads)+1); ok {
				uploads = append(uploads, upload)
			}
		}
	}
	return uploads
}

func openAIChatWebUploadFromImageURL(imageURL string, index int) (OpenAIImagesUpload, bool) {
	imageURL = strings.TrimSpace(imageURL)
	if !strings.HasPrefix(imageURL, "data:") {
		return OpenAIImagesUpload{}, false
	}
	header, data, ok := strings.Cut(imageURL, ",")
	if !ok {
		return OpenAIImagesUpload{}, false
	}
	if !strings.Contains(strings.ToLower(header), ";base64") {
		return OpenAIImagesUpload{}, false
	}
	contentType := strings.TrimPrefix(strings.Split(header, ";")[0], "data:")
	if strings.TrimSpace(contentType) == "" {
		contentType = "image/png"
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return OpenAIImagesUpload{}, false
	}
	ext := ".png"
	if exts, _ := mime.ExtensionsByType(contentType); len(exts) > 0 && strings.TrimSpace(exts[0]) != "" {
		ext = exts[0]
	}
	width, height := parseOpenAIImageDimensions(raw)
	return OpenAIImagesUpload{
		FieldName:   "image",
		FileName:    fmt.Sprintf("image_%d%s", index, ext),
		ContentType: contentType,
		Data:        raw,
		Width:       width,
		Height:      height,
	}, true
}

func parseOpenAIChatWebImageStreamBody(body []byte) *openAIChatWebImageStreamSummary {
	summary := &openAIChatWebImageStreamSummary{}
	for _, rawLine := range strings.Split(string(body), "\n") {
		payload, ok := extractOpenAISSEDataLine(strings.TrimSpace(rawLine))
		if !ok || payload == "" || payload == "[DONE]" {
			continue
		}
		summary.FileIDs = appendUniqueChatWebFileIDs(summary.FileIDs, payload)

		var obj map[string]any
		if err := json.Unmarshal([]byte(payload), &obj); err != nil {
			continue
		}
		if summary.ConversationID == "" {
			summary.ConversationID = strings.TrimSpace(chatWebString(obj["conversation_id"]))
		}
		if summary.ConversationID == "" {
			if nested, ok := obj["v"].(map[string]any); ok {
				summary.ConversationID = strings.TrimSpace(chatWebString(nested["conversation_id"]))
			}
		}
		if message, ok := obj["message"].(map[string]any); ok {
			if content, ok := message["content"].(map[string]any); ok && strings.TrimSpace(chatWebString(content["content_type"])) == "text" {
				if parts, ok := content["parts"].([]any); ok && len(parts) > 0 {
					if text := strings.TrimSpace(chatWebString(parts[0])); text != "" {
						summary.Text += text
					}
				}
			}
		}
	}
	return summary
}

func appendUniqueChatWebFileIDs(existing []string, payload string) []string {
	for _, prefix := range []struct {
		Needle string
		Store  string
	}{
		{Needle: "file-service://", Store: ""},
		{Needle: "sediment://", Store: "sed:"},
	} {
		search := payload
		for {
			index := strings.Index(search, prefix.Needle)
			if index < 0 {
				break
			}
			search = search[index+len(prefix.Needle):]
			var builder strings.Builder
			for _, ch := range search {
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
					builder.WriteRune(ch)
					continue
				}
				break
			}
			if builder.Len() > 0 {
				value := prefix.Store + builder.String()
				if !containsString(existing, value) {
					existing = append(existing, value)
				}
			}
		}
	}
	return existing
}

func extractOpenAIChatWebConversationFileIDs(payload map[string]any) []string {
	mapping, ok := payload["mapping"].(map[string]any)
	if !ok {
		return nil
	}
	var fileIDs []string
	for _, node := range mapping {
		entry, ok := node.(map[string]any)
		if !ok {
			continue
		}
		message, ok := entry["message"].(map[string]any)
		if !ok {
			continue
		}
		author, _ := message["author"].(map[string]any)
		if strings.TrimSpace(chatWebString(author["role"])) != "tool" {
			continue
		}
		metadata, _ := message["metadata"].(map[string]any)
		if strings.TrimSpace(chatWebString(metadata["async_task_type"])) != "image_gen" {
			continue
		}
		content, _ := message["content"].(map[string]any)
		if strings.TrimSpace(chatWebString(content["content_type"])) != "multimodal_text" {
			continue
		}
		parts, _ := content["parts"].([]any)
		for _, part := range parts {
			obj, ok := part.(map[string]any)
			if !ok {
				continue
			}
			pointer := strings.TrimSpace(chatWebString(obj["asset_pointer"]))
			switch {
			case strings.HasPrefix(pointer, "file-service://"):
				value := strings.TrimPrefix(pointer, "file-service://")
				if value != "" && !containsString(fileIDs, value) {
					fileIDs = append(fileIDs, value)
				}
			case strings.HasPrefix(pointer, "sediment://"):
				value := "sed:" + strings.TrimPrefix(pointer, "sediment://")
				if value != "" && !containsString(fileIDs, value) {
					fileIDs = append(fileIDs, value)
				}
			}
		}
	}
	return fileIDs
}

func filterOpenAIChatWebOutputFileIDs(fileIDs []string, uploaded []openAIChatWebUploadedImage) []string {
	if len(fileIDs) == 0 {
		return nil
	}
	inputs := make(map[string]struct{}, len(uploaded))
	for _, item := range uploaded {
		inputs[canonicalOpenAIChatWebFileID(item.FileID)] = struct{}{}
	}
	filtered := make([]string, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		if _, exists := inputs[canonicalOpenAIChatWebFileID(fileID)]; exists {
			continue
		}
		filtered = append(filtered, fileID)
	}
	return filtered
}

func canonicalOpenAIChatWebFileID(fileID string) string {
	return strings.TrimSpace(strings.TrimPrefix(fileID, "sed:"))
}

func cloneOpenAIImageUploads(in []OpenAIImagesUpload) []OpenAIImagesUpload {
	if len(in) == 0 {
		return nil
	}
	out := make([]OpenAIImagesUpload, len(in))
	for i := range in {
		out[i] = in[i]
		if len(in[i].Data) > 0 {
			out[i].Data = append([]byte(nil), in[i].Data...)
		}
	}
	return out
}

func mergeOpenAIImageUsage(dst *OpenAIUsage, src OpenAIUsage) {
	if dst == nil {
		return
	}
	dst.InputTokens += src.InputTokens
	dst.OutputTokens += src.OutputTokens
	dst.CacheCreationInputTokens += src.CacheCreationInputTokens
	dst.CacheReadInputTokens += src.CacheReadInputTokens
}

func containsString(items []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func chatWebString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case json.Number:
		return typed.String()
	case float64:
		return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.0f", typed), ".0"), ".")
	default:
		return fmt.Sprintf("%v", value)
	}
}

func chatWebBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func chatWebInt(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return int(i)
		}
	case string:
		if parsed, err := json.Number(strings.TrimSpace(typed)).Int64(); err == nil {
			return int(parsed)
		}
	}
	return fallback
}

func withOpenAIChatWebImageTimeout(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	timeoutCtx, _ := context.WithTimeout(ctx, openAIChatWebImagesTimeout)
	return timeoutCtx
}
