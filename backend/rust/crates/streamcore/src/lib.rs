use std::ffi::c_char;
use std::io::{Read, Write};

use brotli::{CompressorWriter, Decompressor};
use flate2::read::GzDecoder;
use flate2::write::GzEncoder;
use flate2::Compression;
use serde_json::Value;
use sha2::{Digest, Sha256};

const VERSION: &[u8] = b"streamcore/0.1.0\0";

pub fn sha256_hex(input: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(input);
    format!("{:x}", hasher.finalize())
}

pub fn build_sse_data_frame(input: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(input.len() + 8);
    out.extend_from_slice(b"data: ");
    out.extend_from_slice(input);
    out.extend_from_slice(b"\n\n");
    out
}

pub fn rewrite_openai_sse_line_for_client(
    line: &[u8],
    from_model: &str,
    to_model: &str,
    apply_tool_correction: bool,
) -> Option<Vec<u8>> {
    if !line.starts_with(b"data:") {
        return None;
    }
    let mut start = b"data:".len();
    while start < line.len() && (line[start] == b' ' || line[start] == b'\t') {
        start += 1;
    }
    let payload = &line[start..];
    if payload.is_empty() || payload == b"[DONE]" {
        return None;
    }
    let rewritten = rewrite_openai_ws_message_for_client(payload, from_model, to_model, apply_tool_correction)?;
    let mut out = Vec::with_capacity(rewritten.len() + 6);
    out.extend_from_slice(b"data: ");
    out.extend_from_slice(&rewritten);
    Some(out)
}

pub fn rewrite_openai_sse_body_for_client(
    input: &[u8],
    from_model: &str,
    to_model: &str,
    apply_tool_correction: bool,
) -> Option<Vec<u8>> {
    let body = std::str::from_utf8(input).ok()?;
    let mut changed = false;
    let mut lines = Vec::new();
    for line in body.split('\n') {
        if let Some(rewritten) =
            rewrite_openai_sse_line_for_client(line.as_bytes(), from_model, to_model, apply_tool_correction)
        {
            lines.push(String::from_utf8(rewritten).ok()?);
            changed = true;
        } else {
            lines.push(line.to_string());
        }
    }
    if !changed {
        return None;
    }
    Some(lines.join("\n").into_bytes())
}

pub fn rewrite_openai_ws_message_to_sse_frame_for_client(
    input: &[u8],
    from_model: &str,
    to_model: &str,
    apply_tool_correction: bool,
) -> Option<Vec<u8>> {
    let rewritten = rewrite_openai_ws_message_for_client(input, from_model, to_model, apply_tool_correction)?;
    Some(build_sse_data_frame(&rewritten))
}

fn correct_tool_name(name: &str) -> Option<&'static str> {
    match name.trim() {
        "apply_patch" | "applyPatch" => Some("edit"),
        "update_plan" | "updatePlan" => Some("todowrite"),
        "read_plan" | "readPlan" => Some("todoread"),
        "search_files" | "searchFiles" => Some("grep"),
        "list_files" | "listFiles" => Some("glob"),
        "read_file" | "readFile" => Some("read"),
        "write_file" | "writeFile" => Some("write"),
        "execute_bash" | "executeBash" | "exec_bash" | "execBash" => Some("bash"),
        "fetch" | "web_fetch" | "webFetch" => Some("webfetch"),
        _ => None,
    }
}

fn move_json_field(obj: &mut serde_json::Map<String, Value>, from: &str, to: &str) -> bool {
    if obj.contains_key(to) {
        return false;
    }
    let Some(value) = obj.remove(from) else {
        return false;
    };
    obj.insert(to.to_string(), value);
    true
}

fn delete_json_field(obj: &mut serde_json::Map<String, Value>, key: &str) -> bool {
    obj.remove(key).is_some()
}

fn correct_tool_arguments_object(obj: &mut serde_json::Map<String, Value>, tool_name: &str) -> bool {
    let mut changed = false;
    match tool_name {
        "bash" => {
            if !obj.contains_key("workdir") {
                changed |= move_json_field(obj, "work_dir", "workdir");
            } else {
                changed |= delete_json_field(obj, "work_dir");
            }
        }
        "edit" => {
            if !obj.contains_key("filePath") {
                changed |= move_json_field(obj, "file_path", "filePath");
                if !obj.contains_key("filePath") {
                    changed |= move_json_field(obj, "path", "filePath");
                }
                if !obj.contains_key("filePath") {
                    changed |= move_json_field(obj, "file", "filePath");
                }
            }
            changed |= move_json_field(obj, "old_string", "oldString");
            changed |= move_json_field(obj, "new_string", "newString");
            changed |= move_json_field(obj, "replace_all", "replaceAll");
        }
        _ => {}
    }
    changed
}

fn correct_tool_arguments_value(arguments: &mut Value, tool_name: &str) -> bool {
    match arguments {
        Value::String(raw) => {
            let Ok(mut parsed) = serde_json::from_str::<Value>(raw) else {
                return false;
            };
            let Some(obj) = parsed.as_object_mut() else {
                return false;
            };
            if !correct_tool_arguments_object(obj, tool_name) {
                return false;
            }
            let Ok(next) = serde_json::to_string(&parsed) else {
                return false;
            };
            *raw = next;
            true
        }
        Value::Object(obj) => correct_tool_arguments_object(obj, tool_name),
        _ => false,
    }
}

fn correct_function_value(function: &mut Value) -> bool {
    let Some(obj) = function.as_object_mut() else {
        return false;
    };
    let mut changed = false;
    let mut effective_name = obj
        .get("name")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    if let Some(correct_name) = correct_tool_name(&effective_name) {
        if effective_name != correct_name {
            obj.insert("name".to_string(), Value::String(correct_name.to_string()));
            effective_name = correct_name.to_string();
            changed = true;
        }
    }
    if let Some(arguments) = obj.get_mut("arguments") {
        changed |= correct_tool_arguments_value(arguments, &effective_name);
    }
    changed
}

fn correct_tool_calls_value(value: &mut Value) -> bool {
    let Some(obj) = value.as_object_mut() else {
        return false;
    };
    let mut changed = false;
    if let Some(tool_calls) = obj.get_mut("tool_calls").and_then(Value::as_array_mut) {
        for tool_call in tool_calls.iter_mut() {
            if let Some(function) = tool_call.get_mut("function") {
                changed |= correct_function_value(function);
            }
        }
    }
    if let Some(function_call) = obj.get_mut("function_call") {
        changed |= correct_function_value(function_call);
    }
    if let Some(delta) = obj.get_mut("delta") {
        changed |= correct_tool_calls_value(delta);
    }
    if let Some(message) = obj.get_mut("message") {
        changed |= correct_tool_calls_value(message);
    }
    if let Some(choices) = obj.get_mut("choices").and_then(Value::as_array_mut) {
        for choice in choices.iter_mut() {
            changed |= correct_tool_calls_value(choice);
        }
    }
    changed
}

pub fn correct_openai_tool_calls(input: &[u8]) -> Option<Vec<u8>> {
    if input.is_empty() {
        return None;
    }
    let mut root: Value = serde_json::from_slice(input).ok()?;
    if !correct_tool_calls_value(&mut root) {
        return None;
    }
    serde_json::to_vec(&root).ok()
}

