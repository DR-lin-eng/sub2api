package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"strings"
	"time"

	kiropkg "github.com/Wei-Shaw/sub2api/internal/pkg/kiro"
	"github.com/gin-gonic/gin"
)

const (
	kiroPreludeSize = 12
	kiroMinMsgSize  = kiroPreludeSize + 4
)

type KiroGatewayService struct {
	httpUpstream      HTTPUpstream
	tokenProvider     *KiroTokenProvider
	tokenUsageService *KiroUsageService
}

func NewKiroGatewayService(
	httpUpstream HTTPUpstream,
	tokenProvider *KiroTokenProvider,
	tokenUsageService *KiroUsageService,
) *KiroGatewayService {
	return &KiroGatewayService{
		httpUpstream:      httpUpstream,
		tokenProvider:     tokenProvider,
		tokenUsageService: tokenUsageService,
	}
}

func (s *KiroGatewayService) Forward(ctx context.Context, c *gin.Context, account *Account, parsed *ParsedRequest) (*ForwardResult, error) {
	if account == nil || parsed == nil {
		return nil, fmt.Errorf("invalid kiro forward args")
	}
	converted, err := kiropkg.ConvertAnthropicRequest(parsed.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": gin.H{"type": "invalid_request_error", "message": err.Error()},
		})
		return nil, err
	}

	accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": "Failed to refresh Kiro token"},
		})
		return nil, err
	}

	req, err := s.buildRequest(ctx, account, converted.Body, accessToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": "Failed to build Kiro upstream request"},
		})
		return nil, err
	}

	start := time.Now()
	resp, err := s.httpUpstream.Do(req, accountProxyURL(account), account.ID, account.Concurrency)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": "Kiro upstream request failed"},
		})
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if shouldKiroFailover(resp.StatusCode) {
			return nil, &UpstreamFailoverError{
				StatusCode:   resp.StatusCode,
				ResponseBody: body,
			}
		}
		c.JSON(http.StatusBadGateway, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": fmt.Sprintf("Kiro upstream returned %d", resp.StatusCode)},
		})
		return nil, fmt.Errorf("kiro upstream returned %d", resp.StatusCode)
	}

	inputTokens := kiropkg.EstimateInputTokens(parsed.Body)
	if parsed.Stream {
		return s.forwardStream(c, resp, parsed, converted, inputTokens, start)
	}
	return s.forwardNonStream(c, resp, parsed, converted, inputTokens, start)
}

func (s *KiroGatewayService) ForwardCountTokens(ctx context.Context, c *gin.Context, account *Account, parsed *ParsedRequest) error {
	inputTokens := kiropkg.EstimateInputTokens(parsed.Body)
	c.JSON(http.StatusOK, gin.H{"input_tokens": inputTokens})
	return nil
}

func (s *KiroGatewayService) buildRequest(ctx context.Context, account *Account, body []byte, accessToken string) (*http.Request, error) {
	url := fmt.Sprintf("https://q.%s.amazonaws.com/generateAssistantResponse", KiroRegion(account))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	machineID := kiropkg.GenerateMachineID(account.GetCredential("machine_id"), "", account.GetCredential("refresh_token"))
	host := fmt.Sprintf("q.%s.amazonaws.com", KiroRegion(account))
	kiroVersion := KiroVersion(account)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("host", host)
	req.Header.Set("Connection", "close")
	req.Header.Set("x-amzn-codewhisperer-optout", "true")
	req.Header.Set("x-amzn-kiro-agent-mode", "vibe")
	req.Header.Set("x-amz-user-agent", fmt.Sprintf("aws-sdk-js/1.0.27 KiroIDE-%s-%s", kiroVersion, machineID))
	req.Header.Set("User-Agent", fmt.Sprintf("aws-sdk-js/1.0.27 ua/2.1 os/%s lang/js md/nodejs#%s api/codewhispererstreaming#1.0.27 m/E KiroIDE-%s-%s", KiroSystemVersion(account), KiroNodeVersion(account), kiroVersion, machineID))
	req.Header.Set("amz-sdk-invocation-id", generateRequestID())
	req.Header.Set("amz-sdk-request", "attempt=1; max=3")
	return req, nil
}

