//go:build cgo && (linux || darwin)

package ffi

/*
#cgo darwin LDFLAGS: -ldl
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

typedef size_t (*streamcore_sha256_hex_fn)(const unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_build_sse_data_frame_fn)(const unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_correct_openai_tool_calls_fn)(const unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_rewrite_openai_ws_message_for_client_fn)(const unsigned char*, size_t, const unsigned char*, size_t, const unsigned char*, size_t, int, unsigned char*, size_t);
typedef int (*streamcore_rewrite_openai_sse_line_for_client_fn)(const unsigned char*, size_t, const unsigned char*, size_t, const unsigned char*, size_t, int, unsigned char*, size_t);
typedef int (*streamcore_rewrite_openai_sse_body_for_client_fn)(const unsigned char*, size_t, const unsigned char*, size_t, const unsigned char*, size_t, int, unsigned char*, size_t);
typedef int (*streamcore_rewrite_openai_ws_message_to_sse_frame_for_client_fn)(const unsigned char*, size_t, const unsigned char*, size_t, const unsigned char*, size_t, int, unsigned char*, size_t);
typedef int (*streamcore_openai_ws_parse_usage_fn)(const unsigned char*, size_t, long long*, long long*, long long*);
typedef int (*streamcore_openai_ws_parse_envelope_fn)(const unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_openai_ws_parse_error_fields_fn)(const unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_openai_ws_parse_request_payload_summary_fn)(const unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, int*, int*, int*);
typedef int (*streamcore_openai_ws_parse_frame_summary_fn)(const unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, long long*, long long*, long long*, int*, int*, int*);
typedef int (*streamcore_parse_openai_sse_body_summary_fn)(const unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, unsigned char*, size_t, long long*, long long*, long long*, int*, int*);
typedef int (*streamcore_openai_ws_event_predicate_fn)(const unsigned char*, size_t);
typedef int (*streamcore_openai_ws_message_predicate_fn)(const unsigned char*, size_t);
typedef int (*streamcore_openai_ws_replace_model_fn)(const unsigned char*, size_t, const unsigned char*, size_t, const unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_openai_ws_drop_previous_response_id_fn)(const unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_openai_ws_set_previous_response_id_fn)(const unsigned char*, size_t, const unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_openai_ws_set_request_type_fn)(const unsigned char*, size_t, const unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_openai_ws_set_turn_metadata_fn)(const unsigned char*, size_t, const unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_openai_ws_set_input_sequence_fn)(const unsigned char*, size_t, const unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_openai_ws_normalize_payload_without_input_and_previous_response_id_fn)(const unsigned char*, size_t, unsigned char*, size_t);
typedef int (*streamcore_openai_ws_build_replay_input_sequence_fn)(const unsigned char*, size_t, int, const unsigned char*, size_t, int, unsigned char*, size_t, int*);

static void* ffi_dlopen(const char* path) {
	return dlopen(path, RTLD_NOW | RTLD_LOCAL);
}

static const char* ffi_dlerror_string() {
	const char* err = dlerror();
	return err;
}

static void* ffi_dlsym(void* handle, const char* name) {
	return dlsym(handle, name);
}

static int ffi_dlclose(void* handle) {
	return dlclose(handle);
}

static size_t ffi_call_sha256_hex(void* fn, const unsigned char* input, size_t input_len, unsigned char* out, size_t out_len) {
	return ((streamcore_sha256_hex_fn)fn)(input, input_len, out, out_len);
}

static int ffi_call_build_sse_data_frame(void* fn, const unsigned char* input, size_t input_len, unsigned char* out, size_t out_len) {
	return ((streamcore_build_sse_data_frame_fn)fn)(input, input_len, out, out_len);
}

static int ffi_call_correct_openai_tool_calls(void* fn, const unsigned char* input, size_t input_len, unsigned char* out, size_t out_len) {
	return ((streamcore_correct_openai_tool_calls_fn)fn)(input, input_len, out, out_len);
}

static int ffi_call_rewrite_openai_ws_message_for_client(void* fn, const unsigned char* input, size_t input_len, const unsigned char* from_model, size_t from_model_len, const unsigned char* to_model, size_t to_model_len, int apply_tool_correction, unsigned char* out, size_t out_len) {
	return ((streamcore_rewrite_openai_ws_message_for_client_fn)fn)(input, input_len, from_model, from_model_len, to_model, to_model_len, apply_tool_correction, out, out_len);
}

static int ffi_call_rewrite_openai_sse_line_for_client(void* fn, const unsigned char* input, size_t input_len, const unsigned char* from_model, size_t from_model_len, const unsigned char* to_model, size_t to_model_len, int apply_tool_correction, unsigned char* out, size_t out_len) {
	return ((streamcore_rewrite_openai_sse_line_for_client_fn)fn)(input, input_len, from_model, from_model_len, to_model, to_model_len, apply_tool_correction, out, out_len);
}

static int ffi_call_rewrite_openai_sse_body_for_client(void* fn, const unsigned char* input, size_t input_len, const unsigned char* from_model, size_t from_model_len, const unsigned char* to_model, size_t to_model_len, int apply_tool_correction, unsigned char* out, size_t out_len) {
	return ((streamcore_rewrite_openai_sse_body_for_client_fn)fn)(input, input_len, from_model, from_model_len, to_model, to_model_len, apply_tool_correction, out, out_len);
}

static int ffi_call_rewrite_openai_ws_message_to_sse_frame_for_client(void* fn, const unsigned char* input, size_t input_len, const unsigned char* from_model, size_t from_model_len, const unsigned char* to_model, size_t to_model_len, int apply_tool_correction, unsigned char* out, size_t out_len) {
	return ((streamcore_rewrite_openai_ws_message_to_sse_frame_for_client_fn)fn)(input, input_len, from_model, from_model_len, to_model, to_model_len, apply_tool_correction, out, out_len);
}

static int ffi_call_openai_ws_parse_usage(void* fn, const unsigned char* input, size_t input_len, long long* input_tokens, long long* output_tokens, long long* cached_tokens) {
	return ((streamcore_openai_ws_parse_usage_fn)fn)(input, input_len, input_tokens, output_tokens, cached_tokens);
}

static int ffi_call_openai_ws_parse_envelope(void* fn, const unsigned char* input, size_t input_len, unsigned char* event_type_out, size_t event_type_out_len, unsigned char* response_id_out, size_t response_id_out_len, unsigned char* response_raw_out, size_t response_raw_out_len) {
	return ((streamcore_openai_ws_parse_envelope_fn)fn)(input, input_len, event_type_out, event_type_out_len, response_id_out, response_id_out_len, response_raw_out, response_raw_out_len);
}

static int ffi_call_openai_ws_parse_error_fields(void* fn, const unsigned char* input, size_t input_len, unsigned char* code_out, size_t code_out_len, unsigned char* err_type_out, size_t err_type_out_len, unsigned char* message_out, size_t message_out_len) {
	return ((streamcore_openai_ws_parse_error_fields_fn)fn)(input, input_len, code_out, code_out_len, err_type_out, err_type_out_len, message_out, message_out_len);
}

static int ffi_call_openai_ws_parse_request_payload_summary(void* fn, const unsigned char* input, size_t input_len, unsigned char* event_type_out, size_t event_type_out_len, unsigned char* model_out, size_t model_out_len, unsigned char* prompt_cache_key_out, size_t prompt_cache_key_out_len, unsigned char* previous_response_id_out, size_t previous_response_id_out_len, int* stream_exists_out, int* stream_out, int* has_function_call_output_out) {
	return ((streamcore_openai_ws_parse_request_payload_summary_fn)fn)(input, input_len, event_type_out, event_type_out_len, model_out, model_out_len, prompt_cache_key_out, prompt_cache_key_out_len, previous_response_id_out, previous_response_id_out_len, stream_exists_out, stream_out, has_function_call_output_out);
}

static int ffi_call_openai_ws_parse_frame_summary(void* fn, const unsigned char* input, size_t input_len, unsigned char* event_type_out, size_t event_type_out_len, unsigned char* response_id_out, size_t response_id_out_len, unsigned char* response_raw_out, size_t response_raw_out_len, unsigned char* code_out, size_t code_out_len, unsigned char* err_type_out, size_t err_type_out_len, unsigned char* message_out, size_t message_out_len, long long* input_tokens_out, long long* output_tokens_out, long long* cached_tokens_out, int* is_terminal_out, int* is_token_out, int* has_tool_calls_out) {
	return ((streamcore_openai_ws_parse_frame_summary_fn)fn)(input, input_len, event_type_out, event_type_out_len, response_id_out, response_id_out_len, response_raw_out, response_raw_out_len, code_out, code_out_len, err_type_out, err_type_out_len, message_out, message_out_len, input_tokens_out, output_tokens_out, cached_tokens_out, is_terminal_out, is_token_out, has_tool_calls_out);
}

static int ffi_call_parse_openai_sse_body_summary(void* fn, const unsigned char* input, size_t input_len, unsigned char* terminal_event_type_out, size_t terminal_event_type_out_len, unsigned char* terminal_payload_out, size_t terminal_payload_out_len, unsigned char* final_response_raw_out, size_t final_response_raw_out_len, long long* input_tokens_out, long long* output_tokens_out, long long* cached_tokens_out, int* has_terminal_event_out, int* has_final_response_out) {
	return ((streamcore_parse_openai_sse_body_summary_fn)fn)(input, input_len, terminal_event_type_out, terminal_event_type_out_len, terminal_payload_out, terminal_payload_out_len, final_response_raw_out, final_response_raw_out_len, input_tokens_out, output_tokens_out, cached_tokens_out, has_terminal_event_out, has_final_response_out);
}

static int ffi_call_openai_ws_event_predicate(void* fn, const unsigned char* input, size_t input_len) {
	return ((streamcore_openai_ws_event_predicate_fn)fn)(input, input_len);
}

static int ffi_call_openai_ws_message_predicate(void* fn, const unsigned char* input, size_t input_len) {
	return ((streamcore_openai_ws_message_predicate_fn)fn)(input, input_len);
}

static int ffi_call_openai_ws_replace_model(void* fn, const unsigned char* input, size_t input_len, const unsigned char* from_model, size_t from_model_len, const unsigned char* to_model, size_t to_model_len, unsigned char* output, size_t output_len) {
	return ((streamcore_openai_ws_replace_model_fn)fn)(input, input_len, from_model, from_model_len, to_model, to_model_len, output, output_len);
}

static int ffi_call_openai_ws_drop_previous_response_id(void* fn, const unsigned char* input, size_t input_len, unsigned char* output, size_t output_len) {
	return ((streamcore_openai_ws_drop_previous_response_id_fn)fn)(input, input_len, output, output_len);
}

static int ffi_call_openai_ws_set_previous_response_id(void* fn, const unsigned char* input, size_t input_len, const unsigned char* prev, size_t prev_len, unsigned char* output, size_t output_len) {
	return ((streamcore_openai_ws_set_previous_response_id_fn)fn)(input, input_len, prev, prev_len, output, output_len);
}

static int ffi_call_openai_ws_set_request_type(void* fn, const unsigned char* input, size_t input_len, const unsigned char* event_type, size_t event_type_len, unsigned char* output, size_t output_len) {
	return ((streamcore_openai_ws_set_request_type_fn)fn)(input, input_len, event_type, event_type_len, output, output_len);
}

static int ffi_call_openai_ws_set_turn_metadata(void* fn, const unsigned char* input, size_t input_len, const unsigned char* metadata, size_t metadata_len, unsigned char* output, size_t output_len) {
	return ((streamcore_openai_ws_set_turn_metadata_fn)fn)(input, input_len, metadata, metadata_len, output, output_len);
}

static int ffi_call_openai_ws_set_input_sequence(void* fn, const unsigned char* input, size_t input_len, const unsigned char* input_sequence, size_t input_sequence_len, unsigned char* output, size_t output_len) {
	return ((streamcore_openai_ws_set_input_sequence_fn)fn)(input, input_len, input_sequence, input_sequence_len, output, output_len);
}

static int ffi_call_openai_ws_normalize_payload_without_input_and_previous_response_id(void* fn, const unsigned char* input, size_t input_len, unsigned char* output, size_t output_len) {
	return ((streamcore_openai_ws_normalize_payload_without_input_and_previous_response_id_fn)fn)(input, input_len, output, output_len);
}

static int ffi_call_openai_ws_build_replay_input_sequence(void* fn, const unsigned char* previous_full_input, size_t previous_full_input_len, int previous_full_input_exists, const unsigned char* current_payload, size_t current_payload_len, int has_previous_response_id, unsigned char* output, size_t output_len, int* exists_out) {
	return ((streamcore_openai_ws_build_replay_input_sequence_fn)fn)(previous_full_input, previous_full_input_len, previous_full_input_exists, current_payload, current_payload_len, has_previous_response_id, output, output_len, exists_out);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type dynamicLibrary struct {
	path                      string
	handle                    unsafe.Pointer
	sha256Hex                 unsafe.Pointer
	buildSSEDataFrame         unsafe.Pointer
	correctOpenAIToolCalls    unsafe.Pointer
	rewriteOpenAIWSMessage    unsafe.Pointer
	rewriteOpenAISSELine      unsafe.Pointer
	rewriteOpenAISSEBody      unsafe.Pointer
	rewriteOpenAIWSMessageSSE unsafe.Pointer
	openAIWSParseUsage        unsafe.Pointer
	openAIWSParseEnvelope     unsafe.Pointer
	openAIWSParseErrorFields  unsafe.Pointer
	openAIWSParseReqSummary   unsafe.Pointer
	openAIWSParseFrameSummary unsafe.Pointer
	openAISSEBodySummary      unsafe.Pointer
	openAIWSIsTerminalEvent   unsafe.Pointer
	openAIWSIsTokenEvent      unsafe.Pointer
	openAIWSHasToolCalls      unsafe.Pointer
	openAIWSReplaceModel      unsafe.Pointer
	openAIWSDropPrevID        unsafe.Pointer
	openAIWSSetPrevID         unsafe.Pointer
	openAIWSSetReqType        unsafe.Pointer
	openAIWSSetTurnMetadata   unsafe.Pointer
	openAIWSSetInputSequence  unsafe.Pointer
	openAIWSNormalizePayload  unsafe.Pointer
	openAIWSBuildReplayInput  unsafe.Pointer
}

func loadDynamicLibrary(path string) (*dynamicLibrary, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	handle := C.ffi_dlopen(cPath)
	if handle == nil {
		return nil, fmt.Errorf("dlopen failed: %s", dlErrorString())
	}

	cSymbol := C.CString("streamcore_sha256_hex")
	defer C.free(unsafe.Pointer(cSymbol))
	sha256Hex := C.ffi_dlsym(handle, cSymbol)
	if sha256Hex == nil {
		_ = C.ffi_dlclose(handle)
		return nil, fmt.Errorf("dlsym streamcore_sha256_hex failed: %s", dlErrorString())
	}

	cSSEFrameSymbol := C.CString("streamcore_build_sse_data_frame")
	defer C.free(unsafe.Pointer(cSSEFrameSymbol))
	buildSSEDataFrame := C.ffi_dlsym(handle, cSSEFrameSymbol)

	cToolCorrectSymbol := C.CString("streamcore_correct_openai_tool_calls")
	defer C.free(unsafe.Pointer(cToolCorrectSymbol))
	correctOpenAIToolCalls := C.ffi_dlsym(handle, cToolCorrectSymbol)

	cRewriteMessageSymbol := C.CString("streamcore_rewrite_openai_ws_message_for_client")
	defer C.free(unsafe.Pointer(cRewriteMessageSymbol))
	rewriteOpenAIWSMessage := C.ffi_dlsym(handle, cRewriteMessageSymbol)

	cRewriteSSELineSymbol := C.CString("streamcore_rewrite_openai_sse_line_for_client")
	defer C.free(unsafe.Pointer(cRewriteSSELineSymbol))
	rewriteOpenAISSELine := C.ffi_dlsym(handle, cRewriteSSELineSymbol)

	cRewriteSSEBodySymbol := C.CString("streamcore_rewrite_openai_sse_body_for_client")
	defer C.free(unsafe.Pointer(cRewriteSSEBodySymbol))
	rewriteOpenAISSEBody := C.ffi_dlsym(handle, cRewriteSSEBodySymbol)

	cRewriteMessageSSESymbol := C.CString("streamcore_rewrite_openai_ws_message_to_sse_frame_for_client")
	defer C.free(unsafe.Pointer(cRewriteMessageSSESymbol))
	rewriteOpenAIWSMessageSSE := C.ffi_dlsym(handle, cRewriteMessageSSESymbol)

	cUsageSymbol := C.CString("streamcore_openai_ws_parse_usage")
	defer C.free(unsafe.Pointer(cUsageSymbol))
	openAIWSParseUsage := C.ffi_dlsym(handle, cUsageSymbol)

	cEnvelopeSymbol := C.CString("streamcore_openai_ws_parse_envelope")
	defer C.free(unsafe.Pointer(cEnvelopeSymbol))
	openAIWSParseEnvelope := C.ffi_dlsym(handle, cEnvelopeSymbol)

	cErrorSymbol := C.CString("streamcore_openai_ws_parse_error_fields")
	defer C.free(unsafe.Pointer(cErrorSymbol))
	openAIWSParseErrorFields := C.ffi_dlsym(handle, cErrorSymbol)

	cReqSummarySymbol := C.CString("streamcore_openai_ws_parse_request_payload_summary")
	defer C.free(unsafe.Pointer(cReqSummarySymbol))
	openAIWSParseReqSummary := C.ffi_dlsym(handle, cReqSummarySymbol)

	cFrameSummarySymbol := C.CString("streamcore_openai_ws_parse_frame_summary")
	defer C.free(unsafe.Pointer(cFrameSummarySymbol))
	openAIWSParseFrameSummary := C.ffi_dlsym(handle, cFrameSummarySymbol)

	cSSEBodySummarySymbol := C.CString("streamcore_parse_openai_sse_body_summary")
	defer C.free(unsafe.Pointer(cSSEBodySummarySymbol))
	openAISSEBodySummary := C.ffi_dlsym(handle, cSSEBodySummarySymbol)

	cTerminalSymbol := C.CString("streamcore_openai_ws_is_terminal_event")
	defer C.free(unsafe.Pointer(cTerminalSymbol))
	openAIWSIsTerminalEvent := C.ffi_dlsym(handle, cTerminalSymbol)

	cTokenSymbol := C.CString("streamcore_openai_ws_is_token_event")
	defer C.free(unsafe.Pointer(cTokenSymbol))
	openAIWSIsTokenEvent := C.ffi_dlsym(handle, cTokenSymbol)

	cToolCallsSymbol := C.CString("streamcore_openai_ws_message_likely_contains_tool_calls")
	defer C.free(unsafe.Pointer(cToolCallsSymbol))
	openAIWSHasToolCalls := C.ffi_dlsym(handle, cToolCallsSymbol)

	cReplaceModelSymbol := C.CString("streamcore_openai_ws_replace_model")
	defer C.free(unsafe.Pointer(cReplaceModelSymbol))
	openAIWSReplaceModel := C.ffi_dlsym(handle, cReplaceModelSymbol)

	cDropPrevSymbol := C.CString("streamcore_openai_ws_drop_previous_response_id")
	defer C.free(unsafe.Pointer(cDropPrevSymbol))
	openAIWSDropPrevID := C.ffi_dlsym(handle, cDropPrevSymbol)

	cSetPrevSymbol := C.CString("streamcore_openai_ws_set_previous_response_id")
	defer C.free(unsafe.Pointer(cSetPrevSymbol))
	openAIWSSetPrevID := C.ffi_dlsym(handle, cSetPrevSymbol)

	cSetReqTypeSymbol := C.CString("streamcore_openai_ws_set_request_type")
	defer C.free(unsafe.Pointer(cSetReqTypeSymbol))
	openAIWSSetReqType := C.ffi_dlsym(handle, cSetReqTypeSymbol)

	cSetTurnMetadataSymbol := C.CString("streamcore_openai_ws_set_turn_metadata")
	defer C.free(unsafe.Pointer(cSetTurnMetadataSymbol))
	openAIWSSetTurnMetadata := C.ffi_dlsym(handle, cSetTurnMetadataSymbol)

	cSetInputSequenceSymbol := C.CString("streamcore_openai_ws_set_input_sequence")
	defer C.free(unsafe.Pointer(cSetInputSequenceSymbol))
	openAIWSSetInputSequence := C.ffi_dlsym(handle, cSetInputSequenceSymbol)

	cNormalizePayloadSymbol := C.CString("streamcore_openai_ws_normalize_payload_without_input_and_previous_response_id")
	defer C.free(unsafe.Pointer(cNormalizePayloadSymbol))
	openAIWSNormalizePayload := C.ffi_dlsym(handle, cNormalizePayloadSymbol)

	cBuildReplayInputSymbol := C.CString("streamcore_openai_ws_build_replay_input_sequence")
	defer C.free(unsafe.Pointer(cBuildReplayInputSymbol))
	openAIWSBuildReplayInput := C.ffi_dlsym(handle, cBuildReplayInputSymbol)

	return &dynamicLibrary{
		path:                      path,
		handle:                    handle,
		sha256Hex:                 sha256Hex,
		buildSSEDataFrame:         buildSSEDataFrame,
		correctOpenAIToolCalls:    correctOpenAIToolCalls,
		rewriteOpenAIWSMessage:    rewriteOpenAIWSMessage,
		rewriteOpenAISSELine:      rewriteOpenAISSELine,
		rewriteOpenAISSEBody:      rewriteOpenAISSEBody,
		rewriteOpenAIWSMessageSSE: rewriteOpenAIWSMessageSSE,
		openAIWSParseUsage:        openAIWSParseUsage,
		openAIWSParseEnvelope:     openAIWSParseEnvelope,
		openAIWSParseErrorFields:  openAIWSParseErrorFields,
		openAIWSParseReqSummary:   openAIWSParseReqSummary,
		openAIWSParseFrameSummary: openAIWSParseFrameSummary,
		openAISSEBodySummary:      openAISSEBodySummary,
		openAIWSIsTerminalEvent:   openAIWSIsTerminalEvent,
		openAIWSIsTokenEvent:      openAIWSIsTokenEvent,
		openAIWSHasToolCalls:      openAIWSHasToolCalls,
		openAIWSReplaceModel:      openAIWSReplaceModel,
		openAIWSDropPrevID:        openAIWSDropPrevID,
		openAIWSSetPrevID:         openAIWSSetPrevID,
		openAIWSSetReqType:        openAIWSSetReqType,
		openAIWSSetTurnMetadata:   openAIWSSetTurnMetadata,
		openAIWSSetInputSequence:  openAIWSSetInputSequence,
		openAIWSNormalizePayload:  openAIWSNormalizePayload,
		openAIWSBuildReplayInput:  openAIWSBuildReplayInput,
	}, nil
}

func (d *dynamicLibrary) Close() error {
	if d == nil || d.handle == nil {
		return nil
	}
	if rc := C.ffi_dlclose(d.handle); rc != 0 {
		return fmt.Errorf("dlclose failed: %s", dlErrorString())
	}
	d.handle = nil
	d.sha256Hex = nil
	d.buildSSEDataFrame = nil
	d.correctOpenAIToolCalls = nil
	d.rewriteOpenAIWSMessage = nil
	d.rewriteOpenAISSELine = nil
	d.rewriteOpenAISSEBody = nil
	d.rewriteOpenAIWSMessageSSE = nil
	d.openAIWSParseUsage = nil
	d.openAIWSParseEnvelope = nil
	d.openAIWSParseErrorFields = nil
	d.openAIWSParseReqSummary = nil
	d.openAIWSParseFrameSummary = nil
	d.openAISSEBodySummary = nil
	d.openAIWSIsTerminalEvent = nil
	d.openAIWSIsTokenEvent = nil
	d.openAIWSHasToolCalls = nil
	d.openAIWSReplaceModel = nil
	d.openAIWSDropPrevID = nil
	d.openAIWSSetPrevID = nil
	d.openAIWSSetReqType = nil
	d.openAIWSSetTurnMetadata = nil
	d.openAIWSSetInputSequence = nil
	d.openAIWSNormalizePayload = nil
	d.openAIWSBuildReplayInput = nil
	return nil
}

func (d *dynamicLibrary) CallSHA256Hex(input []byte) (string, bool) {
	if d == nil || d.sha256Hex == nil {
		return "", false
	}
	inLen := len(input)
	if inLen == 0 {
		input = []byte{}
	}
	out := make([]byte, 65)
	var inPtr *C.uchar
	if len(input) > 0 {
		inPtr = (*C.uchar)(unsafe.Pointer(&input[0]))
	}
	written := C.ffi_call_sha256_hex(
		d.sha256Hex,
		inPtr,
		C.size_t(inLen),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if written == 0 {
		return "", false
	}
	return string(out[:int(written)]), true
}

func (d *dynamicLibrary) CallBuildSSEDataFrame(input []byte) ([]byte, bool) {
	if d == nil || d.buildSSEDataFrame == nil || len(input) == 0 {
		return nil, false
	}
	out := make([]byte, len(input)+16)
	ok := C.ffi_call_build_sse_data_frame(
		d.buildSSEDataFrame,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallCorrectOpenAIToolCalls(input []byte) ([]byte, bool) {
	if d == nil || d.correctOpenAIToolCalls == nil || len(input) == 0 {
		return nil, false
	}
	out := make([]byte, mutationOutputBufferLen(len(input), 256))
	ok := C.ffi_call_correct_openai_tool_calls(
		d.correctOpenAIToolCalls,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallRewriteOpenAIWSMessageForClient(input []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	if d == nil || d.rewriteOpenAIWSMessage == nil || len(input) == 0 {
		return nil, false
	}
	var fromPtr *C.uchar
	var toPtr *C.uchar
	fromBytes := []byte(fromModel)
	toBytes := []byte(toModel)
	if len(fromBytes) > 0 {
		fromPtr = (*C.uchar)(unsafe.Pointer(&fromBytes[0]))
	}
	if len(toBytes) > 0 {
		toPtr = (*C.uchar)(unsafe.Pointer(&toBytes[0]))
	}
	out := make([]byte, mutationOutputBufferLen(len(input), len(fromBytes)+len(toBytes)+256))
	ok := C.ffi_call_rewrite_openai_ws_message_for_client(
		d.rewriteOpenAIWSMessage,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		fromPtr,
		C.size_t(len(fromBytes)),
		toPtr,
		C.size_t(len(toBytes)),
		boolToCInt(applyToolCorrection),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallRewriteOpenAISSELineForClient(input []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	if d == nil || d.rewriteOpenAISSELine == nil || len(input) == 0 {
		return nil, false
	}
	var fromPtr *C.uchar
	var toPtr *C.uchar
	fromBytes := []byte(fromModel)
	toBytes := []byte(toModel)
	if len(fromBytes) > 0 {
		fromPtr = (*C.uchar)(unsafe.Pointer(&fromBytes[0]))
	}
	if len(toBytes) > 0 {
		toPtr = (*C.uchar)(unsafe.Pointer(&toBytes[0]))
	}
	out := make([]byte, mutationOutputBufferLen(len(input), len(fromBytes)+len(toBytes)+256))
	ok := C.ffi_call_rewrite_openai_sse_line_for_client(
		d.rewriteOpenAISSELine,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		fromPtr,
		C.size_t(len(fromBytes)),
		toPtr,
		C.size_t(len(toBytes)),
		boolToCInt(applyToolCorrection),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallRewriteOpenAISSEBodyForClient(input []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	if d == nil || d.rewriteOpenAISSEBody == nil || len(input) == 0 {
		return nil, false
	}
	var fromPtr *C.uchar
	var toPtr *C.uchar
	fromBytes := []byte(fromModel)
	toBytes := []byte(toModel)
	if len(fromBytes) > 0 {
		fromPtr = (*C.uchar)(unsafe.Pointer(&fromBytes[0]))
	}
	if len(toBytes) > 0 {
		toPtr = (*C.uchar)(unsafe.Pointer(&toBytes[0]))
	}
	out := make([]byte, mutationOutputBufferLen(len(input), len(fromBytes)+len(toBytes)+256))
	ok := C.ffi_call_rewrite_openai_sse_body_for_client(
		d.rewriteOpenAISSEBody,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		fromPtr,
		C.size_t(len(fromBytes)),
		toPtr,
		C.size_t(len(toBytes)),
		boolToCInt(applyToolCorrection),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallRewriteOpenAIWSMessageToSSEFrameForClient(input []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	if d == nil || d.rewriteOpenAIWSMessageSSE == nil || len(input) == 0 {
		return nil, false
	}
	var fromPtr *C.uchar
	var toPtr *C.uchar
	fromBytes := []byte(fromModel)
	toBytes := []byte(toModel)
	if len(fromBytes) > 0 {
		fromPtr = (*C.uchar)(unsafe.Pointer(&fromBytes[0]))
	}
	if len(toBytes) > 0 {
		toPtr = (*C.uchar)(unsafe.Pointer(&toBytes[0]))
	}
	out := make([]byte, mutationOutputBufferLen(len(input), len(fromBytes)+len(toBytes)+256))
	ok := C.ffi_call_rewrite_openai_ws_message_to_sse_frame_for_client(
		d.rewriteOpenAIWSMessageSSE,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		fromPtr,
		C.size_t(len(fromBytes)),
		toPtr,
		C.size_t(len(toBytes)),
		boolToCInt(applyToolCorrection),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallOpenAIWSParseUsage(input []byte) (int, int, int, bool) {
	if d == nil || d.openAIWSParseUsage == nil {
		return 0, 0, 0, false
	}
	inLen := len(input)
	if inLen == 0 {
		return 0, 0, 0, false
	}
	var inputTokens C.longlong
	var outputTokens C.longlong
	var cachedTokens C.longlong
	ok := C.ffi_call_openai_ws_parse_usage(
		d.openAIWSParseUsage,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(inLen),
		&inputTokens,
		&outputTokens,
		&cachedTokens,
	)
	if ok == 0 {
		return 0, 0, 0, false
	}
	return int(inputTokens), int(outputTokens), int(cachedTokens), true
}

func (d *dynamicLibrary) CallOpenAIWSParseEnvelope(input []byte) (string, string, string, bool) {
	if d == nil || d.openAIWSParseEnvelope == nil || len(input) == 0 {
		return "", "", "", false
	}
	eventType := make([]byte, 256)
	responseID := make([]byte, 256)
	responseRaw := make([]byte, 65536)
	ok := C.ffi_call_openai_ws_parse_envelope(
		d.openAIWSParseEnvelope,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&eventType[0])),
		C.size_t(len(eventType)),
		(*C.uchar)(unsafe.Pointer(&responseID[0])),
		C.size_t(len(responseID)),
		(*C.uchar)(unsafe.Pointer(&responseRaw[0])),
		C.size_t(len(responseRaw)),
	)
	if ok == 0 {
		return "", "", "", false
	}
	return cStringBufferToString(eventType), cStringBufferToString(responseID), cStringBufferToString(responseRaw), true
}

func (d *dynamicLibrary) CallOpenAIWSParseErrorFields(input []byte) (string, string, string, bool) {
	if d == nil || d.openAIWSParseErrorFields == nil || len(input) == 0 {
		return "", "", "", false
	}
	code := make([]byte, 256)
	errType := make([]byte, 256)
	message := make([]byte, 2048)
	ok := C.ffi_call_openai_ws_parse_error_fields(
		d.openAIWSParseErrorFields,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&code[0])),
		C.size_t(len(code)),
		(*C.uchar)(unsafe.Pointer(&errType[0])),
		C.size_t(len(errType)),
		(*C.uchar)(unsafe.Pointer(&message[0])),
		C.size_t(len(message)),
	)
	if ok == 0 {
		return "", "", "", false
	}
	return cStringBufferToString(code), cStringBufferToString(errType), cStringBufferToString(message), true
}

func (d *dynamicLibrary) CallOpenAIWSParseRequestPayloadSummary(input []byte) (string, string, string, string, bool, bool, bool, bool) {
	if d == nil || d.openAIWSParseReqSummary == nil || len(input) == 0 {
		return "", "", "", "", false, false, false, false
	}
	eventType := make([]byte, 256)
	model := make([]byte, 512)
	promptCacheKey := make([]byte, 512)
	previousResponseID := make([]byte, 512)
	var streamExists C.int
	var stream C.int
	var hasFunctionCallOutput C.int
	ok := C.ffi_call_openai_ws_parse_request_payload_summary(
		d.openAIWSParseReqSummary,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&eventType[0])),
		C.size_t(len(eventType)),
		(*C.uchar)(unsafe.Pointer(&model[0])),
		C.size_t(len(model)),
		(*C.uchar)(unsafe.Pointer(&promptCacheKey[0])),
		C.size_t(len(promptCacheKey)),
		(*C.uchar)(unsafe.Pointer(&previousResponseID[0])),
		C.size_t(len(previousResponseID)),
		&streamExists,
		&stream,
		&hasFunctionCallOutput,
	)
	if ok == 0 {
		return "", "", "", "", false, false, false, false
	}
	return cStringBufferToString(eventType),
		cStringBufferToString(model),
		cStringBufferToString(promptCacheKey),
		cStringBufferToString(previousResponseID),
		streamExists != 0,
		stream != 0,
		hasFunctionCallOutput != 0,
		true
}

func (d *dynamicLibrary) CallOpenAIWSParseFrameSummary(input []byte) (string, string, string, string, string, string, int, int, int, bool, bool, bool, bool) {
	if d == nil || d.openAIWSParseFrameSummary == nil || len(input) == 0 {
		return "", "", "", "", "", "", 0, 0, 0, false, false, false, false
	}
	eventType := make([]byte, 256)
	responseID := make([]byte, 256)
	responseRaw := make([]byte, 65536)
	code := make([]byte, 256)
	errType := make([]byte, 256)
	message := make([]byte, 2048)
	var inputTokens C.longlong
	var outputTokens C.longlong
	var cachedTokens C.longlong
	var isTerminal C.int
	var isToken C.int
	var hasToolCalls C.int
	ok := C.ffi_call_openai_ws_parse_frame_summary(
		d.openAIWSParseFrameSummary,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&eventType[0])),
		C.size_t(len(eventType)),
		(*C.uchar)(unsafe.Pointer(&responseID[0])),
		C.size_t(len(responseID)),
		(*C.uchar)(unsafe.Pointer(&responseRaw[0])),
		C.size_t(len(responseRaw)),
		(*C.uchar)(unsafe.Pointer(&code[0])),
		C.size_t(len(code)),
		(*C.uchar)(unsafe.Pointer(&errType[0])),
		C.size_t(len(errType)),
		(*C.uchar)(unsafe.Pointer(&message[0])),
		C.size_t(len(message)),
		&inputTokens,
		&outputTokens,
		&cachedTokens,
		&isTerminal,
		&isToken,
		&hasToolCalls,
	)
	if ok == 0 {
		return "", "", "", "", "", "", 0, 0, 0, false, false, false, false
	}
	return cStringBufferToString(eventType),
		cStringBufferToString(responseID),
		cStringBufferToString(responseRaw),
		cStringBufferToString(code),
		cStringBufferToString(errType),
		cStringBufferToString(message),
		int(inputTokens),
		int(outputTokens),
		int(cachedTokens),
		isTerminal != 0,
		isToken != 0,
		hasToolCalls != 0,
		true
}

func (d *dynamicLibrary) CallParseOpenAISSEBodySummary(input []byte) (string, string, string, int, int, int, bool, bool, bool) {
	if d == nil || d.openAISSEBodySummary == nil || len(input) == 0 {
		return "", "", "", 0, 0, 0, false, false, false
	}
	terminalEventType := make([]byte, 256)
	terminalPayload := make([]byte, 65536)
	finalResponseRaw := make([]byte, 65536)
	var inputTokens C.longlong
	var outputTokens C.longlong
	var cachedTokens C.longlong
	var hasTerminalEvent C.int
	var hasFinalResponse C.int
	ok := C.ffi_call_parse_openai_sse_body_summary(
		d.openAISSEBodySummary,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&terminalEventType[0])),
		C.size_t(len(terminalEventType)),
		(*C.uchar)(unsafe.Pointer(&terminalPayload[0])),
		C.size_t(len(terminalPayload)),
		(*C.uchar)(unsafe.Pointer(&finalResponseRaw[0])),
		C.size_t(len(finalResponseRaw)),
		&inputTokens,
		&outputTokens,
		&cachedTokens,
		&hasTerminalEvent,
		&hasFinalResponse,
	)
	if ok == 0 {
		return "", "", "", 0, 0, 0, false, false, false
	}
	return cStringBufferToString(terminalEventType),
		cStringBufferToString(terminalPayload),
		cStringBufferToString(finalResponseRaw),
		int(inputTokens),
		int(outputTokens),
		int(cachedTokens),
		hasTerminalEvent != 0,
		hasFinalResponse != 0,
		true
}

func (d *dynamicLibrary) CallOpenAIWSIsTerminalEvent(eventType string) (bool, bool) {
	if d == nil || d.openAIWSIsTerminalEvent == nil {
		return false, false
	}
	payload := []byte(eventType)
	if len(payload) == 0 {
		return false, true
	}
	ok := C.ffi_call_openai_ws_event_predicate(
		d.openAIWSIsTerminalEvent,
		(*C.uchar)(unsafe.Pointer(&payload[0])),
		C.size_t(len(payload)),
	)
	return ok != 0, true
}

func (d *dynamicLibrary) CallOpenAIWSIsTokenEvent(eventType string) (bool, bool) {
	if d == nil || d.openAIWSIsTokenEvent == nil {
		return false, false
	}
	payload := []byte(eventType)
	if len(payload) == 0 {
		return false, true
	}
	ok := C.ffi_call_openai_ws_event_predicate(
		d.openAIWSIsTokenEvent,
		(*C.uchar)(unsafe.Pointer(&payload[0])),
		C.size_t(len(payload)),
	)
	return ok != 0, true
}

func (d *dynamicLibrary) CallOpenAIWSHasToolCalls(message []byte) (bool, bool) {
	if d == nil || d.openAIWSHasToolCalls == nil {
		return false, false
	}
	if len(message) == 0 {
		return false, true
	}
	ok := C.ffi_call_openai_ws_message_predicate(
		d.openAIWSHasToolCalls,
		(*C.uchar)(unsafe.Pointer(&message[0])),
		C.size_t(len(message)),
	)
	return ok != 0, true
}

func (d *dynamicLibrary) CallOpenAIWSReplaceModel(input []byte, fromModel, toModel string) ([]byte, bool) {
	if d == nil || d.openAIWSReplaceModel == nil {
		return nil, false
	}
	from := []byte(fromModel)
	to := []byte(toModel)
	if len(input) == 0 || len(from) == 0 || len(to) == 0 {
		return nil, false
	}
	out := make([]byte, mutationOutputBufferLen(len(input), len(from)+len(to)))
	ok := C.ffi_call_openai_ws_replace_model(
		d.openAIWSReplaceModel,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&from[0])),
		C.size_t(len(from)),
		(*C.uchar)(unsafe.Pointer(&to[0])),
		C.size_t(len(to)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallOpenAIWSDropPreviousResponseID(input []byte) ([]byte, bool) {
	if d == nil || d.openAIWSDropPrevID == nil || len(input) == 0 {
		return nil, false
	}
	out := make([]byte, mutationOutputBufferLen(len(input), 0))
	ok := C.ffi_call_openai_ws_drop_previous_response_id(
		d.openAIWSDropPrevID,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallOpenAIWSSetPreviousResponseID(input []byte, previousResponseID string) ([]byte, bool) {
	if d == nil || d.openAIWSSetPrevID == nil {
		return nil, false
	}
	prev := []byte(previousResponseID)
	if len(input) == 0 || len(prev) == 0 {
		return nil, false
	}
	out := make([]byte, mutationOutputBufferLen(len(input), len(prev)))
	ok := C.ffi_call_openai_ws_set_previous_response_id(
		d.openAIWSSetPrevID,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&prev[0])),
		C.size_t(len(prev)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallOpenAIWSSetRequestType(input []byte, eventType string) ([]byte, bool) {
	if d == nil || d.openAIWSSetReqType == nil {
		return nil, false
	}
	event := []byte(eventType)
	if len(input) == 0 || len(event) == 0 {
		return nil, false
	}
	out := make([]byte, mutationOutputBufferLen(len(input), len(event)))
	ok := C.ffi_call_openai_ws_set_request_type(
		d.openAIWSSetReqType,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&event[0])),
		C.size_t(len(event)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallOpenAIWSSetTurnMetadata(input []byte, turnMetadata string) ([]byte, bool) {
	if d == nil || d.openAIWSSetTurnMetadata == nil {
		return nil, false
	}
	metadata := []byte(turnMetadata)
	if len(input) == 0 || len(metadata) == 0 {
		return nil, false
	}
	out := make([]byte, mutationOutputBufferLen(len(input), len(metadata)))
	ok := C.ffi_call_openai_ws_set_turn_metadata(
		d.openAIWSSetTurnMetadata,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&metadata[0])),
		C.size_t(len(metadata)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallOpenAIWSSetInputSequence(input []byte, inputSequenceJSON []byte) ([]byte, bool) {
	if d == nil || d.openAIWSSetInputSequence == nil {
		return nil, false
	}
	if len(input) == 0 || len(inputSequenceJSON) == 0 {
		return nil, false
	}
	out := make([]byte, mutationOutputBufferLen(len(input), len(inputSequenceJSON)))
	ok := C.ffi_call_openai_ws_set_input_sequence(
		d.openAIWSSetInputSequence,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&inputSequenceJSON[0])),
		C.size_t(len(inputSequenceJSON)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallOpenAIWSNormalizePayloadWithoutInputAndPreviousResponseID(input []byte) ([]byte, bool) {
	if d == nil || d.openAIWSNormalizePayload == nil || len(input) == 0 {
		return nil, false
	}
	out := make([]byte, mutationOutputBufferLen(len(input), 0))
	ok := C.ffi_call_openai_ws_normalize_payload_without_input_and_previous_response_id(
		d.openAIWSNormalizePayload,
		(*C.uchar)(unsafe.Pointer(&input[0])),
		C.size_t(len(input)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
	)
	if ok == 0 {
		return nil, false
	}
	return cStringBufferToBytes(out), true
}

func (d *dynamicLibrary) CallOpenAIWSBuildReplayInputSequence(previousFullInputJSON []byte, previousFullInputExists bool, currentPayload []byte, hasPreviousResponseID bool) ([]byte, bool, bool) {
	if d == nil || d.openAIWSBuildReplayInput == nil || len(currentPayload) == 0 {
		return nil, false, false
	}
	var prevPtr *C.uchar
	if len(previousFullInputJSON) > 0 {
		prevPtr = (*C.uchar)(unsafe.Pointer(&previousFullInputJSON[0]))
	}
	out := make([]byte, mutationOutputBufferLen(len(currentPayload), len(previousFullInputJSON)))
	var exists C.int
	ok := C.ffi_call_openai_ws_build_replay_input_sequence(
		d.openAIWSBuildReplayInput,
		prevPtr,
		C.size_t(len(previousFullInputJSON)),
		boolToCInt(previousFullInputExists),
		(*C.uchar)(unsafe.Pointer(&currentPayload[0])),
		C.size_t(len(currentPayload)),
		boolToCInt(hasPreviousResponseID),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		C.size_t(len(out)),
		&exists,
	)
	if ok == 0 {
		return nil, false, false
	}
	if exists == 0 {
		return nil, false, true
	}
	return cStringBufferToBytes(out), true, true
}

func boolToCInt(v bool) C.int {
	if v {
		return 1
	}
	return 0
}

func mutationOutputBufferLen(inputLen, extra int) int {
	size := inputLen*2 + extra + 256
	if size < 512 {
		return 512
	}
	return size
}

func cStringBufferToString(buf []byte) string {
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}

func cStringBufferToBytes(buf []byte) []byte {
	for i, b := range buf {
		if b == 0 {
			return append([]byte(nil), buf[:i]...)
		}
	}
	return append([]byte(nil), buf...)
}

func dlErrorString() string {
	err := C.ffi_dlerror_string()
	if err == nil {
		return "unknown"
	}
	return C.GoString(err)
}
