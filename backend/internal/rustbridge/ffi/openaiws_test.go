package ffi

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestParseOpenAIWSEventEnvelope(t *testing.T) {
	message := []byte(`{"type":"response.completed","response":{"id":"resp_123","usage":{"input_tokens":3,"output_tokens":4}}}`)
	parsed := ParseOpenAIWSEventEnvelope(message)
	if parsed.EventType != "response.completed" {
		t.Fatalf("unexpected event type: %q", parsed.EventType)
	}
	if parsed.ResponseID != "resp_123" {
		t.Fatalf("unexpected response id: %q", parsed.ResponseID)
	}
	if parsed.ResponseRaw == "" {
		t.Fatal("expected response raw")
	}
}

func TestParseOpenAIWSUsageFields(t *testing.T) {
	message := []byte(`{"response":{"usage":{"input_tokens":11,"output_tokens":22,"input_tokens_details":{"cached_tokens":7}}}}`)
	parsed := ParseOpenAIWSUsageFields(message)
	if parsed.InputTokens != 11 || parsed.OutputTokens != 22 || parsed.CachedInputTokens != 7 {
		t.Fatalf("unexpected usage fields: %+v", parsed)
	}
}

func TestParseOpenAIWSRequestPayloadSummary(t *testing.T) {
	message := []byte(`{"type":"response.create","model":"gpt-5.1","stream":false,"prompt_cache_key":" cache-key ","previous_response_id":" resp_1 ","input":[{"type":"function_call_output","call_id":"call_1"}]}`)
	parsed := ParseOpenAIWSRequestPayloadSummary(message)
	if parsed.EventType != "response.create" {
		t.Fatalf("unexpected event type: %q", parsed.EventType)
	}
	if parsed.Model != "gpt-5.1" {
		t.Fatalf("unexpected model: %q", parsed.Model)
	}
	if parsed.PromptCacheKey != "cache-key" {
		t.Fatalf("unexpected prompt cache key: %q", parsed.PromptCacheKey)
	}
	if parsed.PreviousResponseID != "resp_1" {
		t.Fatalf("unexpected previous response id: %q", parsed.PreviousResponseID)
	}
	if !parsed.StreamExists || parsed.Stream {
		t.Fatalf("unexpected stream flags: exists=%v stream=%v", parsed.StreamExists, parsed.Stream)
	}
	if !parsed.HasFunctionCallOutput {
		t.Fatal("expected function_call_output detection")
	}
}

func TestParseOpenAIWSErrorFields(t *testing.T) {
	message := []byte(`{"error":{"code":"rate_limit","type":"rate_limit_error","message":"slow down"}}`)
	parsed := ParseOpenAIWSErrorFields(message)
	if parsed.Code != "rate_limit" || parsed.ErrType != "rate_limit_error" || parsed.Message != "slow down" {
		t.Fatalf("unexpected error fields: %+v", parsed)
	}
}

func TestParseOpenAIWSFrameSummary(t *testing.T) {
	message := []byte(`{"type":"response.completed","response":{"id":"resp_123","usage":{"input_tokens":11,"output_tokens":22,"input_tokens_details":{"cached_tokens":7}},"tool_calls":[{"id":"tc1"}]},"tool_calls":[{"id":"tc1"}]}`)
	parsed := ParseOpenAIWSFrameSummary(message)
	if parsed.EventType != "response.completed" || parsed.ResponseID != "resp_123" || parsed.InputTokens != 11 || parsed.OutputTokens != 22 || parsed.CachedInputTokens != 7 {
		t.Fatalf("unexpected frame summary: %+v", parsed)
	}
	if !parsed.IsTerminalEvent || !parsed.IsTokenEvent || !parsed.HasToolCalls {
		t.Fatalf("unexpected frame summary flags: %+v", parsed)
	}
}