func (s *KiroGatewayService) forwardNonStream(c *gin.Context, resp *http.Response, parsed *ParsedRequest, converted *kiropkg.ConvertResult, inputTokens int, start time.Time) (*ForwardResult, error) {
	frames, err := readAllKiroFrames(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": "Failed to decode Kiro response"},
		})
		return nil, err
	}

	textBuilder := strings.Builder{}
	toolUses := make([]map[string]any, 0)
	toolBuffers := make(map[string]*kiroToolState)
	stopReason := "end_turn"
	contextInputTokens := 0

	for _, frame := range frames {
		switch frame.EventType {
		case "assistantResponseEvent":
			if content := stringField(frame.Payload, "content"); content != "" {
				textBuilder.WriteString(content)
			}
		case "toolUseEvent":
			state := ensureKiroToolState(toolBuffers, stringField(frame.Payload, "toolUseId"), stringField(frame.Payload, "name"))
			state.InputBuilder.WriteString(stringField(frame.Payload, "input"))
			if booleanField(frame.Payload, "stop") {
				input := map[string]any{}
				_ = json.Unmarshal([]byte(state.InputBuilder.String()), &input)
				toolUses = append(toolUses, map[string]any{
					"type":  "tool_use",
					"id":    state.ToolUseID,
					"name":  state.Name,
					"input": input,
				})
				stopReason = "tool_use"
			}
		case "contextUsageEvent":
			contextInputTokens = contextUsageToTokens(numberField(frame.Payload, "contextUsagePercentage"))
		}
	}

	content := make([]map[string]any, 0)
	if text := textBuilder.String(); text != "" {
		content = append(content, map[string]any{"type": "text", "text": text})
	}
	content = append(content, toolUses...)
	if contextInputTokens > 0 {
		inputTokens = contextInputTokens
	}
	outputTokens := kiropkg.EstimateOutputTokens(textBuilder.String())

	c.JSON(http.StatusOK, gin.H{
		"id":            "msg_" + strings.ReplaceAll(generateRequestID(), "-", ""),
		"type":          "message",
		"role":          "assistant",
		"content":       content,
		"model":         parsed.Model,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": gin.H{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		},
	})

	return &ForwardResult{
		RequestID:     resp.Header.Get("x-amzn-requestid"),
		Model:         parsed.Model,
		UpstreamModel: converted.Model,
		Stream:        false,
		Duration:      time.Since(start),
		Usage: ClaudeUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		},
	}, nil
}