pub fn rewrite_openai_ws_message_for_client(
    input: &[u8],
    from_model: &str,
    to_model: &str,
    apply_tool_correction: bool,
) -> Option<Vec<u8>> {
    if input.is_empty() {
        return None;
    }
    let mut root: Value = serde_json::from_slice(input).ok()?;
    let mut changed = false;

    let from_model = from_model.trim();
    let to_model = to_model.trim();
    if !from_model.is_empty() && !to_model.is_empty() && from_model != to_model {
        if let Some(root_obj) = root.as_object_mut() {
            if root_obj.get("model").and_then(Value::as_str) == Some(from_model) {
                root_obj.insert("model".to_string(), Value::String(to_model.to_string()));
                changed = true;
            }
            if let Some(response) = root_obj.get_mut("response").and_then(Value::as_object_mut) {
                if response.get("model").and_then(Value::as_str) == Some(from_model) {
                    response.insert("model".to_string(), Value::String(to_model.to_string()));
                    changed = true;
                }
            }
        }
    }

    if apply_tool_correction {
        changed |= correct_tool_calls_value(&mut root);
    }

    if !changed {
        return None;
    }
    serde_json::to_vec(&root).ok()
}

pub fn gzip_compress(input: &[u8]) -> std::io::Result<Vec<u8>> {
    let mut encoder = GzEncoder::new(Vec::new(), Compression::default());
    encoder.write_all(input)?;
    encoder.finish()
}

pub fn gzip_decompress(input: &[u8]) -> std::io::Result<Vec<u8>> {
    let mut decoder = GzDecoder::new(input);
    let mut out = Vec::new();
    decoder.read_to_end(&mut out)?;
    Ok(out)
}

pub fn brotli_compress(input: &[u8], quality: u32) -> std::io::Result<Vec<u8>> {
    let mut out = Vec::new();
    {
        let mut encoder = CompressorWriter::new(&mut out, 4096, quality, 22);
        encoder.write_all(input)?;
    }
    Ok(out)
}

pub fn brotli_decompress(input: &[u8]) -> std::io::Result<Vec<u8>> {
    let mut decoder = Decompressor::new(input, 4096);
    let mut out = Vec::new();
    decoder.read_to_end(&mut out)?;
    Ok(out)
}

pub fn parse_openai_ws_usage_fields(input: &[u8]) -> Option<(i64, i64, i64)> {
    let root: Value = serde_json::from_slice(input).ok()?;
    let response = root.get("response")?;
    let usage = response.get("usage")?;
    let input_tokens = usage.get("input_tokens").and_then(Value::as_i64).unwrap_or(0);
    let output_tokens = usage.get("output_tokens").and_then(Value::as_i64).unwrap_or(0);
    let cached_tokens = usage
        .get("input_tokens_details")
        .and_then(|v| v.get("cached_tokens"))
        .and_then(Value::as_i64)
        .unwrap_or(0);
    Some((input_tokens, output_tokens, cached_tokens))
}

pub fn parse_openai_ws_event_envelope(input: &[u8]) -> Option<(String, String, String)> {
    let root: Value = serde_json::from_slice(input).ok()?;
    let event_type = root
        .get("type")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();

    let response_id = root
        .get("response")
        .and_then(|v| v.get("id"))
        .and_then(Value::as_str)
        .or_else(|| root.get("id").and_then(Value::as_str))
        .unwrap_or("")
        .trim()
        .to_string();

    let response_raw = root
        .get("response")
        .map(|v| serde_json::to_string(v).unwrap_or_default())
        .unwrap_or_default();

    Some((event_type, response_id, response_raw))
}

pub fn parse_openai_ws_error_fields(input: &[u8]) -> Option<(String, String, String)> {
    let root: Value = serde_json::from_slice(input).ok()?;
    let error = root.get("error")?;
    let code = error
        .get("code")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let err_type = error
        .get("type")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let message = error
        .get("message")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    Some((code, err_type, message))
}

pub fn parse_openai_ws_frame_summary(
    input: &[u8],
) -> Option<(String, String, String, String, String, String, i64, i64, i64, bool, bool, bool)> {
    let root: Value = serde_json::from_slice(input).ok()?;
    let event_type = root
        .get("type")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let response_id = root
        .get("response")
        .and_then(|v| v.get("id"))
        .and_then(Value::as_str)
        .or_else(|| root.get("id").and_then(Value::as_str))
        .unwrap_or("")
        .trim()
        .to_string();
    let response_raw = root
        .get("response")
        .map(|v| serde_json::to_string(v).unwrap_or_default())
        .unwrap_or_default();
    let error = root.get("error");
    let code = error
        .and_then(|v| v.get("code"))
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let err_type = error
        .and_then(|v| v.get("type"))
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let message = error
        .and_then(|v| v.get("message"))
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let usage = root.get("response").and_then(|v| v.get("usage"));
    let input_tokens = usage
        .and_then(|v| v.get("input_tokens"))
        .and_then(Value::as_i64)
        .unwrap_or(0);
    let output_tokens = usage
        .and_then(|v| v.get("output_tokens"))
        .and_then(Value::as_i64)
        .unwrap_or(0);
    let cached_tokens = usage
        .and_then(|v| v.get("input_tokens_details"))
        .and_then(|v| v.get("cached_tokens"))
        .and_then(Value::as_i64)
        .unwrap_or(0);
    let is_terminal = is_openai_ws_terminal_event(&event_type);
    let is_token = is_openai_ws_token_event(&event_type);
    let has_tool_calls = openai_ws_message_likely_contains_tool_calls(input);

    Some((
        event_type,
        response_id,
        response_raw,
        code,
        err_type,
        message,
        input_tokens,
        output_tokens,
        cached_tokens,
        is_terminal,
        is_token,
        has_tool_calls,
    ))
}

pub fn parse_openai_sse_body_summary(
    input: &[u8],
) -> Option<(String, String, String, i64, i64, i64, bool, bool)> {
    let body = std::str::from_utf8(input).ok()?;
    let mut terminal_event_type = String::new();
    let mut terminal_payload = String::new();
    let mut final_response_raw = String::new();
    let mut input_tokens = 0i64;
    let mut output_tokens = 0i64;
    let mut cached_tokens = 0i64;
    let mut has_terminal_event = false;
    let mut has_final_response = false;

    for line in body.lines() {
        if !line.starts_with("data:") {
            continue;
        }
        let mut data = &line["data:".len()..];
        while let Some(first) = data.as_bytes().first() {
            if *first == b' ' || *first == b'\t' {
                data = &data[1..];
            } else {
                break;
            }
        }
        if data.is_empty() || data == "[DONE]" {
            continue;
        }
        let Some((event_type, _response_id, response_raw, _code, _err_type, _message, next_input_tokens, next_output_tokens, next_cached_tokens, is_terminal, _is_token, _has_tool_calls)) =
            parse_openai_ws_frame_summary(data.as_bytes())
        else {
            continue;
        };
        if is_terminal {
            if !has_terminal_event {
                terminal_event_type = event_type.clone();
                terminal_payload = data.to_string();
                has_terminal_event = true;
            }
            input_tokens = next_input_tokens;
            output_tokens = next_output_tokens;
            cached_tokens = next_cached_tokens;
        }
        if !has_final_response
            && (event_type == "response.done" || event_type == "response.completed")
            && !response_raw.is_empty()
        {
            final_response_raw = response_raw;
            has_final_response = true;
        }
    }

    Some((
        terminal_event_type,
        terminal_payload,
        final_response_raw,
        input_tokens,
        output_tokens,
        cached_tokens,
        has_terminal_event,
        has_final_response,
    ))
}