func TestParseOpenAISSEBodySummary(t *testing.T) {
	body := []byte("event: message\ndata: {\"type\":\"response.in_progress\",\"response\":{\"id\":\"resp_1\"}}\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":11,\"output_tokens\":22,\"input_tokens_details\":{\"cached_tokens\":3}}}}\ndata: [DONE]\n")
	parsed := ParseOpenAISSEBodySummary(body)
	if !parsed.HasTerminalEvent || !parsed.HasFinalResponse {
		t.Fatalf("unexpected sse body summary flags: %+v", parsed)
	}
	if parsed.TerminalEventType != "response.completed" || parsed.InputTokens != 11 || parsed.OutputTokens != 22 || parsed.CachedInputTokens != 3 {
		t.Fatalf("unexpected sse body summary: %+v", parsed)
	}
	if !strings.Contains(parsed.FinalResponseRaw, `"id":"resp_1"`) {
		t.Fatalf("unexpected sse body final response: %+v", parsed)
	}
}

func TestBuildSSEDataFrame(t *testing.T) {
	frame, ok := BuildSSEDataFrame([]byte(`{"type":"response.created"}`))
	if ok {
		t.Fatalf("did not expect rust ffi sse frame build without configured library: %s", string(frame))
	}
}

func TestCorrectOpenAIToolCalls(t *testing.T) {
	corrected, ok := CorrectOpenAIToolCalls([]byte(`{"tool_calls":[{"function":{"name":"apply_patch"}}]}`))
	if ok {
		t.Fatalf("did not expect rust ffi tool correction without configured library: %s", string(corrected))
	}
}

func TestRewriteOpenAIWSMessageForClient(t *testing.T) {
	rewritten, ok := RewriteOpenAIWSMessageForClient([]byte(`{"model":"gpt-5.1","tool_calls":[{"function":{"name":"apply_patch"}}]}`), "gpt-5.1", "custom-model", true)
	if ok {
		t.Fatalf("did not expect rust ffi client rewrite without configured library: %s", string(rewritten))
	}
}

func TestRewriteOpenAISSELineForClient(t *testing.T) {
	rewritten, ok := RewriteOpenAISSELineForClient([]byte(`data: {"model":"gpt-5.1","tool_calls":[{"function":{"name":"apply_patch"}}]}`), "gpt-5.1", "custom-model", true)
	if ok {
		t.Fatalf("did not expect rust ffi sse line rewrite without configured library: %s", string(rewritten))
	}
}

func TestRewriteOpenAISSEBodyForClient(t *testing.T) {
	rewritten, ok := RewriteOpenAISSEBodyForClient([]byte("event: message\ndata: {\"model\":\"gpt-5.1\",\"tool_calls\":[{\"function\":{\"name\":\"apply_patch\"}}]}\n"), "gpt-5.1", "custom-model", true)
	if ok {
		t.Fatalf("did not expect rust ffi sse body rewrite without configured library: %s", string(rewritten))
	}
}

func TestRewriteOpenAIWSMessageToSSEFrameForClient(t *testing.T) {
	rewritten, ok := RewriteOpenAIWSMessageToSSEFrameForClient([]byte(`{"model":"gpt-5.1","tool_calls":[{"function":{"name":"apply_patch"}}]}`), "gpt-5.1", "custom-model", true)
	if ok {
		t.Fatalf("did not expect rust ffi ws-to-sse-frame rewrite without configured library: %s", string(rewritten))
	}
}

func TestOpenAIWSEventPredicatesAndToolCallDetection(t *testing.T) {
	if !IsOpenAIWSTerminalEvent("response.completed") {
		t.Fatal("expected terminal event")
	}
	if IsOpenAIWSTerminalEvent("response.created") {
		t.Fatal("did not expect response.created to be terminal")
	}
	if !IsOpenAIWSTokenEvent("response.output_text.delta") {
		t.Fatal("expected token event")
	}
	if IsOpenAIWSTokenEvent("response.output_item.added") {
		t.Fatal("did not expect response.output_item.added to be token event")
	}
	if !OpenAIWSMessageLikelyContainsToolCalls([]byte(`{"item":{"tool_calls":[{"id":"tc1"}]}}`)) {
		t.Fatal("expected tool call detection")
	}
	if OpenAIWSMessageLikelyContainsToolCalls([]byte(`{"delta":"hello"}`)) {
		t.Fatal("did not expect tool call detection")
	}
}

