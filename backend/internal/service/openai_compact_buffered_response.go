package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const (
	defaultOpenAICompactResponseHeaderTimeout = 5 * time.Minute
	openAICompactDisconnectCooldown           = 2 * time.Minute
)

type openAICompactBufferedResult struct {
	body  []byte
	usage OpenAIUsage
	meta  openAICompactResponseMeta
}

type openAICompactResponseMeta struct {
	UpstreamFormat string
	EventCount     int
	FirstEventMs   *int
	TerminalEvent  string
	Partial        bool
}

type openAICompactProtocolError struct {
	message string
}

func (e *openAICompactProtocolError) Error() string {
	return fmt.Sprintf("openai compact protocol error: %s", e.message)
}

func (e *openAICompactProtocolError) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

func resolveOpenAICompactResponseHeaderTimeout(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.Gateway.ResponseHeaderTimeout > 0 {
		return time.Duration(cfg.Gateway.ResponseHeaderTimeout) * time.Second
	}
	return defaultOpenAICompactResponseHeaderTimeout
}

func buildOpenAICompactBufferedFailoverError(account *Account, cause error) *UpstreamFailoverError {
	message := "stream disconnected before completion"
	if cause != nil {
		causeText := sanitizeUpstreamErrorMessage(strings.TrimSpace(cause.Error()))
		switch {
		case causeText == "":
		case strings.Contains(strings.ToLower(causeText), "stream disconnected before completion"):
			message = causeText
		default:
			message = message + ": " + causeText
		}
	}
	payload := []byte(`{"error":{"message":` + strconv.Quote(message) + `}}`)
	return &UpstreamFailoverError{
		StatusCode:           http.StatusBadGateway,
		ResponseBody:         payload,
		TempUnscheduleFor:    openAICompactDisconnectCooldown,
		TempUnscheduleReason: "upstream compact response stream disconnected before completion (auto temp-unschedule 2m)",
		RetryableOnSameAccount: account != nil &&
			account.IsPoolMode(),
	}
}

func writeOpenAICompactProgressHeaders(header http.Header, meta openAICompactResponseMeta) {
	if header == nil || meta.UpstreamFormat == "" {
		return
	}
	header.Set("X-Sub2API-Compact-Upstream-Format", meta.UpstreamFormat)
	if meta.EventCount > 0 {
		header.Set("X-Sub2API-Compact-Event-Count", strconv.Itoa(meta.EventCount))
	}
	if meta.FirstEventMs != nil {
		header.Set("X-Sub2API-Compact-First-Event-Ms", strconv.Itoa(*meta.FirstEventMs))
	}
	if strings.TrimSpace(meta.TerminalEvent) != "" {
		header.Set("X-Sub2API-Compact-Terminal-Event", strings.TrimSpace(meta.TerminalEvent))
	}
	if meta.Partial {
		header.Set("X-Sub2API-Compact-Partial", "true")
	}
}