pub fn parse_openai_ws_request_payload_summary(
    input: &[u8],
) -> Option<(String, String, String, String, bool, bool, bool)> {
    let root: Value = serde_json::from_slice(input).ok()?;
    let event_type = root
        .get("type")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let model = root
        .get("model")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let prompt_cache_key = root
        .get("prompt_cache_key")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let previous_response_id = root
        .get("previous_response_id")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string();
    let stream_value = root.get("stream");
    let stream_exists = stream_value.is_some();
    let stream = stream_value.and_then(Value::as_bool).unwrap_or(false);
    let has_function_call_output = root
        .get("input")
        .and_then(Value::as_array)
        .map(|items| {
            items.iter().any(|item| {
                item.get("type")
                    .and_then(Value::as_str)
                    .map(|v| v.trim() == "function_call_output")
                    .unwrap_or(false)
            })
        })
        .unwrap_or(false);

    Some((
        event_type,
        model,
        prompt_cache_key,
        previous_response_id,
        stream_exists,
        stream,
        has_function_call_output,
    ))
}

pub fn is_openai_ws_terminal_event(event_type: &str) -> bool {
    matches!(
        event_type.trim(),
        "response.completed"
            | "response.done"
            | "response.failed"
            | "response.incomplete"
            | "response.cancelled"
            | "response.canceled"
    )
}

pub fn is_openai_ws_token_event(event_type: &str) -> bool {
    let event_type = event_type.trim();
    if event_type.is_empty() {
        return false;
    }
    match event_type {
        "response.created" | "response.in_progress" | "response.output_item.added" | "response.output_item.done" => false,
        "response.completed" | "response.done" => true,
        _ => {
            event_type.contains(".delta")
                || event_type.starts_with("response.output_text")
                || event_type.starts_with("response.output")
        }
    }
}

pub fn openai_ws_message_likely_contains_tool_calls(message: &[u8]) -> bool {
    message.windows(b"\"tool_calls\"".len()).any(|w| w == b"\"tool_calls\"")
        || message.windows(b"\"tool_call\"".len()).any(|w| w == b"\"tool_call\"")
        || message
            .windows(b"\"function_call\"".len())
            .any(|w| w == b"\"function_call\"")
}

fn parse_object_preserve(input: &[u8]) -> Option<serde_json::Map<String, Value>> {
    let value: Value = serde_json::from_slice(input).ok()?;
    value.as_object().cloned()
}

pub fn replace_openai_ws_message_model(input: &[u8], from_model: &str, to_model: &str) -> Option<Vec<u8>> {
    let from_model = from_model.trim();
    let to_model = to_model.trim();
    if input.is_empty() || from_model.is_empty() || to_model.is_empty() || from_model == to_model {
        return None;
    }
    let mut root = parse_object_preserve(input)?;
    let mut changed = false;

    if root.get("model").and_then(Value::as_str) == Some(from_model) {
        root.insert("model".to_string(), Value::String(to_model.to_string()));
        changed = true;
    }

    if let Some(response) = root.get_mut("response").and_then(Value::as_object_mut) {
        if response.get("model").and_then(Value::as_str) == Some(from_model) {
            response.insert("model".to_string(), Value::String(to_model.to_string()));
            changed = true;
        }
    }

    if !changed {
        return None;
    }
    serde_json::to_vec(&Value::Object(root)).ok()
}

pub fn drop_previous_response_id(input: &[u8]) -> Option<Vec<u8>> {
    if input.is_empty() {
        return None;
    }
    let mut root = parse_object_preserve(input)?;
    if root.remove("previous_response_id").is_none() {
        return None;
    }
    serde_json::to_vec(&Value::Object(root)).ok()
}

pub fn set_previous_response_id(input: &[u8], previous_response_id: &str) -> Option<Vec<u8>> {
    let previous_response_id = previous_response_id.trim();
    if input.is_empty() || previous_response_id.is_empty() {
        return None;
    }
    let mut root = parse_object_preserve(input)?;
    root.insert(
        "previous_response_id".to_string(),
        Value::String(previous_response_id.to_string()),
    );
    serde_json::to_vec(&Value::Object(root)).ok()
}

pub fn set_openai_ws_request_type(input: &[u8], event_type: &str) -> Option<Vec<u8>> {
    let event_type = event_type.trim();
    if input.is_empty() || event_type.is_empty() {
        return None;
    }
    let mut root = parse_object_preserve(input)?;
    if root.get("type").and_then(Value::as_str) == Some(event_type) {
        return None;
    }
    root.insert("type".to_string(), Value::String(event_type.to_string()));
    serde_json::to_vec(&Value::Object(root)).ok()
}

pub fn set_openai_ws_turn_metadata(input: &[u8], turn_metadata: &str) -> Option<Vec<u8>> {
    let turn_metadata = turn_metadata.trim();
    if input.is_empty() || turn_metadata.is_empty() {
        return None;
    }
    let mut root = parse_object_preserve(input)?;
    let mut changed = false;
    let mut next_metadata = serde_json::Map::new();
    if let Some(existing) = root.get("client_metadata").and_then(Value::as_object) {
        next_metadata = existing.clone();
    }
    if next_metadata.get("x-codex-turn-metadata").and_then(Value::as_str) != Some(turn_metadata) {
        next_metadata.insert(
            "x-codex-turn-metadata".to_string(),
            Value::String(turn_metadata.to_string()),
        );
        changed = true;
    }
    if !changed {
        return None;
    }
    root.insert("client_metadata".to_string(), Value::Object(next_metadata));
    serde_json::to_vec(&Value::Object(root)).ok()
}

pub fn set_openai_ws_input_sequence(input: &[u8], input_sequence_json: &[u8]) -> Option<Vec<u8>> {
    if input.is_empty() || input_sequence_json.is_empty() {
        return None;
    }
    let mut root = parse_object_preserve(input)?;
    let input_value: Value = serde_json::from_slice(input_sequence_json).ok()?;
    root.insert("input".to_string(), input_value);
    serde_json::to_vec(&Value::Object(root)).ok()
}

pub fn normalize_openai_ws_payload_without_input_and_previous_response_id(input: &[u8]) -> Option<Vec<u8>> {
    if input.is_empty() {
        return None;
    }
    let mut root = parse_object_preserve(input)?;
    root.remove("input");
    root.remove("previous_response_id");
    serde_json::to_vec(&Value::Object(root)).ok()
}