func TestSnapshotMetrics_RecordsFallbacksWithoutLibrary(t *testing.T) {
	resetMetricsForTest()
	t.Cleanup(resetMetricsForTest)
	_ = Configure(config.RustFFIConfig{})

	_ = BuildETagFromBytes([]byte("sub2api"))
	_ = ParseOpenAIWSEventEnvelope([]byte(`{"type":"response.completed","response":{"id":"resp_1"}}`))
	_ = ParseOpenAISSEBodySummary([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\"}}\n"))
	_, _ = BuildSSEDataFrame([]byte(`{"type":"response.created"}`))
	_, _ = RewriteOpenAIWSMessageForClient([]byte(`{"model":"gpt-5.1"}`), "gpt-5.1", "custom-model", false)

	snapshot := SnapshotMetrics()
	if snapshot.Total.Calls < 5 || snapshot.Total.Fallbacks < 5 {
		t.Fatalf("expected fallback metrics to be recorded, got %+v", snapshot.Total)
	}
	if snapshot.Hash.Fallbacks < 1 || snapshot.EventParse.Fallbacks < 1 || snapshot.SSEBodySummary.Fallbacks < 1 || snapshot.Framing.Fallbacks < 1 || snapshot.PayloadMutate.Fallbacks < 1 {
		t.Fatalf("expected category fallback metrics, got %+v", snapshot)
	}
}

func TestOpenAIWSMutationHelpers(t *testing.T) {
	replaced, ok := ReplaceOpenAIWSMessageModel(
		[]byte(`{"type":"response.created","model":"gpt-5.1","response":{"model":"gpt-5.1"}}`),
		"gpt-5.1",
		"custom-model",
	)
	if ok {
		t.Fatalf("did not expect rust ffi replacement without configured library: %s", string(replaced))
	}

	dropped, ok := DropOpenAIWSPreviousResponseID(
		[]byte(`{"type":"response.create","previous_response_id":"resp_old","input":[]}`),
	)
	if ok {
		t.Fatalf("did not expect rust ffi previous_response_id drop without configured library: %s", string(dropped))
	}

	setPayload, ok := SetOpenAIWSPreviousResponseID(
		[]byte(`{"type":"response.create","input":[]}`),
		"resp_new",
	)
	if ok {
		t.Fatalf("did not expect rust ffi previous_response_id set without configured library: %s", string(setPayload))
	}

	setTypePayload, ok := SetOpenAIWSRequestType(
		[]byte(`{"model":"gpt-5.1","input":[]}`),
		"response.create",
	)
	if ok {
		t.Fatalf("did not expect rust ffi request type set without configured library: %s", string(setTypePayload))
	}

	setMetadataPayload, ok := SetOpenAIWSTurnMetadata(
		[]byte(`{"type":"response.create","model":"gpt-5.1"}`),
		"turn_meta_1",
	)
	if ok {
		t.Fatalf("did not expect rust ffi turn metadata set without configured library: %s", string(setMetadataPayload))
	}

	setInputPayload, ok := SetOpenAIWSInputSequence(
		[]byte(`{"type":"response.create","previous_response_id":"resp_1"}`),
		[]byte(`[{"type":"input_text","text":"hello"}]`),
	)
	if ok {
		t.Fatalf("did not expect rust ffi input sequence set without configured library: %s", string(setInputPayload))
	}

	normalizedPayload, ok := NormalizeOpenAIWSPayloadWithoutInputAndPreviousResponseID(
		[]byte(`{"model":"gpt-5.1","input":[1],"previous_response_id":"resp_x","metadata":{"b":2,"a":1}}`),
	)
	if ok {
		t.Fatalf("did not expect rust ffi payload normalization without configured library: %s", string(normalizedPayload))
	}

	replayInput, exists, ok := BuildOpenAIWSReplayInputSequence(
		[]byte(`[{"type":"input_text","text":"hello"}]`),
		true,
		[]byte(`{"previous_response_id":"resp_1","input":[{"type":"input_text","text":"world"}]}`),
		true,
	)
	if ok {
		t.Fatalf("did not expect rust ffi replay input build without configured library: exists=%v payload=%s", exists, string(replayInput))
	}
}

func TestBuildETagFromBytes_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:     true,
		HashEnabled: true,
		LibraryPath: libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	got := BuildETagFromBytes([]byte("sub2api"))
	want := "\"7e00c4e93784ee94268cd479b51ca0633d3fd2311d165b17b962a8d04806ab88\""
	if got != want {
		t.Fatalf("unexpected etag from rust library: got %q want %q", got, want)
	}
}