func (s *KiroGatewayService) forwardStream(c *gin.Context, resp *http.Response, parsed *ParsedRequest, converted *kiropkg.ConvertResult, inputTokens int, start time.Time) (*ForwardResult, error) {
	writer := c.Writer
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)
	writer.Flush()

	msgID := "msg_" + strings.ReplaceAll(generateRequestID(), "-", "")
	initial := map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            msgID,
			"type":          "message",
			"role":          "assistant",
			"model":         parsed.Model,
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  inputTokens,
				"output_tokens": 0,
			},
		},
	}
	if err := writeSSEEvent(writer, "message_start", initial); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(resp.Body)
	buffer := make([]byte, 0, 64*1024)
	textStarted := false
	textBlockIndex := 0
	nextBlockIndex := 1
	toolStates := make(map[string]*kiroToolState)
	var firstTokenMs *int
	var outputBuilder strings.Builder
	stopReason := "end_turn"
	contextInputTokens := 0

	for {
		chunk := make([]byte, 4096)
		n, readErr := reader.Read(chunk)
		if n > 0 {
			buffer = append(buffer, chunk[:n]...)
			for {
				frame, consumed, ok, err := parseKiroFrame(buffer)
				if err != nil {
					return nil, err
				}
				if !ok {
					break
				}
				buffer = buffer[consumed:]

				switch frame.EventType {
				case "assistantResponseEvent":
					content := stringField(frame.Payload, "content")
					if content == "" {
						continue
					}
					if firstTokenMs == nil {
						v := int(time.Since(start).Milliseconds())
						firstTokenMs = &v
					}
					if !textStarted {
						textStarted = true
						if err := writeSSEEvent(writer, "content_block_start", map[string]any{
							"type":  "content_block_start",
							"index": textBlockIndex,
							"content_block": map[string]any{
								"type": "text",
								"text": "",
							},
						}); err != nil {
							return nil, err
						}
					}
					outputBuilder.WriteString(content)
					if err := writeSSEEvent(writer, "content_block_delta", map[string]any{
						"type":  "content_block_delta",
						"index": textBlockIndex,
						"delta": map[string]any{
							"type": "text_delta",
							"text": content,
						},
					}); err != nil {
						return nil, err
					}
				case "toolUseEvent":
					toolUseID := stringField(frame.Payload, "toolUseId")
					state := ensureKiroToolState(toolStates, toolUseID, stringField(frame.Payload, "name"))
					if !state.Started {
						state.Started = true
						state.BlockIndex = nextBlockIndex
						nextBlockIndex++
						if err := writeSSEEvent(writer, "content_block_start", map[string]any{
							"type":  "content_block_start",
							"index": state.BlockIndex,
							"content_block": map[string]any{
								"type":  "tool_use",
								"id":    state.ToolUseID,
								"name":  state.Name,
								"input": map[string]any{},
							},
						}); err != nil {
							return nil, err
						}
					}
					inputChunk := stringField(frame.Payload, "input")
					if inputChunk != "" {
						state.InputBuilder.WriteString(inputChunk)
						if err := writeSSEEvent(writer, "content_block_delta", map[string]any{
							"type":  "content_block_delta",
							"index": state.BlockIndex,
							"delta": map[string]any{
								"type":         "input_json_delta",
								"partial_json": inputChunk,
							},
						}); err != nil {
							return nil, err
						}
					}
					if booleanField(frame.Payload, "stop") {
						stopReason = "tool_use"
						if err := writeSSEEvent(writer, "content_block_stop", map[string]any{
							"type":  "content_block_stop",
							"index": state.BlockIndex,
						}); err != nil {
							return nil, err
						}
					}
				case "contextUsageEvent":
					contextInputTokens = contextUsageToTokens(numberField(frame.Payload, "contextUsagePercentage"))
				case "exception":
					if err := writeSSEEvent(writer, "message_delta", map[string]any{
						"type":  "message_delta",
						"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil},
						"usage": map[string]any{"input_tokens": inputTokens, "output_tokens": kiropkg.EstimateOutputTokens(outputBuilder.String())},
					}); err != nil {
						return nil, err
					}
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, readErr
		}
	}

	if textStarted {
		if err := writeSSEEvent(writer, "content_block_stop", map[string]any{
			"type":  "content_block_stop",
			"index": textBlockIndex,
		}); err != nil {
			return nil, err
		}
	}

	if contextInputTokens > 0 {
		inputTokens = contextInputTokens
	}
	outputTokens := kiropkg.EstimateOutputTokens(outputBuilder.String())
	if err := writeSSEEvent(writer, "message_delta", map[string]any{
		"type":  "message_delta",
		"delta": map[string]any{"stop_reason": stopReason, "stop_sequence": nil},
		"usage": map[string]any{"input_tokens": inputTokens, "output_tokens": outputTokens},
	}); err != nil {
		return nil, err
	}
	if err := writeSSEEvent(writer, "message_stop", map[string]any{"type": "message_stop"}); err != nil {
		return nil, err
	}

	return &ForwardResult{
		RequestID:     resp.Header.Get("x-amzn-requestid"),
		Model:         parsed.Model,
		UpstreamModel: converted.Model,
		Stream:        true,
		Duration:      time.Since(start),
		FirstTokenMs:  firstTokenMs,
		Usage: ClaudeUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		},
	}, nil
}

type kiroFrame struct {
	MessageType string
	EventType   string
	Payload     map[string]any
}

type kiroToolState struct {
	ToolUseID    string
	Name         string
	BlockIndex   int
	Started      bool
	InputBuilder strings.Builder
}

func ensureKiroToolState(states map[string]*kiroToolState, toolUseID, name string) *kiroToolState {
	if toolUseID == "" {
		toolUseID = generateRequestID()
	}
	if state, ok := states[toolUseID]; ok {
		if state.Name == "" {
			state.Name = name
		}
		return state
	}
	state := &kiroToolState{ToolUseID: toolUseID, Name: name}
	states[toolUseID] = state
	return state
}