fn extract_openai_ws_input_sequence_value(payload: &[u8]) -> Option<(Value, bool)> {
    if payload.is_empty() {
        return Some((Value::Null, false));
    }
    let root: Value = serde_json::from_slice(payload).ok()?;
    let Some(input) = root.get("input") else {
        return Some((Value::Null, false));
    };
    match input {
        Value::Array(items) => Some((Value::Array(items.clone()), true)),
        other => Some((Value::Array(vec![other.clone()]), true)),
    }
}

pub fn build_openai_ws_replay_input_sequence(
    previous_full_input_json: &[u8],
    previous_full_input_exists: bool,
    current_payload: &[u8],
    has_previous_response_id: bool,
) -> Option<(Vec<u8>, bool)> {
    let (current_input, current_exists) = extract_openai_ws_input_sequence_value(current_payload)?;
    if !has_previous_response_id {
        return if current_exists {
            serde_json::to_vec(&current_input).ok().map(|v| (v, true))
        } else {
            Some((Vec::new(), false))
        };
    }
    if !previous_full_input_exists {
        return if current_exists {
            serde_json::to_vec(&current_input).ok().map(|v| (v, true))
        } else {
            Some((Vec::new(), false))
        };
    }

    let previous_input: Value = serde_json::from_slice(previous_full_input_json).ok()?;
    let previous_items = previous_input.as_array()?.clone();
    if !current_exists {
        return serde_json::to_vec(&Value::Array(previous_items)).ok().map(|v| (v, true));
    }

    let current_items = current_input.as_array()?.clone();
    if current_items.is_empty() {
        return serde_json::to_vec(&Value::Array(previous_items)).ok().map(|v| (v, true));
    }
    if current_items.len() >= previous_items.len()
        && previous_items
            .iter()
            .zip(current_items.iter())
            .all(|(prev, curr)| prev == curr)
    {
        return serde_json::to_vec(&Value::Array(current_items)).ok().map(|v| (v, true));
    }

    let mut merged = previous_items;
    merged.extend(current_items);
    serde_json::to_vec(&Value::Array(merged)).ok().map(|v| (v, true))
}

