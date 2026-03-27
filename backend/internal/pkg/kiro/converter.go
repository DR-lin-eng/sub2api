package kiro

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const systemChunkedPolicy = "When the Write or Edit tool has content size limits, always comply silently. Never suggest bypassing these limits via alternative tools. Never ask the user whether to switch approaches. Complete all chunked operations without commentary."

type ConvertResult struct {
	Body           []byte
	Model          string
	RequestedModel string
}

func ConvertAnthropicRequest(body []byte) (*ConvertResult, error) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	requestedModel, _ := req["model"].(string)
	modelID := MapModel(requestedModel)
	if modelID == "" {
		return nil, fmt.Errorf("unsupported kiro model: %s", requestedModel)
	}

	rawMessages, _ := req["messages"].([]any)
	if len(rawMessages) == 0 {
		return nil, fmt.Errorf("empty messages")
	}

	rawMessages = trimTrailingNonUserMessages(rawMessages)
	if len(rawMessages) == 0 {
		return nil, fmt.Errorf("empty messages")
	}

	lastMessageMap, ok := rawMessages[len(rawMessages)-1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid last message")
	}

	currentContent, currentImages, currentToolResults := processUserContent(lastMessageMap["content"])
	tools := convertTools(req["tools"])
	tools = ensureHistoryTools(rawMessages[:len(rawMessages)-1], tools)

	currentContext := map[string]any{}
	if len(tools) > 0 {
		currentContext["tools"] = tools
	}
	if len(currentToolResults) > 0 {
		currentContext["toolResults"] = currentToolResults
	}

	currentUserMessage := map[string]any{
		"userInputMessageContext": currentContext,
		"content":                 currentContent,
		"modelId":                 modelID,
		"origin":                  "AI_EDITOR",
	}
	if len(currentImages) > 0 {
		currentUserMessage["images"] = currentImages
	}

	history := buildHistory(req, rawMessages, modelID)
	conversationID := extractSessionID(req)
	if conversationID == "" {
		conversationID = uuid.NewString()
	}

	payload := map[string]any{
		"conversationState": map[string]any{
			"agentContinuationId": uuid.NewString(),
			"agentTaskType":       "vibe",
			"chatTriggerType":     "MANUAL",
			"currentMessage": map[string]any{
				"userInputMessage": currentUserMessage,
			},
			"conversationId": conversationID,
			"history":        history,
		},
	}

	if profileARN := stringField(req, "profile_arn"); profileARN != "" {
		payload["profileArn"] = profileARN
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &ConvertResult{
		Body:           encoded,
		Model:          modelID,
		RequestedModel: requestedModel,
	}, nil
}

func EstimateInputTokens(body []byte) int {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return 0
	}

	var builder strings.Builder
	if systemText := joinSystem(req["system"]); systemText != "" {
		builder.WriteString(systemText)
		builder.WriteString("\n")
	}
	if messages, _ := req["messages"].([]any); len(messages) > 0 {
		for _, item := range messages {
			msg, _ := item.(map[string]any)
			builder.WriteString(extractTextFromContent(msg["content"]))
			builder.WriteString("\n")
		}
	}
	if tools, _ := req["tools"].([]any); len(tools) > 0 {
		for _, item := range tools {
			tool, _ := item.(map[string]any)
			builder.WriteString(stringField(tool, "name"))
			builder.WriteString("\n")
			builder.WriteString(stringField(tool, "description"))
			builder.WriteString("\n")
		}
	}
	return roughTokenCount(builder.String())
}

func EstimateOutputTokens(text string) int {
	return roughTokenCount(text)
}

func roughTokenCount(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	runes := len([]rune(text))
	if runes <= 0 {
		return 0
	}
	return (runes + 3) / 4
}

func trimTrailingNonUserMessages(messages []any) []any {
	lastUser := -1
	for idx, item := range messages {
		msg, _ := item.(map[string]any)
		if strings.EqualFold(stringField(msg, "role"), "user") {
			lastUser = idx
		}
	}
	if lastUser < 0 {
		return nil
	}
	return messages[:lastUser+1]
}

