package service

import (
	"fmt"

	"github.com/tidwall/gjson"
)

const claudeMimicDefaultCacheControlTTL = "5m"

// stripMessageCacheControl removes client-provided message cache breakpoints.
// The proxy adds stable breakpoints afterward so multi-turn prefixes stay cacheable.
func stripMessageCacheControl(body []byte) []byte {
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}

	msgIdx := -1
	messages.ForEach(func(_, msg gjson.Result) bool {
		msgIdx++
		content := msg.Get("content")
		if !content.IsArray() {
			return true
		}

		blockIdx := -1
		content.ForEach(func(_, block gjson.Result) bool {
			blockIdx++
			if !block.Get("cache_control").Exists() {
				return true
			}
			if next, ok := deleteJSONPathBytes(body, fmt.Sprintf("messages.%d.content.%d.cache_control", msgIdx, blockIdx)); ok {
				body = next
			}
			return true
		})
		return true
	})
	return body
}

// addMessageCacheBreakpoints injects cache breakpoints on the last message and
// the second-to-last user turn when the conversation is long enough.
func addMessageCacheBreakpoints(body []byte) []byte {
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	arr := messages.Array()
	if len(arr) == 0 {
		return body
	}

	body = injectCacheControlOnLastContentBlock(body, len(arr)-1, arr[len(arr)-1])

	if len(arr) >= 4 {
		userCount := 0
		for i := len(arr) - 1; i >= 0; i-- {
			if arr[i].Get("role").String() != "user" {
				continue
			}
			userCount++
			if userCount == 2 {
				body = injectCacheControlOnLastContentBlock(body, i, arr[i])
				break
			}
		}
	}

	return body
}

func injectCacheControlOnLastContentBlock(body []byte, idx int, msg gjson.Result) []byte {
	content := msg.Get("content")
	if content.Type == gjson.String {
		raw := fmt.Sprintf(
			`[{"type":"text","text":%s,"cache_control":{"type":"ephemeral","ttl":%q}}]`,
			mustJSONString(content.String()),
			claudeMimicDefaultCacheControlTTL,
		)
		if next, ok := setJSONRawBytes(body, fmt.Sprintf("messages.%d.content", idx), []byte(raw)); ok {
			body = next
		}
		return body
	}

	if !content.IsArray() {
		return body
	}
	contentArr := content.Array()
	if len(contentArr) == 0 {
		return body
	}

	lastBlockIdx := len(contentArr) - 1
	lastBlock := contentArr[lastBlockIdx]
	existingCC := lastBlock.Get("cache_control")
	if existingCC.Exists() && existingCC.Get("ttl").String() != "" {
		return body
	}

	path := fmt.Sprintf("messages.%d.content.%d.cache_control", idx, lastBlockIdx)
	if existingCC.Exists() {
		if next, ok := setJSONValueBytes(body, path+".ttl", claudeMimicDefaultCacheControlTTL); ok {
			body = next
		}
		return body
	}

	raw := fmt.Sprintf(`{"type":"ephemeral","ttl":%q}`, claudeMimicDefaultCacheControlTTL)
	if next, ok := setJSONRawBytes(body, path, []byte(raw)); ok {
		body = next
	}
	return body
}

func mustJSONString(s string) string {
	return fmt.Sprintf("%q", s)
}