func TestParseOpenAIWSUsageFields_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	message := []byte(`{"response":{"usage":{"input_tokens":31,"output_tokens":41,"input_tokens_details":{"cached_tokens":17}}}}`)
	parsed := ParseOpenAIWSUsageFields(message)
	if parsed.InputTokens != 31 || parsed.OutputTokens != 41 || parsed.CachedInputTokens != 17 {
		t.Fatalf("unexpected usage fields from rust library: %+v", parsed)
	}
}

func TestParseOpenAIWSEventEnvelope_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	message := []byte(`{"type":"response.completed","response":{"id":"resp_ffi","usage":{"input_tokens":1}}}`)
	parsed := ParseOpenAIWSEventEnvelope(message)
	if parsed.EventType != "response.completed" || parsed.ResponseID != "resp_ffi" || parsed.ResponseRaw == "" {
		t.Fatalf("unexpected envelope from rust library: %+v", parsed)
	}
}

func TestParseOpenAIWSErrorFields_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	message := []byte(`{"error":{"code":"rate_limit","type":"rate_limit_error","message":"slow down"}}`)
	parsed := ParseOpenAIWSErrorFields(message)
	if parsed.Code != "rate_limit" || parsed.ErrType != "rate_limit_error" || parsed.Message != "slow down" {
		t.Fatalf("unexpected error fields from rust library: %+v", parsed)
	}
}

func TestParseOpenAIWSFrameSummary_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	message := []byte(`{"type":"response.completed","response":{"id":"resp_ffi","usage":{"input_tokens":3,"output_tokens":4,"input_tokens_details":{"cached_tokens":1}}},"tool_calls":[{"id":"tc1"}]}`)
	parsed := ParseOpenAIWSFrameSummary(message)
	if parsed.EventType != "response.completed" || parsed.ResponseID != "resp_ffi" || parsed.InputTokens != 3 || parsed.OutputTokens != 4 || parsed.CachedInputTokens != 1 {
		t.Fatalf("unexpected frame summary from rust library: %+v", parsed)
	}
	if !parsed.IsTerminalEvent || !parsed.IsTokenEvent || !parsed.HasToolCalls {
		t.Fatalf("unexpected frame summary flags from rust library: %+v", parsed)
	}
}

func TestParseOpenAISSEBodySummary_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	body := []byte("event: message\ndata: {\"type\":\"response.in_progress\",\"response\":{\"id\":\"resp_ffi\"}}\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_ffi\",\"usage\":{\"input_tokens\":7,\"output_tokens\":9,\"input_tokens_details\":{\"cached_tokens\":1}}}}\ndata: [DONE]\n")
	parsed := ParseOpenAISSEBodySummary(body)
	if !parsed.HasTerminalEvent || !parsed.HasFinalResponse {
		t.Fatalf("unexpected sse body summary flags from rust library: %+v", parsed)
	}
	if parsed.TerminalEventType != "response.completed" || parsed.InputTokens != 7 || parsed.OutputTokens != 9 || parsed.CachedInputTokens != 1 {
		t.Fatalf("unexpected sse body summary from rust library: %+v", parsed)
	}
}

func TestBuildSSEDataFrame_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	frame, ok := BuildSSEDataFrame([]byte(`{"type":"response.created"}`))
	if !ok {
		t.Fatal("expected rust ffi sse frame build")
	}
	if string(frame) != "data: {\"type\":\"response.created\"}\n\n" {
		t.Fatalf("unexpected sse frame from rust library: %q", string(frame))
	}
}