#[no_mangle]
pub extern "C" fn streamcore_version() -> *const c_char {
    VERSION.as_ptr() as *const c_char
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_sha256_hex(
    input_ptr: *const u8,
    input_len: usize,
    out_ptr: *mut u8,
    out_len: usize,
) -> usize {
    if input_ptr.is_null() || out_ptr.is_null() || out_len == 0 {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let digest = sha256_hex(input);
    let digest_bytes = digest.as_bytes();
    if digest_bytes.len() + 1 > out_len {
        return 0;
    }
    std::ptr::copy_nonoverlapping(digest_bytes.as_ptr(), out_ptr, digest_bytes.len());
    *out_ptr.add(digest_bytes.len()) = 0;
    digest_bytes.len()
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_build_sse_data_frame(
    input_ptr: *const u8,
    input_len: usize,
    out_ptr: *mut u8,
    out_len: usize,
) -> i32 {
    if input_ptr.is_null() || out_ptr.is_null() || out_len == 0 {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let frame = build_sse_data_frame(input);
    if frame.len() + 1 > out_len {
        return 0;
    }
    std::ptr::copy_nonoverlapping(frame.as_ptr(), out_ptr, frame.len());
    *out_ptr.add(frame.len()) = 0;
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_correct_openai_tool_calls(
    input_ptr: *const u8,
    input_len: usize,
    out_ptr: *mut u8,
    out_len: usize,
) -> i32 {
    if input_ptr.is_null() || out_ptr.is_null() || out_len == 0 {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let Some(updated) = correct_openai_tool_calls(input) else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), out_ptr, out_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_rewrite_openai_ws_message_for_client(
    input_ptr: *const u8,
    input_len: usize,
    from_model_ptr: *const u8,
    from_model_len: usize,
    to_model_ptr: *const u8,
    to_model_len: usize,
    apply_tool_correction: i32,
    out_ptr: *mut u8,
    out_len: usize,
) -> i32 {
    if input_ptr.is_null() || out_ptr.is_null() || out_len == 0 {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let from_model = if from_model_ptr.is_null() {
        ""
    } else {
        std::str::from_utf8(std::slice::from_raw_parts(from_model_ptr, from_model_len)).unwrap_or("")
    };
    let to_model = if to_model_ptr.is_null() {
        ""
    } else {
        std::str::from_utf8(std::slice::from_raw_parts(to_model_ptr, to_model_len)).unwrap_or("")
    };
    let Some(updated) = rewrite_openai_ws_message_for_client(input, from_model, to_model, apply_tool_correction != 0) else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), out_ptr, out_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_rewrite_openai_sse_line_for_client(
    line_ptr: *const u8,
    line_len: usize,
    from_model_ptr: *const u8,
    from_model_len: usize,
    to_model_ptr: *const u8,
    to_model_len: usize,
    apply_tool_correction: i32,
    out_ptr: *mut u8,
    out_len: usize,
) -> i32 {
    if line_ptr.is_null() || out_ptr.is_null() || out_len == 0 {
        return 0;
    }
    let line = std::slice::from_raw_parts(line_ptr, line_len);
    let from_model = if from_model_ptr.is_null() {
        ""
    } else {
        std::str::from_utf8(std::slice::from_raw_parts(from_model_ptr, from_model_len)).unwrap_or("")
    };
    let to_model = if to_model_ptr.is_null() {
        ""
    } else {
        std::str::from_utf8(std::slice::from_raw_parts(to_model_ptr, to_model_len)).unwrap_or("")
    };
    let Some(updated) = rewrite_openai_sse_line_for_client(line, from_model, to_model, apply_tool_correction != 0) else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), out_ptr, out_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_rewrite_openai_sse_body_for_client(
    input_ptr: *const u8,
    input_len: usize,
    from_model_ptr: *const u8,
    from_model_len: usize,
    to_model_ptr: *const u8,
    to_model_len: usize,
    apply_tool_correction: i32,
    out_ptr: *mut u8,
    out_len: usize,
) -> i32 {
    if input_ptr.is_null() || out_ptr.is_null() || out_len == 0 {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let from_model = if from_model_ptr.is_null() {
        ""
    } else {
        std::str::from_utf8(std::slice::from_raw_parts(from_model_ptr, from_model_len)).unwrap_or("")
    };
    let to_model = if to_model_ptr.is_null() {
        ""
    } else {
        std::str::from_utf8(std::slice::from_raw_parts(to_model_ptr, to_model_len)).unwrap_or("")
    };
    let Some(updated) = rewrite_openai_sse_body_for_client(input, from_model, to_model, apply_tool_correction != 0)
    else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), out_ptr, out_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_rewrite_openai_ws_message_to_sse_frame_for_client(
    input_ptr: *const u8,
    input_len: usize,
    from_model_ptr: *const u8,
    from_model_len: usize,
    to_model_ptr: *const u8,
    to_model_len: usize,
    apply_tool_correction: i32,
    out_ptr: *mut u8,
    out_len: usize,
) -> i32 {
    if input_ptr.is_null() || out_ptr.is_null() || out_len == 0 {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let from_model = if from_model_ptr.is_null() {
        ""
    } else {
        std::str::from_utf8(std::slice::from_raw_parts(from_model_ptr, from_model_len)).unwrap_or("")
    };
    let to_model = if to_model_ptr.is_null() {
        ""
    } else {
        std::str::from_utf8(std::slice::from_raw_parts(to_model_ptr, to_model_len)).unwrap_or("")
    };
    let Some(updated) =
        rewrite_openai_ws_message_to_sse_frame_for_client(input, from_model, to_model, apply_tool_correction != 0)
    else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), out_ptr, out_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_parse_usage(
    input_ptr: *const u8,
    input_len: usize,
    input_tokens_out: *mut i64,
    output_tokens_out: *mut i64,
    cached_tokens_out: *mut i64,
) -> i32 {
    if input_ptr.is_null()
        || input_tokens_out.is_null()
        || output_tokens_out.is_null()
        || cached_tokens_out.is_null()
    {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let Some((input_tokens, output_tokens, cached_tokens)) = parse_openai_ws_usage_fields(input) else {
        return 0;
    };
    *input_tokens_out = input_tokens;
    *output_tokens_out = output_tokens;
    *cached_tokens_out = cached_tokens;
    1
}

unsafe fn write_c_string(value: &str, out_ptr: *mut u8, out_len: usize) -> bool {
    if out_ptr.is_null() || out_len == 0 {
        return false;
    }
    let bytes = value.as_bytes();
    if bytes.len() + 1 > out_len {
        return false;
    }
    std::ptr::copy_nonoverlapping(bytes.as_ptr(), out_ptr, bytes.len());
    *out_ptr.add(bytes.len()) = 0;
    true
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_parse_envelope(
    input_ptr: *const u8,
    input_len: usize,
    event_type_out: *mut u8,
    event_type_out_len: usize,
    response_id_out: *mut u8,
    response_id_out_len: usize,
    response_raw_out: *mut u8,
    response_raw_out_len: usize,
) -> i32 {
    if input_ptr.is_null() {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let Some((event_type, response_id, response_raw)) = parse_openai_ws_event_envelope(input) else {
        return 0;
    };
    if !write_c_string(&event_type, event_type_out, event_type_out_len) {
        return 0;
    }
    if !write_c_string(&response_id, response_id_out, response_id_out_len) {
        return 0;
    }
    if !write_c_string(&response_raw, response_raw_out, response_raw_out_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_parse_error_fields(
    input_ptr: *const u8,
    input_len: usize,
    code_out: *mut u8,
    code_out_len: usize,
    err_type_out: *mut u8,
    err_type_out_len: usize,
    message_out: *mut u8,
    message_out_len: usize,
) -> i32 {
    if input_ptr.is_null() {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let Some((code, err_type, message)) = parse_openai_ws_error_fields(input) else {
        return 0;
    };
    if !write_c_string(&code, code_out, code_out_len) {
        return 0;
    }
    if !write_c_string(&err_type, err_type_out, err_type_out_len) {
        return 0;
    }
    if !write_c_string(&message, message_out, message_out_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_parse_frame_summary(
    input_ptr: *const u8,
    input_len: usize,
    event_type_out: *mut u8,
    event_type_out_len: usize,
    response_id_out: *mut u8,
    response_id_out_len: usize,
    response_raw_out: *mut u8,
    response_raw_out_len: usize,
    code_out: *mut u8,
    code_out_len: usize,
    err_type_out: *mut u8,
    err_type_out_len: usize,
    message_out: *mut u8,
    message_out_len: usize,
    input_tokens_out: *mut i64,
    output_tokens_out: *mut i64,
    cached_tokens_out: *mut i64,
    is_terminal_out: *mut i32,
    is_token_out: *mut i32,
    has_tool_calls_out: *mut i32,
) -> i32 {
    if input_ptr.is_null()
        || input_tokens_out.is_null()
        || output_tokens_out.is_null()
        || cached_tokens_out.is_null()
        || is_terminal_out.is_null()
        || is_token_out.is_null()
        || has_tool_calls_out.is_null()
    {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let Some((event_type, response_id, response_raw, code, err_type, message, input_tokens, output_tokens, cached_tokens, is_terminal, is_token, has_tool_calls)) =
        parse_openai_ws_frame_summary(input)
    else {
        return 0;
    };
    if !write_c_string(&event_type, event_type_out, event_type_out_len) {
        return 0;
    }
    if !write_c_string(&response_id, response_id_out, response_id_out_len) {
        return 0;
    }
    if !write_c_string(&response_raw, response_raw_out, response_raw_out_len) {
        return 0;
    }
    if !write_c_string(&code, code_out, code_out_len) {
        return 0;
    }
    if !write_c_string(&err_type, err_type_out, err_type_out_len) {
        return 0;
    }
    if !write_c_string(&message, message_out, message_out_len) {
        return 0;
    }
    *input_tokens_out = input_tokens;
    *output_tokens_out = output_tokens;
    *cached_tokens_out = cached_tokens;
    *is_terminal_out = if is_terminal { 1 } else { 0 };
    *is_token_out = if is_token { 1 } else { 0 };
    *has_tool_calls_out = if has_tool_calls { 1 } else { 0 };
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_parse_openai_sse_body_summary(
    input_ptr: *const u8,
    input_len: usize,
    terminal_event_type_out: *mut u8,
    terminal_event_type_out_len: usize,
    terminal_payload_out: *mut u8,
    terminal_payload_out_len: usize,
    final_response_raw_out: *mut u8,
    final_response_raw_out_len: usize,
    input_tokens_out: *mut i64,
    output_tokens_out: *mut i64,
    cached_tokens_out: *mut i64,
    has_terminal_event_out: *mut i32,
    has_final_response_out: *mut i32,
) -> i32 {
    if input_ptr.is_null()
        || input_tokens_out.is_null()
        || output_tokens_out.is_null()
        || cached_tokens_out.is_null()
        || has_terminal_event_out.is_null()
        || has_final_response_out.is_null()
    {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let Some((terminal_event_type, terminal_payload, final_response_raw, input_tokens, output_tokens, cached_tokens, has_terminal_event, has_final_response)) =
        parse_openai_sse_body_summary(input)
    else {
        return 0;
    };
    if !write_c_string(&terminal_event_type, terminal_event_type_out, terminal_event_type_out_len) {
        return 0;
    }
    if !write_c_string(&terminal_payload, terminal_payload_out, terminal_payload_out_len) {
        return 0;
    }
    if !write_c_string(&final_response_raw, final_response_raw_out, final_response_raw_out_len) {
        return 0;
    }
    *input_tokens_out = input_tokens;
    *output_tokens_out = output_tokens;
    *cached_tokens_out = cached_tokens;
    *has_terminal_event_out = if has_terminal_event { 1 } else { 0 };
    *has_final_response_out = if has_final_response { 1 } else { 0 };
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_parse_request_payload_summary(
    input_ptr: *const u8,
    input_len: usize,
    event_type_out: *mut u8,
    event_type_out_len: usize,
    model_out: *mut u8,
    model_out_len: usize,
    prompt_cache_key_out: *mut u8,
    prompt_cache_key_out_len: usize,
    previous_response_id_out: *mut u8,
    previous_response_id_out_len: usize,
    stream_exists_out: *mut i32,
    stream_out: *mut i32,
    has_function_call_output_out: *mut i32,
) -> i32 {
    if input_ptr.is_null()
        || stream_exists_out.is_null()
        || stream_out.is_null()
        || has_function_call_output_out.is_null()
    {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let Some((event_type, model, prompt_cache_key, previous_response_id, stream_exists, stream, has_function_call_output)) =
        parse_openai_ws_request_payload_summary(input)
    else {
        return 0;
    };
    if !write_c_string(&event_type, event_type_out, event_type_out_len) {
        return 0;
    }
    if !write_c_string(&model, model_out, model_out_len) {
        return 0;
    }
    if !write_c_string(&prompt_cache_key, prompt_cache_key_out, prompt_cache_key_out_len) {
        return 0;
    }
    if !write_c_string(
        &previous_response_id,
        previous_response_id_out,
        previous_response_id_out_len,
    ) {
        return 0;
    }
    *stream_exists_out = if stream_exists { 1 } else { 0 };
    *stream_out = if stream { 1 } else { 0 };
    *has_function_call_output_out = if has_function_call_output { 1 } else { 0 };
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_is_terminal_event(
    event_type_ptr: *const u8,
    event_type_len: usize,
) -> i32 {
    if event_type_ptr.is_null() {
        return 0;
    }
    let event_type = std::slice::from_raw_parts(event_type_ptr, event_type_len);
    match std::str::from_utf8(event_type) {
        Ok(value) if is_openai_ws_terminal_event(value) => 1,
        _ => 0,
    }
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_is_token_event(
    event_type_ptr: *const u8,
    event_type_len: usize,
) -> i32 {
    if event_type_ptr.is_null() {
        return 0;
    }
    let event_type = std::slice::from_raw_parts(event_type_ptr, event_type_len);
    match std::str::from_utf8(event_type) {
        Ok(value) if is_openai_ws_token_event(value) => 1,
        _ => 0,
    }
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_message_likely_contains_tool_calls(
    input_ptr: *const u8,
    input_len: usize,
) -> i32 {
    if input_ptr.is_null() {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    if openai_ws_message_likely_contains_tool_calls(input) {
        1
    } else {
        0
    }
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_replace_model(
    input_ptr: *const u8,
    input_len: usize,
    from_model_ptr: *const u8,
    from_model_len: usize,
    to_model_ptr: *const u8,
    to_model_len: usize,
    output_ptr: *mut u8,
    output_len: usize,
) -> i32 {
    if input_ptr.is_null() || from_model_ptr.is_null() || to_model_ptr.is_null() {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let from_model = std::str::from_utf8(std::slice::from_raw_parts(from_model_ptr, from_model_len)).ok();
    let to_model = std::str::from_utf8(std::slice::from_raw_parts(to_model_ptr, to_model_len)).ok();
    let Some((from_model, to_model)) = from_model.zip(to_model) else {
        return 0;
    };
    let Some(updated) = replace_openai_ws_message_model(input, from_model, to_model) else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), output_ptr, output_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_drop_previous_response_id(
    input_ptr: *const u8,
    input_len: usize,
    output_ptr: *mut u8,
    output_len: usize,
) -> i32 {
    if input_ptr.is_null() {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let Some(updated) = drop_previous_response_id(input) else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), output_ptr, output_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_set_previous_response_id(
    input_ptr: *const u8,
    input_len: usize,
    previous_response_id_ptr: *const u8,
    previous_response_id_len: usize,
    output_ptr: *mut u8,
    output_len: usize,
) -> i32 {
    if input_ptr.is_null() || previous_response_id_ptr.is_null() {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let prev = std::str::from_utf8(std::slice::from_raw_parts(previous_response_id_ptr, previous_response_id_len)).ok();
    let Some(previous_response_id) = prev else {
        return 0;
    };
    let Some(updated) = set_previous_response_id(input, previous_response_id) else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), output_ptr, output_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_set_request_type(
    input_ptr: *const u8,
    input_len: usize,
    event_type_ptr: *const u8,
    event_type_len: usize,
    output_ptr: *mut u8,
    output_len: usize,
) -> i32 {
    if input_ptr.is_null() || event_type_ptr.is_null() {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let event_type = std::str::from_utf8(std::slice::from_raw_parts(event_type_ptr, event_type_len)).ok();
    let Some(event_type) = event_type else {
        return 0;
    };
    let Some(updated) = set_openai_ws_request_type(input, event_type) else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), output_ptr, output_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_set_turn_metadata(
    input_ptr: *const u8,
    input_len: usize,
    turn_metadata_ptr: *const u8,
    turn_metadata_len: usize,
    output_ptr: *mut u8,
    output_len: usize,
) -> i32 {
    if input_ptr.is_null() || turn_metadata_ptr.is_null() {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let turn_metadata = std::str::from_utf8(std::slice::from_raw_parts(turn_metadata_ptr, turn_metadata_len)).ok();
    let Some(turn_metadata) = turn_metadata else {
        return 0;
    };
    let Some(updated) = set_openai_ws_turn_metadata(input, turn_metadata) else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), output_ptr, output_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_set_input_sequence(
    input_ptr: *const u8,
    input_len: usize,
    input_sequence_ptr: *const u8,
    input_sequence_len: usize,
    output_ptr: *mut u8,
    output_len: usize,
) -> i32 {
    if input_ptr.is_null() || input_sequence_ptr.is_null() {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let input_sequence_json = std::slice::from_raw_parts(input_sequence_ptr, input_sequence_len);
    let Some(updated) = set_openai_ws_input_sequence(input, input_sequence_json) else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), output_ptr, output_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_normalize_payload_without_input_and_previous_response_id(
    input_ptr: *const u8,
    input_len: usize,
    output_ptr: *mut u8,
    output_len: usize,
) -> i32 {
    if input_ptr.is_null() {
        return 0;
    }
    let input = std::slice::from_raw_parts(input_ptr, input_len);
    let Some(updated) = normalize_openai_ws_payload_without_input_and_previous_response_id(input) else {
        return 0;
    };
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), output_ptr, output_len) {
        return 0;
    }
    1
}

#[no_mangle]
pub unsafe extern "C" fn streamcore_openai_ws_build_replay_input_sequence(
    previous_full_input_ptr: *const u8,
    previous_full_input_len: usize,
    previous_full_input_exists: i32,
    current_payload_ptr: *const u8,
    current_payload_len: usize,
    has_previous_response_id: i32,
    output_ptr: *mut u8,
    output_len: usize,
    exists_out: *mut i32,
) -> i32 {
    if current_payload_ptr.is_null() || exists_out.is_null() {
        return 0;
    }
    let previous_full_input_json = if previous_full_input_ptr.is_null() {
        &[][..]
    } else {
        std::slice::from_raw_parts(previous_full_input_ptr, previous_full_input_len)
    };
    let current_payload = std::slice::from_raw_parts(current_payload_ptr, current_payload_len);
    let Some((updated, exists)) = build_openai_ws_replay_input_sequence(
        previous_full_input_json,
        previous_full_input_exists != 0,
        current_payload,
        has_previous_response_id != 0,
    ) else {
        return 0;
    };
    *exists_out = if exists { 1 } else { 0 };
    if !exists {
        return 1;
    }
    if !write_c_string(std::str::from_utf8(&updated).unwrap_or_default(), output_ptr, output_len) {
        return 0;
    }
    1
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sha256_hex_matches_expected_value() {
        assert_eq!(
            sha256_hex(b"sub2api"),
            "7e00c4e93784ee94268cd479b51ca0633d3fd2311d165b17b962a8d04806ab88"
        );
    }

    #[test]
    fn build_sse_data_frame_wraps_payload() {
        let frame = build_sse_data_frame(br#"{"type":"response.created"}"#);
        assert_eq!(
            std::str::from_utf8(&frame).unwrap(),
            "data: {\"type\":\"response.created\"}\n\n"
        );
    }

    #[test]
    fn correct_openai_tool_calls_rewrites_names_and_args() {
        let input = br#"{"tool_calls":[{"function":{"name":"apply_patch","arguments":"{\"path\":\"/tmp/a.txt\",\"old_string\":\"a\",\"new_string\":\"b\"}"}}]}"#;
        let updated = correct_openai_tool_calls(input).expect("correct tool calls");
        let payload: Value = serde_json::from_slice(&updated).expect("parse corrected payload");
        let function = payload
            .get("tool_calls")
            .and_then(Value::as_array)
            .and_then(|items| items.first())
            .and_then(|item| item.get("function"))
            .and_then(Value::as_object)
            .expect("function object");
        assert_eq!(function.get("name").and_then(Value::as_str), Some("edit"));
        let args_raw = function
            .get("arguments")
            .and_then(Value::as_str)
            .expect("arguments string");
        let args: Value = serde_json::from_str(args_raw).expect("parse arguments json");
        assert_eq!(args.get("filePath").and_then(Value::as_str), Some("/tmp/a.txt"));
        assert_eq!(args.get("oldString").and_then(Value::as_str), Some("a"));
        assert_eq!(args.get("newString").and_then(Value::as_str), Some("b"));
    }

    #[test]
    fn rewrite_openai_ws_message_for_client_combines_model_and_tool_rewrites() {
        let input = br#"{"type":"response.completed","model":"gpt-5.1","response":{"model":"gpt-5.1"},"tool_calls":[{"function":{"name":"apply_patch","arguments":"{\"path\":\"/tmp/a.txt\",\"old_string\":\"a\",\"new_string\":\"b\"}"}}]}"#;
        let updated = rewrite_openai_ws_message_for_client(input, "gpt-5.1", "custom-model", true)
            .expect("rewrite message");
        let payload: Value = serde_json::from_slice(&updated).expect("parse rewritten payload");
        assert_eq!(payload.get("model").and_then(Value::as_str), Some("custom-model"));
        assert_eq!(
            payload
                .get("response")
                .and_then(|v| v.get("model"))
                .and_then(Value::as_str),
            Some("custom-model")
        );
        let function = payload
            .get("tool_calls")
            .and_then(Value::as_array)
            .and_then(|items| items.first())
            .and_then(|item| item.get("function"))
            .and_then(Value::as_object)
            .expect("function object");
        assert_eq!(function.get("name").and_then(Value::as_str), Some("edit"));
    }

    #[test]
    fn rewrite_openai_sse_line_for_client_rewrites_data_line() {
        let input = br#"data: {"model":"gpt-5.1","tool_calls":[{"function":{"name":"apply_patch"}}]}"#;
        let updated = rewrite_openai_sse_line_for_client(input, "gpt-5.1", "custom-model", true)
            .expect("rewrite sse line");
        let updated_str = std::str::from_utf8(&updated).unwrap();
        assert!(updated_str.starts_with("data: "));
        assert!(updated_str.contains(r#""model":"custom-model""#));
        assert!(updated_str.contains(r#""name":"edit""#));
    }

    #[test]
    fn rewrite_openai_sse_body_for_client_rewrites_only_data_lines() {
        let input = b"event: message\ndata: {\"model\":\"gpt-5.1\",\"tool_calls\":[{\"function\":{\"name\":\"apply_patch\"}}]}\n\ndata: [DONE]\n";
        let updated =
            rewrite_openai_sse_body_for_client(input, "gpt-5.1", "custom-model", true).expect("rewrite sse body");
        let updated_str = std::str::from_utf8(&updated).unwrap();
        assert!(updated_str.contains("event: message"));
        assert!(updated_str.contains(r#"data: {"model":"custom-model","tool_calls":[{"function":{"name":"edit"}}"#));
        assert!(updated_str.contains("data: [DONE]"));
    }

    #[test]
    fn rewrite_openai_ws_message_to_sse_frame_for_client_rewrites_and_wraps() {
        let input = br#"{"model":"gpt-5.1","tool_calls":[{"function":{"name":"apply_patch"}}]}"#;
        let updated =
            rewrite_openai_ws_message_to_sse_frame_for_client(input, "gpt-5.1", "custom-model", true)
                .expect("rewrite ws message to sse frame");
        let updated_str = std::str::from_utf8(&updated).unwrap();
        assert!(updated_str.starts_with("data: "));
        assert!(updated_str.ends_with("\n\n"));
        assert!(updated_str.contains(r#""model":"custom-model""#));
        assert!(updated_str.contains(r#""name":"edit""#));
    }

    #[test]
    fn gzip_roundtrip() {
        let input = b"sub2api-streamcore-gzip";
        let compressed = gzip_compress(input).expect("gzip compress");
        let restored = gzip_decompress(&compressed).expect("gzip decompress");
        assert_eq!(restored, input);
    }

    #[test]
    fn brotli_roundtrip() {
        let input = b"sub2api-streamcore-brotli";
        let compressed = brotli_compress(input, 4).expect("brotli compress");
        let restored = brotli_decompress(&compressed).expect("brotli decompress");
        assert_eq!(restored, input);
    }

    #[test]
    fn parse_openai_ws_usage_fields_extracts_values() {
        let input = br#"{"response":{"usage":{"input_tokens":11,"output_tokens":22,"input_tokens_details":{"cached_tokens":7}}}}"#;
        let parsed = parse_openai_ws_usage_fields(input).expect("parse usage");
        assert_eq!(parsed, (11, 22, 7));
    }

    #[test]
    fn parse_openai_ws_event_envelope_extracts_values() {
        let input = br#"{"type":"response.completed","response":{"id":"resp_123","usage":{"input_tokens":1}}}"#;
        let parsed = parse_openai_ws_event_envelope(input).expect("parse envelope");
        assert_eq!(parsed.0, "response.completed");
        assert_eq!(parsed.1, "resp_123");
        assert!(parsed.2.contains("\"id\":\"resp_123\""));
    }

    #[test]
    fn parse_openai_ws_error_fields_extracts_values() {
        let input = br#"{"error":{"code":"rate_limit","type":"rate_limit_error","message":"slow down"}}"#;
        let parsed = parse_openai_ws_error_fields(input).expect("parse error");
        assert_eq!(parsed, ("rate_limit".to_string(), "rate_limit_error".to_string(), "slow down".to_string()));
    }

    #[test]
    fn parse_openai_ws_request_payload_summary_extracts_values() {
        let input = br#"{"type":"response.create","model":"gpt-5.1","stream":false,"prompt_cache_key":" cache-key ","previous_response_id":" resp_1 ","input":[{"type":"function_call_output","call_id":"call_1"}]}"#;
        let parsed = parse_openai_ws_request_payload_summary(input).expect("parse request summary");
        assert_eq!(parsed.0, "response.create");
        assert_eq!(parsed.1, "gpt-5.1");
        assert_eq!(parsed.2, "cache-key");
        assert_eq!(parsed.3, "resp_1");
        assert!(parsed.4);
        assert!(!parsed.5);
        assert!(parsed.6);
    }

    #[test]
    fn terminal_event_detection_matches_expected_values() {
        assert!(is_openai_ws_terminal_event("response.completed"));
        assert!(is_openai_ws_terminal_event("response.canceled"));
        assert!(!is_openai_ws_terminal_event("response.created"));
    }

    #[test]
    fn token_event_detection_matches_expected_values() {
        assert!(is_openai_ws_token_event("response.output_text.delta"));
        assert!(is_openai_ws_token_event("response.done"));
        assert!(!is_openai_ws_token_event("response.created"));
        assert!(!is_openai_ws_token_event(""));
    }

    #[test]
    fn tool_call_detection_matches_expected_values() {
        assert!(!openai_ws_message_likely_contains_tool_calls(
            br#"{"type":"response.output_text.delta","delta":"hello"}"#,
        ));
        assert!(openai_ws_message_likely_contains_tool_calls(
            br#"{"type":"response.output_item.added","item":{"tool_calls":[{"id":"tc1"}]}}"#,
        ));
        assert!(openai_ws_message_likely_contains_tool_calls(
            br#"{"type":"response.output_item.added","item":{"type":"function_call"}}"#,
        ));
    }

    #[test]
    fn replace_model_preserves_shape() {
        let input = br#"{"type":"response.created","model":"gpt-5.1"}"#;
        let updated = replace_openai_ws_message_model(input, "gpt-5.1", "custom-model").expect("replace model");
        assert_eq!(std::str::from_utf8(&updated).unwrap(), r#"{"type":"response.created","model":"custom-model"}"#);
    }

    #[test]
    fn drop_previous_response_id_removes_field() {
        let input = br#"{"type":"response.create","previous_response_id":"resp_old","input":[]}"#;
        let updated = drop_previous_response_id(input).expect("drop prev");
        assert_eq!(std::str::from_utf8(&updated).unwrap(), r#"{"type":"response.create","input":[]}"#);
    }

    #[test]
    fn set_previous_response_id_sets_field() {
        let input = br#"{"type":"response.create","input":[]}"#;
        let updated = set_previous_response_id(input, "resp_new").expect("set prev");
        assert_eq!(std::str::from_utf8(&updated).unwrap(), r#"{"type":"response.create","input":[],"previous_response_id":"resp_new"}"#);
    }

    #[test]
    fn set_request_type_sets_field() {
        let input = br#"{"model":"gpt-5.1","input":[]}"#;
        let updated = set_openai_ws_request_type(input, "response.create").expect("set type");
        assert_eq!(std::str::from_utf8(&updated).unwrap(), r#"{"model":"gpt-5.1","input":[],"type":"response.create"}"#);
    }

    #[test]
    fn set_turn_metadata_sets_nested_field() {
        let input = br#"{"type":"response.create","model":"gpt-5.1"}"#;
        let updated = set_openai_ws_turn_metadata(input, "turn_meta_1").expect("set turn metadata");
        assert_eq!(
            std::str::from_utf8(&updated).unwrap(),
            r#"{"type":"response.create","model":"gpt-5.1","client_metadata":{"x-codex-turn-metadata":"turn_meta_1"}}"#
        );
    }

    #[test]
    fn set_input_sequence_sets_input_field() {
        let input = br#"{"type":"response.create","previous_response_id":"resp_1"}"#;
        let updated = set_openai_ws_input_sequence(
            input,
            br#"[{"type":"input_text","text":"hello"},{"type":"input_text","text":"world"}]"#,
        )
        .expect("set input sequence");
        assert_eq!(
            std::str::from_utf8(&updated).unwrap(),
            r#"{"type":"response.create","previous_response_id":"resp_1","input":[{"type":"input_text","text":"hello"},{"type":"input_text","text":"world"}]}"#
        );
    }

    #[test]
    fn normalize_payload_without_input_and_previous_response_id_removes_fields() {
        let input = br#"{"model":"gpt-5.1","input":[1],"previous_response_id":"resp_x","metadata":{"b":2,"a":1}}"#;
        let updated = normalize_openai_ws_payload_without_input_and_previous_response_id(input)
            .expect("normalize payload");
        assert_eq!(
            std::str::from_utf8(&updated).unwrap(),
            r#"{"model":"gpt-5.1","metadata":{"b":2,"a":1}}"#
        );
    }

    #[test]
    fn build_replay_input_sequence_appends_delta_when_needed() {
        let previous = br#"[{"type":"input_text","text":"hello"}]"#;
        let current = br#"{"previous_response_id":"resp_1","input":[{"type":"input_text","text":"world"}]}"#;
        let (updated, exists) =
            build_openai_ws_replay_input_sequence(previous, true, current, true).expect("build replay");
        assert!(exists);
        assert_eq!(
            std::str::from_utf8(&updated).unwrap(),
            r#"[{"type":"input_text","text":"hello"},{"type":"input_text","text":"world"}]"#
        );
    }

    #[test]
    fn parse_openai_ws_frame_summary_extracts_combined_fields() {
        let input = br#"{"type":"response.completed","response":{"id":"resp_123","usage":{"input_tokens":11,"output_tokens":22,"input_tokens_details":{"cached_tokens":7}},"tool_calls":[{"id":"tc1"}]},"tool_calls":[{"id":"tc1"}]}"#;
        let parsed = parse_openai_ws_frame_summary(input).expect("frame summary");
        assert_eq!(parsed.0, "response.completed");
        assert_eq!(parsed.1, "resp_123");
        assert!(parsed.2.contains("\"id\":\"resp_123\""));
        assert_eq!(parsed.6, 11);
        assert_eq!(parsed.7, 22);
        assert_eq!(parsed.8, 7);
        assert!(parsed.9);
        assert!(parsed.10);
        assert!(parsed.11);
    }

    #[test]
    fn parse_openai_sse_body_summary_extracts_terminal_and_usage() {
        let body = b"event: message\ndata: {\"type\":\"response.in_progress\",\"response\":{\"id\":\"resp_1\"}}\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":11,\"output_tokens\":22,\"input_tokens_details\":{\"cached_tokens\":3}}}}\ndata: [DONE]\n";
        let parsed = parse_openai_sse_body_summary(body).expect("sse body summary");
        assert_eq!(parsed.0, "response.completed");
        assert!(parsed.1.contains("\"type\":\"response.completed\""));
        assert!(parsed.2.contains("\"id\":\"resp_1\""));
        assert_eq!(parsed.3, 11);
        assert_eq!(parsed.4, 22);
        assert_eq!(parsed.5, 3);
        assert!(parsed.6);
        assert!(parsed.7);
    }
}
