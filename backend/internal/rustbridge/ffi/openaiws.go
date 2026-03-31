package ffi

import (
	"strings"

	"github.com/tidwall/gjson"
)

type OpenAIWSEventEnvelope struct {
	EventType   string
	ResponseID  string
	ResponseRaw string
}

type OpenAIWSUsageFields struct {
	InputTokens       int
	OutputTokens      int
	CachedInputTokens int
}

type OpenAIWSRequestPayloadSummary struct {
	EventType             string
	Model                 string
	PromptCacheKey        string
	PreviousResponseID    string
	StreamExists          bool
	Stream                bool
	HasFunctionCallOutput bool
}

type OpenAIWSFrameSummary struct {
	EventType         string
	ResponseID        string
	ResponseRaw       string
	Code              string
	ErrType           string
	Message           string
	InputTokens       int
	OutputTokens      int
	CachedInputTokens int
	IsTerminalEvent   bool
	IsTokenEvent      bool
	HasToolCalls      bool
}

type OpenAISSEBodySummary struct {
	TerminalEventType string
	TerminalPayload   string
	FinalResponseRaw  string
	InputTokens       int
	OutputTokens      int
	CachedInputTokens int
	HasTerminalEvent  bool
	HasFinalResponse  bool
}

type OpenAIWSErrorFields struct {
	Code    string
	ErrType string
	Message string
}

func IsOpenAIWSTerminalEvent(eventType string) bool {
	normalized := strings.TrimSpace(eventType)
	if normalized == "" {
		return false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if result, ok := lib.CallOpenAIWSIsTerminalEvent(normalized); ok {
				recordMetric(ffiMetricEventPredicate, true)
				return result
			}
		}
	}
	recordMetric(ffiMetricEventPredicate, false)
	switch normalized {
	case "response.completed", "response.done", "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
		return true
	default:
		return false
	}
}

func IsOpenAIWSTokenEvent(eventType string) bool {
	normalized := strings.TrimSpace(eventType)
	if normalized == "" {
		return false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if result, ok := lib.CallOpenAIWSIsTokenEvent(normalized); ok {
				recordMetric(ffiMetricEventPredicate, true)
				return result
			}
		}
	}
	recordMetric(ffiMetricEventPredicate, false)
	switch normalized {
	case "response.created", "response.in_progress", "response.output_item.added", "response.output_item.done":
		return false
	}
	if strings.Contains(normalized, ".delta") {
		return true
	}
	if strings.HasPrefix(normalized, "response.output_text") {
		return true
	}
	if strings.HasPrefix(normalized, "response.output") {
		return true
	}
	return normalized == "response.completed" || normalized == "response.done"
}

func OpenAIWSMessageLikelyContainsToolCalls(message []byte) bool {
	if len(message) == 0 {
		return false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if result, ok := lib.CallOpenAIWSHasToolCalls(message); ok {
				recordMetric(ffiMetricEventPredicate, true)
				return result
			}
		}
	}
	recordMetric(ffiMetricEventPredicate, false)
	return strings.Contains(string(message), `"tool_calls"`) ||
		strings.Contains(string(message), `"tool_call"`) ||
		strings.Contains(string(message), `"function_call"`)
}

func ReplaceOpenAIWSMessageModel(message []byte, fromModel, toModel string) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if updated, ok := lib.CallOpenAIWSReplaceModel(message, fromModel, toModel); ok {
				recordMetric(ffiMetricPayloadMutate, true)
				return updated, true
			}
		}
	}
	recordMetric(ffiMetricPayloadMutate, false)
	return nil, false
}

func DropOpenAIWSPreviousResponseID(message []byte) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if updated, ok := lib.CallOpenAIWSDropPreviousResponseID(message); ok {
				recordMetric(ffiMetricPayloadMutate, true)
				return updated, true
			}
		}
	}
	recordMetric(ffiMetricPayloadMutate, false)
	return nil, false
}

func SetOpenAIWSPreviousResponseID(message []byte, previousResponseID string) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if updated, ok := lib.CallOpenAIWSSetPreviousResponseID(message, previousResponseID); ok {
				recordMetric(ffiMetricPayloadMutate, true)
				return updated, true
			}
		}
	}
	recordMetric(ffiMetricPayloadMutate, false)
	return nil, false
}