func TestCorrectOpenAIToolCalls_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	corrected, ok := CorrectOpenAIToolCalls([]byte(`{"tool_calls":[{"function":{"name":"apply_patch","arguments":"{\"path\":\"/tmp/a.txt\",\"old_string\":\"a\",\"new_string\":\"b\"}"}}]}`))
	if !ok {
		t.Fatal("expected rust ffi tool correction")
	}
	var payload map[string]any
	if err := json.Unmarshal(corrected, &payload); err != nil {
		t.Fatalf("parse corrected payload: %v", err)
	}
	toolCalls, okCalls := payload["tool_calls"].([]any)
	if !okCalls || len(toolCalls) == 0 {
		t.Fatalf("unexpected corrected tool payload from rust library: %s", string(corrected))
	}
	toolCall, okCall := toolCalls[0].(map[string]any)
	if !okCall {
		t.Fatalf("unexpected corrected tool call from rust library: %s", string(corrected))
	}
	functionCall, okFunc := toolCall["function"].(map[string]any)
	if !okFunc || functionCall["name"] != "edit" {
		t.Fatalf("unexpected corrected function from rust library: %s", string(corrected))
	}
	argumentsRaw, okArgs := functionCall["arguments"].(string)
	if !okArgs {
		t.Fatalf("unexpected corrected arguments from rust library: %s", string(corrected))
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(argumentsRaw), &args); err != nil {
		t.Fatalf("parse corrected arguments: %v", err)
	}
	if args["filePath"] != "/tmp/a.txt" || args["oldString"] != "a" || args["newString"] != "b" {
		t.Fatalf("unexpected corrected arguments from rust library: %v", args)
	}
}

func TestRewriteOpenAIWSMessageForClient_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	rewritten, ok := RewriteOpenAIWSMessageForClient(
		[]byte(`{"model":"gpt-5.1","response":{"model":"gpt-5.1"},"tool_calls":[{"function":{"name":"apply_patch","arguments":"{\"path\":\"/tmp/a.txt\",\"old_string\":\"a\",\"new_string\":\"b\"}"}}]}`),
		"gpt-5.1",
		"custom-model",
		true,
	)
	if !ok {
		t.Fatal("expected rust ffi client rewrite")
	}
	var payload map[string]any
	if err := json.Unmarshal(rewritten, &payload); err != nil {
		t.Fatalf("parse rewritten payload: %v", err)
	}
	if payload["model"] != "custom-model" {
		t.Fatalf("unexpected rewritten model: %v", payload["model"])
	}
	response, okResp := payload["response"].(map[string]any)
	if !okResp || response["model"] != "custom-model" {
		t.Fatalf("unexpected rewritten response model: %v", response)
	}
}

func TestRewriteOpenAISSELineForClient_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	rewritten, ok := RewriteOpenAISSELineForClient(
		[]byte(`data: {"model":"gpt-5.1","tool_calls":[{"function":{"name":"apply_patch"}}]}`),
		"gpt-5.1",
		"custom-model",
		true,
	)
	if !ok {
		t.Fatal("expected rust ffi sse line rewrite")
	}
	got := string(rewritten)
	if !strings.Contains(got, `"model":"custom-model"`) || !strings.Contains(got, `"name":"edit"`) || !strings.HasPrefix(got, "data: ") {
		t.Fatalf("unexpected rewritten sse line from rust library: %s", got)
	}
}

func TestRewriteOpenAISSEBodyForClient_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	rewritten, ok := RewriteOpenAISSEBodyForClient(
		[]byte("event: message\ndata: {\"model\":\"gpt-5.1\",\"tool_calls\":[{\"function\":{\"name\":\"apply_patch\"}}]}\n\ndata: [DONE]\n"),
		"gpt-5.1",
		"custom-model",
		true,
	)
	if !ok {
		t.Fatal("expected rust ffi sse body rewrite")
	}
	got := string(rewritten)
	if !strings.Contains(got, "event: message") || !strings.Contains(got, `"model":"custom-model"`) || !strings.Contains(got, `"name":"edit"`) {
		t.Fatalf("unexpected rewritten sse body from rust library: %s", got)
	}
}

func TestRewriteOpenAIWSMessageToSSEFrameForClient_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	rewritten, ok := RewriteOpenAIWSMessageToSSEFrameForClient(
		[]byte(`{"model":"gpt-5.1","tool_calls":[{"function":{"name":"apply_patch"}}]}`),
		"gpt-5.1",
		"custom-model",
		true,
	)
	if !ok {
		t.Fatal("expected rust ffi ws-to-sse-frame rewrite")
	}
	got := string(rewritten)
	if !strings.HasPrefix(got, "data: ") || !strings.HasSuffix(got, "\n\n") || !strings.Contains(got, `"model":"custom-model"`) || !strings.Contains(got, `"name":"edit"`) {
		t.Fatalf("unexpected rewritten ws-to-sse frame from rust library: %s", got)
	}
}