func (s *OpenAIGatewayService) readOpenAICompactBufferedResponse(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
) (*openAICompactBufferedResult, error) {
	if resp == nil || resp.Body == nil {
		return nil, errors.New("response body is nil")
	}

	requestID := strings.TrimSpace(resp.Header.Get("x-request-id"))
	startedAt := time.Now()
	maxLineSize := defaultMaxLineSize
	if s != nil && s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)

	var (
		rawBody          strings.Builder
		finalBody        []byte
		usage            OpenAIUsage
		terminalErrMsg   string
		loggedFirstEvent bool
	)

	meta := openAICompactResponseMeta{}
	partial := newBufferedResponsesAccumulator("", requestID)

	logCompactProgress := func(level zapcoreLevel, message string, fields ...zap.Field) {
		if !isOpenAIResponsesCompactPath(c) {
			return
		}
		path := ""
		if c != nil && c.Request != nil && c.Request.URL != nil {
			path = strings.TrimSpace(c.Request.URL.Path)
		}
		base := []zap.Field{
			zap.String("component", "service.openai_gateway"),
			zap.String("request_id", requestID),
			zap.String("path", path),
		}
		if account != nil {
			base = append(base, zap.Int64("account_id", account.ID))
		}
		base = append(base, fields...)
		log := logger.FromContext(ctx).With(base...)
		switch level {
		case zapcoreLevelWarn:
			log.Warn(message)
		default:
			log.Info(message)
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if rawBody.Len() > 0 {
			rawBody.WriteByte('\n')
		}
		rawBody.WriteString(line)

		data, ok := extractOpenAISSEDataLine(line)
		if !ok {
			continue
		}

		meta.UpstreamFormat = "sse"
		if data == "" || data == "[DONE]" {
			continue
		}

		meta.EventCount++
		if meta.FirstEventMs == nil {
			ms := int(time.Since(startedAt).Milliseconds())
			meta.FirstEventMs = &ms
		}

		eventType := strings.TrimSpace(gjson.Get(data, "type").String())
		if eventType == "" {
			continue
		}
		if openAIStreamEventIsTerminal(data) {
			meta.TerminalEvent = eventType
		}

		if !loggedFirstEvent {
			loggedFirstEvent = true
			fields := []zap.Field{zap.String("event_type", eventType)}
			if meta.FirstEventMs != nil {
				fields = append(fields, zap.Int("first_event_ms", *meta.FirstEventMs))
			}
			logCompactProgress(zapcoreLevelInfo, "codex.remote_compact.upstream_started", fields...)
		}

		s.parseSSEUsageBytes([]byte(data), &usage)

		var event apicompat.ResponsesStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err == nil {
			partial.applyEvent(&event)
		}

		switch eventType {
		case "response.failed":
			terminalErrMsg = extractOpenAISSEErrorMessage([]byte(data))
		case "response.completed", "response.done", "response.incomplete", "response.cancelled", "response.canceled":
			if response := gjson.Get(data, "response"); response.Exists() && response.Type == gjson.JSON && strings.TrimSpace(response.Raw) != "" {
				finalBody = []byte(response.Raw)
				if eventType != "response.completed" && eventType != "response.done" {
					meta.Partial = true
				}
			}
		}
	}

	scanErr := scanner.Err()
	if finalBody != nil {
		if scanErr != nil && !errors.Is(scanErr, context.Canceled) && !errors.Is(scanErr, context.DeadlineExceeded) {
			logger.FromContext(ctx).With(
				zap.String("component", "service.openai_gateway"),
				zap.String("request_id", requestID),
			).Info("OpenAI compact upstream scan ended after terminal event", zap.Error(scanErr))
		}
		return &openAICompactBufferedResult{
			body:  finalBody,
			usage: usage,
			meta:  meta,
		}, nil
	}

	if terminalErrMsg != "" {
		return nil, &openAICompactProtocolError{message: terminalErrMsg}
	}

	if meta.UpstreamFormat == "" {
		body := []byte(rawBody.String())
		if len(body) == 0 || !gjson.ValidBytes(body) {
			return nil, fmt.Errorf("parse response: invalid json response")
		}
		if parsedUsage, ok := extractOpenAIUsageFromJSONBytes(body); ok {
			usage = parsedUsage
		}
		return &openAICompactBufferedResult{
			body:  body,
			usage: usage,
			meta: openAICompactResponseMeta{
				UpstreamFormat: "json",
			},
		}, nil
	}

	if partial.hasUsefulOutput() {
		meta.Partial = true
		if meta.TerminalEvent == "" {
			meta.TerminalEvent = "stream_disconnected"
		}
		body, err := json.Marshal(partial.responseSnapshot())
		if err != nil {
			return nil, fmt.Errorf("marshal compact partial response: %w", err)
		}
		if parsedUsage, ok := extractOpenAIUsageFromJSONBytes(body); ok {
			usage = parsedUsage
		}
		fields := []zap.Field{
			zap.Int("event_count", meta.EventCount),
			zap.String("terminal_event", meta.TerminalEvent),
		}
		if meta.FirstEventMs != nil {
			fields = append(fields, zap.Int("first_event_ms", *meta.FirstEventMs))
		}
		logCompactProgress(zapcoreLevelWarn, "codex.remote_compact.partial_fallback", fields...)
		return &openAICompactBufferedResult{
			body:  body,
			usage: usage,
			meta:  meta,
		}, nil
	}

	if scanErr == nil {
		scanErr = io.ErrUnexpectedEOF
	}
	return nil, buildOpenAICompactBufferedFailoverError(account, scanErr)
}

type zapcoreLevel int

const (
	zapcoreLevelInfo zapcoreLevel = iota
	zapcoreLevelWarn
)
