package service

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

func buildOpenAIStreamFailedEventPayload(requestID, model, code, message string, usage *OpenAIUsage) []byte {
	respID := strings.TrimSpace(requestID)
	if respID == "" {
		respID = "resp_partial"
	}

	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "stream disconnected before completion"
	}

	code = strings.TrimSpace(code)
	if code == "" {
		code = "upstream_error"
	}

	response := &apicompat.ResponsesResponse{
		ID:     respID,
		Object: "response",
		Model:  strings.TrimSpace(model),
		Status: "failed",
		Output: []apicompat.ResponsesOutput{},
		Error: &apicompat.ResponsesError{
			Code:    code,
			Message: message,
		},
	}

	if usage != nil {
		response.Usage = &apicompat.ResponsesUsage{
			InputTokens:  usage.InputTokens,
			OutputTokens: usage.OutputTokens,
			TotalTokens:  usage.InputTokens + usage.OutputTokens,
		}
		if usage.CacheReadInputTokens > 0 {
			response.Usage.InputTokensDetails = &apicompat.ResponsesInputTokensDetails{
				CachedTokens: usage.CacheReadInputTokens,
			}
		}
	}

	payload, err := json.Marshal(&apicompat.ResponsesStreamEvent{
		Type:     "response.failed",
		Response: response,
	})
	if err == nil {
		return payload
	}

	fallback := `{"type":"response.failed","response":{"id":"` + respID + `","object":"response","model":"` + strings.TrimSpace(model) + `","status":"failed","output":[],"error":{"code":"` + code + `","message":` + strconvQuoteASCII(message) + `}}}`
	return []byte(fallback)
}

func strconvQuoteASCII(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}