func SetOpenAIWSRequestType(message []byte, eventType string) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if updated, ok := lib.CallOpenAIWSSetRequestType(message, eventType); ok {
				recordMetric(ffiMetricPayloadMutate, true)
				return updated, true
			}
		}
	}
	recordMetric(ffiMetricPayloadMutate, false)
	return nil, false
}

func SetOpenAIWSTurnMetadata(message []byte, turnMetadata string) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if updated, ok := lib.CallOpenAIWSSetTurnMetadata(message, turnMetadata); ok {
				recordMetric(ffiMetricPayloadMutate, true)
				return updated, true
			}
		}
	}
	recordMetric(ffiMetricPayloadMutate, false)
	return nil, false
}

func SetOpenAIWSInputSequence(message []byte, inputSequenceJSON []byte) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if updated, ok := lib.CallOpenAIWSSetInputSequence(message, inputSequenceJSON); ok {
				recordMetric(ffiMetricPayloadMutate, true)
				return updated, true
			}
		}
	}
	recordMetric(ffiMetricPayloadMutate, false)
	return nil, false
}

func NormalizeOpenAIWSPayloadWithoutInputAndPreviousResponseID(message []byte) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if updated, ok := lib.CallOpenAIWSNormalizePayloadWithoutInputAndPreviousResponseID(message); ok {
				recordMetric(ffiMetricPayloadMutate, true)
				return updated, true
			}
		}
	}
	recordMetric(ffiMetricPayloadMutate, false)
	return nil, false
}

func BuildOpenAIWSReplayInputSequence(previousFullInputJSON []byte, previousFullInputExists bool, currentPayload []byte, hasPreviousResponseID bool) ([]byte, bool, bool) {
	if len(currentPayload) == 0 {
		return nil, false, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if updated, exists, ok := lib.CallOpenAIWSBuildReplayInputSequence(previousFullInputJSON, previousFullInputExists, currentPayload, hasPreviousResponseID); ok {
				recordMetric(ffiMetricPayloadMutate, true)
				return updated, exists, true
			}
		}
	}
	recordMetric(ffiMetricPayloadMutate, false)
	return nil, false, false
}

func ParseOpenAIWSFrameSummary(message []byte) OpenAIWSFrameSummary {
	if len(message) == 0 {
		return OpenAIWSFrameSummary{}
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if eventType, responseID, responseRaw, code, errType, messageText, inputTokens, outputTokens, cachedTokens, isTerminal, isToken, hasToolCalls, ok :=
				lib.CallOpenAIWSParseFrameSummary(message); ok {
				recordMetric(ffiMetricEventParse, true)
				return OpenAIWSFrameSummary{
					EventType:         strings.TrimSpace(eventType),
					ResponseID:        strings.TrimSpace(responseID),
					ResponseRaw:       responseRaw,
					Code:              strings.TrimSpace(code),
					ErrType:           strings.TrimSpace(errType),
					Message:           strings.TrimSpace(messageText),
					InputTokens:       inputTokens,
					OutputTokens:      outputTokens,
					CachedInputTokens: cachedTokens,
					IsTerminalEvent:   isTerminal,
					IsTokenEvent:      isToken,
					HasToolCalls:      hasToolCalls,
				}
			}
		}
	}
	recordMetric(ffiMetricEventParse, false)
	envelope := ParseOpenAIWSEventEnvelope(message)
	usage := ParseOpenAIWSUsageFields(message)
	errFields := ParseOpenAIWSErrorFields(message)
	return OpenAIWSFrameSummary{
		EventType:         envelope.EventType,
		ResponseID:        envelope.ResponseID,
		ResponseRaw:       envelope.ResponseRaw,
		Code:              errFields.Code,
		ErrType:           errFields.ErrType,
		Message:           errFields.Message,
		InputTokens:       usage.InputTokens,
		OutputTokens:      usage.OutputTokens,
		CachedInputTokens: usage.CachedInputTokens,
		IsTerminalEvent:   IsOpenAIWSTerminalEvent(envelope.EventType),
		IsTokenEvent:      IsOpenAIWSTokenEvent(envelope.EventType),
		HasToolCalls:      OpenAIWSMessageLikelyContainsToolCalls(message),
	}
}