func TestParseOpenAIWSRequestPayloadSummary_UsesRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	message := []byte(`{"type":"response.create","model":"gpt-5.1","stream":true,"prompt_cache_key":"cache_ffi","previous_response_id":"resp_ffi","input":[{"type":"function_call_output","call_id":"call_ffi"}]}`)
	parsed := ParseOpenAIWSRequestPayloadSummary(message)
	if parsed.EventType != "response.create" || parsed.Model != "gpt-5.1" || parsed.PromptCacheKey != "cache_ffi" || parsed.PreviousResponseID != "resp_ffi" {
		t.Fatalf("unexpected request payload summary from rust library: %+v", parsed)
	}
	if !parsed.StreamExists || !parsed.Stream || !parsed.HasFunctionCallOutput {
		t.Fatalf("unexpected request payload flags from rust library: %+v", parsed)
	}
}

func TestOpenAIWSEventPredicatesAndToolCallDetection_UseRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	if !IsOpenAIWSTerminalEvent("response.done") {
		t.Fatal("expected terminal event from rust library")
	}
	if IsOpenAIWSTerminalEvent("response.created") {
		t.Fatal("did not expect created event to be terminal from rust library")
	}
	if !IsOpenAIWSTokenEvent("response.output.delta") {
		t.Fatal("expected token event from rust library")
	}
	if OpenAIWSMessageLikelyContainsToolCalls([]byte(`{"delta":"hello"}`)) {
		t.Fatal("did not expect tool call detection from rust library")
	}
	if !OpenAIWSMessageLikelyContainsToolCalls([]byte(`{"item":{"type":"function_call"}}`)) {
		t.Fatal("expected tool call detection from rust library")
	}
}

func TestOpenAIWSMutationHelpers_UseRustSharedLibraryWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}
	t.Cleanup(func() {
		_ = Configure(config.RustFFIConfig{})
	})

	replaced, ok := ReplaceOpenAIWSMessageModel(
		[]byte(`{"type":"response.created","model":"gpt-5.1","response":{"model":"gpt-5.1"}}`),
		"gpt-5.1",
		"custom-model",
	)
	if !ok {
		t.Fatal("expected rust ffi model replacement")
	}
	if string(replaced) != `{"type":"response.created","model":"custom-model","response":{"model":"custom-model"}}` {
		t.Fatalf("unexpected replaced payload from rust library: %s", string(replaced))
	}

	dropped, ok := DropOpenAIWSPreviousResponseID(
		[]byte(`{"type":"response.create","previous_response_id":"resp_old","input":[]}`),
	)
	if !ok {
		t.Fatal("expected rust ffi previous_response_id drop")
	}
	if string(dropped) != `{"type":"response.create","input":[]}` {
		t.Fatalf("unexpected dropped payload from rust library: %s", string(dropped))
	}

	setPayload, ok := SetOpenAIWSPreviousResponseID(
		[]byte(`{"type":"response.create","input":[]}`),
		"resp_new",
	)
	if !ok {
		t.Fatal("expected rust ffi previous_response_id set")
	}
	if string(setPayload) != `{"type":"response.create","input":[],"previous_response_id":"resp_new"}` {
		t.Fatalf("unexpected set payload from rust library: %s", string(setPayload))
	}

	setTypePayload, ok := SetOpenAIWSRequestType(
		[]byte(`{"model":"gpt-5.1","input":[]}`),
		"response.create",
	)
	if !ok {
		t.Fatal("expected rust ffi request type set")
	}
	if string(setTypePayload) != `{"model":"gpt-5.1","input":[],"type":"response.create"}` {
		t.Fatalf("unexpected request type payload from rust library: %s", string(setTypePayload))
	}

	setMetadataPayload, ok := SetOpenAIWSTurnMetadata(
		[]byte(`{"type":"response.create","model":"gpt-5.1"}`),
		"turn_meta_1",
	)
	if !ok {
		t.Fatal("expected rust ffi turn metadata set")
	}
	if string(setMetadataPayload) != `{"type":"response.create","model":"gpt-5.1","client_metadata":{"x-codex-turn-metadata":"turn_meta_1"}}` {
		t.Fatalf("unexpected turn metadata payload from rust library: %s", string(setMetadataPayload))
	}

	setInputPayload, ok := SetOpenAIWSInputSequence(
		[]byte(`{"type":"response.create","previous_response_id":"resp_1"}`),
		[]byte(`[{"type":"input_text","text":"hello"},{"type":"input_text","text":"world"}]`),
	)
	if !ok {
		t.Fatal("expected rust ffi input sequence set")
	}
	if string(setInputPayload) != `{"type":"response.create","previous_response_id":"resp_1","input":[{"type":"input_text","text":"hello"},{"type":"input_text","text":"world"}]}` {
		t.Fatalf("unexpected input sequence payload from rust library: %s", string(setInputPayload))
	}

	normalizedPayload, ok := NormalizeOpenAIWSPayloadWithoutInputAndPreviousResponseID(
		[]byte(`{"model":"gpt-5.1","input":[1],"previous_response_id":"resp_x","metadata":{"b":2,"a":1}}`),
	)
	if !ok {
		t.Fatal("expected rust ffi payload normalization")
	}
	if string(normalizedPayload) != `{"model":"gpt-5.1","metadata":{"b":2,"a":1}}` {
		t.Fatalf("unexpected normalized payload from rust library: %s", string(normalizedPayload))
	}

	replayInput, exists, ok := BuildOpenAIWSReplayInputSequence(
		[]byte(`[{"type":"input_text","text":"hello"}]`),
		true,
		[]byte(`{"previous_response_id":"resp_1","input":[{"type":"input_text","text":"world"}]}`),
		true,
	)
	if !ok {
		t.Fatal("expected rust ffi replay input build")
	}
	if !exists {
		t.Fatal("expected replay input to exist from rust library")
	}
	if string(replayInput) != `[{"type":"input_text","text":"hello"},{"type":"input_text","text":"world"}]` {
		t.Fatalf("unexpected replay input from rust library: %s", string(replayInput))
	}
}