func parseKiroFrame(buffer []byte) (*kiroFrame, int, bool, error) {
	if len(buffer) < kiroPreludeSize {
		return nil, 0, false, nil
	}

	totalLength := int(binary.BigEndian.Uint32(buffer[0:4]))
	headerLength := int(binary.BigEndian.Uint32(buffer[4:8]))
	preludeCRC := binary.BigEndian.Uint32(buffer[8:12])
	if totalLength < kiroMinMsgSize || headerLength < 0 {
		return nil, 0, false, fmt.Errorf("invalid kiro frame size")
	}
	if len(buffer) < totalLength {
		return nil, 0, false, nil
	}
	if crc32.ChecksumIEEE(buffer[:8]) != preludeCRC {
		return nil, 0, false, fmt.Errorf("kiro prelude crc mismatch")
	}

	messageCRC := binary.BigEndian.Uint32(buffer[totalLength-4 : totalLength])
	if crc32.ChecksumIEEE(buffer[:totalLength-4]) != messageCRC {
		return nil, 0, false, fmt.Errorf("kiro message crc mismatch")
	}

	headersBytes := buffer[kiroPreludeSize : kiroPreludeSize+headerLength]
	headers, err := parseKiroHeaders(headersBytes)
	if err != nil {
		return nil, 0, false, err
	}

	payloadBytes := buffer[kiroPreludeSize+headerLength : totalLength-4]
	var payload map[string]any
	_ = json.Unmarshal(payloadBytes, &payload)

	frame := &kiroFrame{
		MessageType: headers[":message-type"],
		EventType:   headers[":event-type"],
		Payload:     payload,
	}
	if frame.MessageType == "" {
		if _, ok := headers[":exception-type"]; ok {
			frame.MessageType = "exception"
		} else if _, ok := headers[":error-code"]; ok {
			frame.MessageType = "error"
		}
	}
	if frame.MessageType == "" {
		frame.MessageType = "event"
	}

	return frame, totalLength, true, nil
}

func parseKiroHeaders(data []byte) (map[string]string, error) {
	headers := make(map[string]string)
	for offset := 0; offset < len(data); {
		nameLen := int(data[offset])
		offset++
		if offset+nameLen > len(data) {
			return nil, fmt.Errorf("kiro header out of bounds")
		}
		name := string(data[offset : offset+nameLen])
		offset += nameLen
		if offset >= len(data) {
			return nil, fmt.Errorf("kiro header missing type")
		}
		valueType := data[offset]
		offset++
		switch valueType {
		case 7:
			if offset+2 > len(data) {
				return nil, fmt.Errorf("kiro header string length missing")
			}
			valueLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2
			if offset+valueLen > len(data) {
				return nil, fmt.Errorf("kiro header string out of bounds")
			}
			headers[name] = string(data[offset : offset+valueLen])
			offset += valueLen
		case 0:
			headers[name] = "true"
		case 1:
			headers[name] = "false"
		case 2:
			offset++
		case 3:
			offset += 2
		case 4:
			offset += 4
		case 5, 8:
			offset += 8
		case 6:
			if offset+2 > len(data) {
				return nil, fmt.Errorf("kiro header byte array length missing")
			}
			valueLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2 + valueLen
		case 9:
			offset += 16
		default:
			return nil, fmt.Errorf("unknown kiro header type %d", valueType)
		}
		if offset > len(data) {
			return nil, fmt.Errorf("kiro header parse overflow")
		}
	}
	return headers, nil
}

func writeSSEEvent(w gin.ResponseWriter, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte("event: " + event + "\n")); err != nil {
		return err
	}
	if _, err := w.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
		return err
	}
	w.Flush()
	return nil
}

func readAllKiroFrames(body io.Reader) ([]*kiroFrame, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	frames := make([]*kiroFrame, 0)
	for len(raw) > 0 {
		frame, consumed, ok, err := parseKiroFrame(raw)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		frames = append(frames, frame)
		raw = raw[consumed:]
	}
	return frames, nil
}

func shouldKiroFailover(statusCode int) bool {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
		return true
	default:
		return statusCode >= 500
	}
}

func accountProxyURL(account *Account) string {
	if account != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}
	return ""
}

func stringField(obj map[string]any, key string) string {
	if obj == nil {
		return ""
	}
	if value, ok := obj[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func booleanField(obj map[string]any, key string) bool {
	if obj == nil {
		return false
	}
	if value, ok := obj[key].(bool); ok {
		return value
	}
	return false
}

func numberField(obj map[string]any, key string) float64 {
	if obj == nil {
		return 0
	}
	if value, ok := obj[key].(float64); ok {
		return value
	}
	return 0
}

func contextUsageToTokens(percentage float64) int {
	if percentage <= 0 {
		return 0
	}
	return int((percentage * 1000000.0) / 100.0)
}