func ParseOpenAISSEBodySummary(body []byte) OpenAISSEBodySummary {
	if len(body) == 0 {
		return OpenAISSEBodySummary{}
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if terminalEventType, terminalPayload, finalResponseRaw, inputTokens, outputTokens, cachedTokens, hasTerminalEvent, hasFinalResponse, ok :=
				lib.CallParseOpenAISSEBodySummary(body); ok {
				recordMetric(ffiMetricSSEBodySummary, true)
				return OpenAISSEBodySummary{
					TerminalEventType: strings.TrimSpace(terminalEventType),
					TerminalPayload:   terminalPayload,
					FinalResponseRaw:  finalResponseRaw,
					InputTokens:       inputTokens,
					OutputTokens:      outputTokens,
					CachedInputTokens: cachedTokens,
					HasTerminalEvent:  hasTerminalEvent,
					HasFinalResponse:  hasFinalResponse,
				}
			}
		}
	}
	recordMetric(ffiMetricSSEBodySummary, false)
	return OpenAISSEBodySummary{}
}

func BuildSSEDataFrame(message []byte) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if frame, ok := lib.CallBuildSSEDataFrame(message); ok {
				recordMetric(ffiMetricFraming, true)
				return frame, true
			}
		}
	}
	recordMetric(ffiMetricFraming, false)
	return nil, false
}

func CorrectOpenAIToolCalls(message []byte) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if corrected, ok := lib.CallCorrectOpenAIToolCalls(message); ok {
				recordMetric(ffiMetricToolCorrection, true)
				return corrected, true
			}
		}
	}
	recordMetric(ffiMetricToolCorrection, false)
	return nil, false
}

func RewriteOpenAIWSMessageForClient(message []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if rewritten, ok := lib.CallRewriteOpenAIWSMessageForClient(message, fromModel, toModel, applyToolCorrection); ok {
				recordMetric(ffiMetricPayloadMutate, true)
				return rewritten, true
			}
		}
	}
	recordMetric(ffiMetricPayloadMutate, false)
	return nil, false
}

func RewriteOpenAISSELineForClient(line []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	if len(line) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if rewritten, ok := lib.CallRewriteOpenAISSELineForClient(line, fromModel, toModel, applyToolCorrection); ok {
				recordMetric(ffiMetricFraming, true)
				return rewritten, true
			}
		}
	}
	recordMetric(ffiMetricFraming, false)
	return nil, false
}

func RewriteOpenAISSEBodyForClient(body []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	if len(body) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if rewritten, ok := lib.CallRewriteOpenAISSEBodyForClient(body, fromModel, toModel, applyToolCorrection); ok {
				recordMetric(ffiMetricFraming, true)
				return rewritten, true
			}
		}
	}
	recordMetric(ffiMetricFraming, false)
	return nil, false
}

func RewriteOpenAIWSMessageToSSEFrameForClient(message []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	if len(message) == 0 {
		return nil, false
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if rewritten, ok := lib.CallRewriteOpenAIWSMessageToSSEFrameForClient(message, fromModel, toModel, applyToolCorrection); ok {
				recordMetric(ffiMetricFraming, true)
				return rewritten, true
			}
		}
	}
	recordMetric(ffiMetricFraming, false)
	return nil, false
}

func ParseOpenAIWSRequestPayloadSummary(message []byte) OpenAIWSRequestPayloadSummary {
	if len(message) == 0 {
		return OpenAIWSRequestPayloadSummary{}
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if eventType, model, promptCacheKey, previousResponseID, streamExists, stream, hasFunctionCallOutput, ok :=
				lib.CallOpenAIWSParseRequestPayloadSummary(message); ok {
				recordMetric(ffiMetricEventParse, true)
				return OpenAIWSRequestPayloadSummary{
					EventType:             strings.TrimSpace(eventType),
					Model:                 strings.TrimSpace(model),
					PromptCacheKey:        strings.TrimSpace(promptCacheKey),
					PreviousResponseID:    strings.TrimSpace(previousResponseID),
					StreamExists:          streamExists,
					Stream:                stream,
					HasFunctionCallOutput: hasFunctionCallOutput,
				}
			}
		}
	}
	recordMetric(ffiMetricEventParse, false)
	values := gjson.GetManyBytes(message, "type", "model", "prompt_cache_key", "previous_response_id", "stream", `input.#(type=="function_call_output")`)
	stream := values[4]
	return OpenAIWSRequestPayloadSummary{
		EventType:             strings.TrimSpace(values[0].String()),
		Model:                 strings.TrimSpace(values[1].String()),
		PromptCacheKey:        strings.TrimSpace(values[2].String()),
		PreviousResponseID:    strings.TrimSpace(values[3].String()),
		StreamExists:          stream.Exists() && (stream.Type == gjson.True || stream.Type == gjson.False),
		Stream:                stream.Bool(),
		HasFunctionCallOutput: values[5].Exists(),
	}
}