func TestSnapshotMetrics_RecordsRustHitsWhenConfigured(t *testing.T) {
	libPath := os.Getenv("SUB2API_STREAMCORE_LIB_PATH")
	if libPath == "" {
		t.Skip("SUB2API_STREAMCORE_LIB_PATH is not set")
	}

	resetMetricsForTest()
	t.Cleanup(func() {
		resetMetricsForTest()
		_ = Configure(config.RustFFIConfig{})
	})

	err := Configure(config.RustFFIConfig{
		Enabled:          true,
		HashEnabled:      true,
		StreamingEnabled: true,
		LibraryPath:      libPath,
	})
	if err != nil {
		t.Fatalf("configure ffi: %v", err)
	}

	_ = BuildETagFromBytes([]byte("sub2api"))
	_ = ParseOpenAIWSFrameSummary([]byte(`{"type":"response.completed","response":{"id":"resp_ffi","usage":{"input_tokens":3,"output_tokens":4,"input_tokens_details":{"cached_tokens":1}}}}`))
	_ = ParseOpenAISSEBodySummary([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_ffi\",\"usage\":{\"input_tokens\":7,\"output_tokens\":9,\"input_tokens_details\":{\"cached_tokens\":1}}}}\n"))
	_, _ = BuildSSEDataFrame([]byte(`{"type":"response.created"}`))
	_, _ = RewriteOpenAIWSMessageForClient([]byte(`{"model":"gpt-5.1"}`), "gpt-5.1", "custom-model", false)

	snapshot := SnapshotMetrics()
	if snapshot.Total.RustHits < 5 {
		t.Fatalf("expected rust hit metrics to be recorded, got %+v", snapshot.Total)
	}
	if snapshot.Hash.RustHits < 1 || snapshot.EventParse.RustHits < 1 || snapshot.SSEBodySummary.RustHits < 1 || snapshot.Framing.RustHits < 1 || snapshot.PayloadMutate.RustHits < 1 {
		t.Fatalf("expected category rust hit metrics, got %+v", snapshot)
	}
}
