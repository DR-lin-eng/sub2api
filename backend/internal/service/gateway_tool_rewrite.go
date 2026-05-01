package service

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const toolNameRewriteKey = "claude_tool_name_rewrite"

var staticToolNameRewrites = map[string]string{
	"sessions_": "cc_sess_",
	"session_":  "cc_ses_",
}

var fakeToolNamePrefixes = []string{
	"analyze_", "compute_", "fetch_", "generate_", "lookup_", "modify_",
	"process_", "query_", "render_", "resolve_", "sync_", "update_",
	"validate_", "convert_", "extract_", "manage_", "monitor_", "parse_",
	"review_", "search_", "transform_", "handle_", "invoke_", "notify_",
}

const dynamicToolMapThreshold = 5

type ToolNameRewrite struct {
	Forward        map[string]string
	Reverse        map[string]string
	ReverseOrdered [][2]string
}

func buildDynamicToolMap(toolNames []string) map[string]string {
	if len(toolNames) <= dynamicToolMapThreshold {
		return nil
	}

	h := fnv.New64a()
	for i, name := range toolNames {
		if i > 0 {
			_, _ = h.Write([]byte{0})
		}
		_, _ = h.Write([]byte(name))
	}

	available := make([]string, len(fakeToolNamePrefixes))
	copy(available, fakeToolNamePrefixes)
	rng := rand.New(rand.NewSource(int64(h.Sum64())))
	rng.Shuffle(len(available), func(i, j int) {
		available[i], available[j] = available[j], available[i]
	})

	mapping := make(map[string]string, len(toolNames))
	for i, name := range toolNames {
		headLen := 3
		if len(name) < headLen {
			headLen = len(name)
		}
		mapping[name] = fmt.Sprintf("%s%s%02d", available[i%len(available)], name[:headLen], i)
	}
	return mapping
}

func sanitizeToolName(name string, dynamic map[string]string) string {
	if dynamic != nil {
		if fake, ok := dynamic[name]; ok {
			return fake
		}
	}
	for prefix, replacement := range staticToolNameRewrites {
		if strings.HasPrefix(name, prefix) {
			return replacement + name[len(prefix):]
		}
	}
	return name
}

func shouldMimicToolName(toolType string) bool {
	return toolType == "" || toolType == "function" || toolType == "custom"
}

func buildToolNameRewriteFromBody(body []byte) *ToolNameRewrite {
	tools := gjson.GetBytes(body, "tools")
	if !tools.IsArray() {
		return nil
	}

	toolNames := make([]string, 0)
	tools.ForEach(func(_, tool gjson.Result) bool {
		if !shouldMimicToolName(tool.Get("type").String()) {
			return true
		}
		name := tool.Get("name").String()
		if name != "" {
			toolNames = append(toolNames, name)
		}
		return true
	})

	dynamic := buildDynamicToolMap(toolNames)
	rw := &ToolNameRewrite{
		Forward: make(map[string]string),
		Reverse: make(map[string]string),
	}
	for _, name := range toolNames {
		fake := sanitizeToolName(name, dynamic)
		if fake == name {
			continue
		}
		rw.Forward[name] = fake
		rw.Reverse[fake] = name
	}
	if len(rw.Forward) == 0 {
		return nil
	}

	rw.ReverseOrdered = make([][2]string, 0, len(rw.Reverse))
	for fake, real := range rw.Reverse {
		rw.ReverseOrdered = append(rw.ReverseOrdered, [2]string{fake, real})
	}
	sort.SliceStable(rw.ReverseOrdered, func(i, j int) bool {
		return len(rw.ReverseOrdered[i][0]) > len(rw.ReverseOrdered[j][0])
	})
	return rw
}

func applyToolNameRewriteToBody(body []byte, rw *ToolNameRewrite) []byte {
	if rw == nil || len(rw.Forward) == 0 {
		return applyToolsLastCacheBreakpoint(body)
	}

	tools := gjson.GetBytes(body, "tools")
	if tools.IsArray() {
		idx := -1
		tools.ForEach(func(_, tool gjson.Result) bool {
			idx++
			if !shouldMimicToolName(tool.Get("type").String()) {
				return true
			}
			name := tool.Get("name").String()
			fake, ok := rw.Forward[name]
			if !ok {
				return true
			}
			if next, err := sjson.SetBytes(body, fmt.Sprintf("tools.%d.name", idx), fake); err == nil {
				body = next
			}
			return true
		})
	}

	if toolChoice := gjson.GetBytes(body, "tool_choice"); toolChoice.Exists() && toolChoice.Get("type").String() == "tool" {
		name := toolChoice.Get("name").String()
		if fake, ok := rw.Forward[name]; ok {
			if next, err := sjson.SetBytes(body, "tool_choice.name", fake); err == nil {
				body = next
			}
		}
	}

	return applyToolsLastCacheBreakpoint(body)
}

func applyToolsLastCacheBreakpoint(body []byte) []byte {
	tools := gjson.GetBytes(body, "tools")
	if !tools.IsArray() {
		return body
	}
	arr := tools.Array()
	if len(arr) == 0 {
		return body
	}

	lastIdx := len(arr) - 1
	existingCC := arr[lastIdx].Get("cache_control")
	if existingCC.Exists() && existingCC.Get("ttl").String() != "" {
		return body
	}

	path := fmt.Sprintf("tools.%d.cache_control", lastIdx)
	if existingCC.Exists() {
		if next, err := sjson.SetBytes(body, path+".ttl", claudeMimicDefaultCacheControlTTL); err == nil {
			body = next
		}
		return body
	}

	raw := fmt.Sprintf(`{"type":"ephemeral","ttl":%q}`, claudeMimicDefaultCacheControlTTL)
	if next, err := sjson.SetRawBytes(body, path, []byte(raw)); err == nil {
		body = next
	}
	return body
}

func restoreToolNamesInBytes(data []byte, rw *ToolNameRewrite) []byte {
	if rw != nil {
		for _, pair := range rw.ReverseOrdered {
			fake, real := pair[0], pair[1]
			if fake == "" || fake == real {
				continue
			}
			data = replaceAllBytes(data, fake, real)
		}
	}
	for prefix, replacement := range staticToolNameRewrites {
		data = replaceAllBytes(data, replacement, prefix)
	}
	return data
}

func replaceAllBytes(data []byte, from, to string) []byte {
	if len(data) == 0 || from == "" || from == to {
		return data
	}
	s := string(data)
	if !strings.Contains(s, from) {
		return data
	}
	return []byte(strings.ReplaceAll(s, from, to))
}

func toolNameRewriteFromContext(c interface {
	Value(string) (any, bool)
}) *ToolNameRewrite {
	if c == nil {
		return nil
	}
	raw, ok := c.Value(toolNameRewriteKey)
	if !ok || raw == nil {
		return nil
	}
	rw, _ := raw.(*ToolNameRewrite)
	return rw
}

func reverseToolNamesIfPresent(c interface {
	Value(string) (any, bool)
}, chunk []byte) []byte {
	if len(chunk) == 0 {
		return chunk
	}
	return restoreToolNamesInBytes(chunk, toolNameRewriteFromContext(c))
}