// ParseOpenAIWSEventEnvelope is the stable seam for future Rust websocket-event parsing.
func ParseOpenAIWSEventEnvelope(message []byte) OpenAIWSEventEnvelope {
	if len(message) == 0 {
		return OpenAIWSEventEnvelope{}
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if eventType, responseID, responseRaw, ok := lib.CallOpenAIWSParseEnvelope(message); ok {
				recordMetric(ffiMetricEventParse, true)
				return OpenAIWSEventEnvelope{
					EventType:   strings.TrimSpace(eventType),
					ResponseID:  strings.TrimSpace(responseID),
					ResponseRaw: responseRaw,
				}
			}
		}
	}
	recordMetric(ffiMetricEventParse, false)
	values := gjson.GetManyBytes(message, "type", "response.id", "id", "response")
	responseID := strings.TrimSpace(values[1].String())
	if responseID == "" {
		responseID = strings.TrimSpace(values[2].String())
	}
	responseRaw := ""
	if values[3].Exists() && values[3].Type == gjson.JSON {
		responseRaw = values[3].Raw
	}
	return OpenAIWSEventEnvelope{
		EventType:   strings.TrimSpace(values[0].String()),
		ResponseID:  responseID,
		ResponseRaw: responseRaw,
	}
}

// ParseOpenAIWSUsageFields is the stable seam for future Rust completed-event usage parsing.
func ParseOpenAIWSUsageFields(message []byte) OpenAIWSUsageFields {
	if len(message) == 0 {
		return OpenAIWSUsageFields{}
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if inputTokens, outputTokens, cachedTokens, ok := lib.CallOpenAIWSParseUsage(message); ok {
				recordMetric(ffiMetricEventParse, true)
				return OpenAIWSUsageFields{
					InputTokens:       inputTokens,
					OutputTokens:      outputTokens,
					CachedInputTokens: cachedTokens,
				}
			}
		}
	}
	recordMetric(ffiMetricEventParse, false)
	values := gjson.GetManyBytes(
		message,
		"response.usage.input_tokens",
		"response.usage.output_tokens",
		"response.usage.input_tokens_details.cached_tokens",
	)
	return OpenAIWSUsageFields{
		InputTokens:       int(values[0].Int()),
		OutputTokens:      int(values[1].Int()),
		CachedInputTokens: int(values[2].Int()),
	}
}

// ParseOpenAIWSErrorFields is the stable seam for future Rust websocket-error parsing.
func ParseOpenAIWSErrorFields(message []byte) OpenAIWSErrorFields {
	if len(message) == 0 {
		return OpenAIWSErrorFields{}
	}
	if cfg := currentRuntimeConfig(); cfg.Enabled && cfg.StreamingEnabled {
		if lib, ok := currentDynamicLibrary(); ok {
			if code, errType, messageText, ok := lib.CallOpenAIWSParseErrorFields(message); ok {
				recordMetric(ffiMetricEventParse, true)
				return OpenAIWSErrorFields{
					Code:    strings.TrimSpace(code),
					ErrType: strings.TrimSpace(errType),
					Message: strings.TrimSpace(messageText),
				}
			}
		}
	}
	recordMetric(ffiMetricEventParse, false)
	values := gjson.GetManyBytes(message, "error.code", "error.type", "error.message")
	return OpenAIWSErrorFields{
		Code:    strings.TrimSpace(values[0].String()),
		ErrType: strings.TrimSpace(values[1].String()),
		Message: strings.TrimSpace(values[2].String()),
	}
}