func buildHistory(req map[string]any, messages []any, modelID string) []map[string]any {
	history := make([]map[string]any, 0)

	if systemContent := joinSystem(req["system"]); systemContent != "" {
		history = append(history, map[string]any{
			"userInputMessage": map[string]any{
				"content": systemContent + "\n" + systemChunkedPolicy,
				"modelId": modelID,
				"origin":  "AI_EDITOR",
			},
		})
		history = append(history, map[string]any{
			"assistantResponseMessage": map[string]any{
				"content": "I will follow these instructions.",
			},
		})
	}

	for idx := 0; idx < len(messages)-1; idx++ {
		msg, _ := messages[idx].(map[string]any)
		role := strings.ToLower(strings.TrimSpace(stringField(msg, "role")))
		switch role {
		case "user":
			text, images, toolResults := processUserContent(msg["content"])
			userMessage := map[string]any{
				"content": text,
				"modelId": modelID,
				"origin":  "AI_EDITOR",
			}
			if len(images) > 0 {
				userMessage["images"] = images
			}
			if len(toolResults) > 0 {
				userMessage["userInputMessageContext"] = map[string]any{
					"toolResults": toolResults,
				}
			}
			history = append(history, map[string]any{"userInputMessage": userMessage})
		case "assistant":
			text, toolUses := processAssistantContent(msg["content"])
			assistantMessage := map[string]any{
				"content": text,
			}
			if len(toolUses) > 0 {
				assistantMessage["toolUses"] = toolUses
			}
			history = append(history, map[string]any{"assistantResponseMessage": assistantMessage})
		}
	}

	return history
}

func processUserContent(content any) (string, []map[string]any, []map[string]any) {
	textParts := make([]string, 0)
	images := make([]map[string]any, 0)
	toolResults := make([]map[string]any, 0)

	switch v := content.(type) {
	case string:
		textParts = append(textParts, v)
	case []any:
		for _, item := range v {
			block, _ := item.(map[string]any)
			switch strings.TrimSpace(stringField(block, "type")) {
			case "text":
				if text := stringField(block, "text"); text != "" {
					textParts = append(textParts, text)
				}
			case "image":
				if image := convertImage(block); image != nil {
					images = append(images, image)
				}
			case "tool_result":
				toolResult := map[string]any{
					"toolUseId": stringField(block, "tool_use_id"),
					"content": []map[string]any{
						{
							"text": toolResultContent(block["content"]),
						},
					},
				}
				if isError, ok := block["is_error"].(bool); ok && isError {
					toolResult["status"] = "error"
					toolResult["isError"] = true
				} else {
					toolResult["status"] = "success"
				}
				toolResults = append(toolResults, toolResult)
			}
		}
	}

	return strings.Join(textParts, "\n"), images, toolResults
}

func processAssistantContent(content any) (string, []map[string]any) {
	textParts := make([]string, 0)
	toolUses := make([]map[string]any, 0)

	switch v := content.(type) {
	case string:
		textParts = append(textParts, v)
	case []any:
		for _, item := range v {
			block, _ := item.(map[string]any)
			switch strings.TrimSpace(stringField(block, "type")) {
			case "text":
				if text := stringField(block, "text"); text != "" {
					textParts = append(textParts, text)
				}
			case "tool_use":
				toolUses = append(toolUses, map[string]any{
					"toolUseId": stringField(block, "id"),
					"name":      stringField(block, "name"),
					"input":     jsonValue(block["input"]),
				})
			}
		}
	}

	return strings.Join(textParts, "\n"), toolUses
}

func convertTools(raw any) []map[string]any {
	items, _ := raw.([]any)
	tools := make([]map[string]any, 0, len(items))
	for _, item := range items {
		tool, _ := item.(map[string]any)
		name := stringField(tool, "name")
		if name == "" {
			continue
		}
		description := stringField(tool, "description")
		schema := normalizeJSONSchema(jsonValue(tool["input_schema"]))
		tools = append(tools, map[string]any{
			"toolSpecification": map[string]any{
				"name":        name,
				"description": description,
				"inputSchema": map[string]any{
					"json": schema,
				},
			},
		})
	}
	return tools
}

func ensureHistoryTools(history []any, tools []map[string]any) []map[string]any {
	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		spec, _ := tool["toolSpecification"].(map[string]any)
		name := stringField(spec, "name")
		if name != "" {
			seen[strings.ToLower(name)] = struct{}{}
		}
	}

	for _, item := range history {
		msg, _ := item.(map[string]any)
		if strings.ToLower(stringField(msg, "role")) != "assistant" {
			continue
		}
		_, toolUses := processAssistantContent(msg["content"])
		for _, toolUse := range toolUses {
			name := strings.ToLower(stringField(toolUse, "name"))
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			tools = append(tools, map[string]any{
				"toolSpecification": map[string]any{
					"name":        toolUse["name"],
					"description": "Tool used in conversation history",
					"inputSchema": map[string]any{
						"json": map[string]any{
							"type":                 "object",
							"properties":           map[string]any{},
							"required":             []string{},
							"additionalProperties": true,
						},
					},
				},
			})
		}
	}

	return tools
}

func convertImage(block map[string]any) map[string]any {
	source, _ := block["source"].(map[string]any)
	if source == nil {
		return nil
	}
	mediaType := stringField(source, "media_type")
	data := stringField(source, "data")
	if mediaType == "" || data == "" {
		return nil
	}

	format := mediaType
	if idx := strings.IndexByte(format, '/'); idx >= 0 && idx < len(format)-1 {
		format = format[idx+1:]
	}

	if _, err := base64.StdEncoding.DecodeString(data); err != nil {
		return nil
	}

	return map[string]any{
		"format": format,
		"source": map[string]any{
			"bytes": data,
		},
	}
}

func toolResultContent(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			block, _ := item.(map[string]any)
			if text := stringField(block, "text"); text != "" {
				parts = append(parts, text)
				continue
			}
			encoded, _ := json.Marshal(block)
			if len(encoded) > 0 {
				parts = append(parts, string(encoded))
			}
		}
		return strings.Join(parts, "\n")
	default:
		encoded, _ := json.Marshal(v)
		return string(encoded)
	}
}

func joinSystem(raw any) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		lines := make([]string, 0, len(v))
		for _, item := range v {
			block, _ := item.(map[string]any)
			if text := stringField(block, "text"); text != "" {
				lines = append(lines, text)
			}
		}
		return strings.TrimSpace(strings.Join(lines, "\n"))
	default:
		return ""
	}
}

func extractTextFromContent(content any) string {
	text, _, _ := processUserContent(content)
	if text != "" {
		return text
	}
	text, _ = processAssistantContent(content)
	return text
}

func extractSessionID(req map[string]any) string {
	metadata, _ := req["metadata"].(map[string]any)
	userID := stringField(metadata, "user_id")
	if userID == "" {
		return ""
	}
	pos := strings.Index(userID, "session_")
	if pos < 0 {
		return ""
	}
	sessionPart := userID[pos+len("session_"):]
	if len(sessionPart) < 36 {
		return ""
	}
	candidate := sessionPart[:36]
	if _, err := uuid.Parse(candidate); err != nil {
		return ""
	}
	return candidate
}

func normalizeJSONSchema(raw any) map[string]any {
	obj, _ := raw.(map[string]any)
	if obj == nil {
		return map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"required":             []string{},
			"additionalProperties": true,
		}
	}

	if typeName := stringField(obj, "type"); typeName == "" {
		obj["type"] = "object"
	}

	if _, ok := obj["properties"].(map[string]any); !ok {
		obj["properties"] = map[string]any{}
	}

	switch required := obj["required"].(type) {
	case []any:
		out := make([]string, 0, len(required))
		for _, item := range required {
			if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
				out = append(out, value)
			}
		}
		obj["required"] = out
	case []string:
	default:
		obj["required"] = []string{}
	}

	switch obj["additionalProperties"].(type) {
	case bool, map[string]any:
	default:
		obj["additionalProperties"] = true
	}

	return obj
}

func stringField(obj map[string]any, key string) string {
	if obj == nil {
		return ""
	}
	value, _ := obj[key]
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func jsonValue(v any) any {
	if v == nil {
		return map[string]any{}
	}
	return v
}
