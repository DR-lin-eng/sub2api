package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildDynamicToolMap_BelowThreshold(t *testing.T) {
	names := []string{"bash", "edit", "read", "write", "search"}
	require.Nil(t, buildDynamicToolMap(names))
}

func TestBuildDynamicToolMap_AboveThresholdIsStable(t *testing.T) {
	names := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}
	a := buildDynamicToolMap(names)
	b := buildDynamicToolMap(names)

	require.NotNil(t, a)
	require.Equal(t, a, b)
	require.Len(t, a, len(names))
	for _, name := range names {
		require.Contains(t, a, name)
		require.NotEqual(t, name, a[name])
	}
}

func TestSanitizeToolName_StaticPrefix(t *testing.T) {
	require.Equal(t, "cc_sess_list", sanitizeToolName("sessions_list", nil))
	require.Equal(t, "cc_ses_get", sanitizeToolName("session_get", nil))
	require.Equal(t, "bash", sanitizeToolName("bash", nil))
}

func TestSanitizeToolName_DynamicTakesPrecedence(t *testing.T) {
	dynamic := map[string]string{"sessions_list": "analyze_ses00"}
	require.Equal(t, "analyze_ses00", sanitizeToolName("sessions_list", dynamic))
}

func TestRestoreToolNamesInBytes_LongestFirst(t *testing.T) {
	rw := &ToolNameRewrite{
		Forward: map[string]string{"foo": "abc_12", "bar": "abc_12_ext"},
		Reverse: map[string]string{"abc_12": "foo", "abc_12_ext": "bar"},
		ReverseOrdered: [][2]string{
			{"abc_12_ext", "bar"},
			{"abc_12", "foo"},
		},
	}

	data := []byte(`{"tool":"abc_12_ext","other":"abc_12"}`)
	require.Equal(t, `{"tool":"bar","other":"foo"}`, string(restoreToolNamesInBytes(data, rw)))
}

func TestRestoreToolNamesInBytes_StaticPrefixRollback(t *testing.T) {
	data := []byte(`{"name":"cc_sess_list","id":"cc_ses_xyz"}`)
	require.Equal(t, `{"name":"sessions_list","id":"session_xyz"}`, string(restoreToolNamesInBytes(data, nil)))
}

func TestApplyToolNameRewriteToBody_RenamesToolsAndToolChoice(t *testing.T) {
	body := []byte(`{"tools":[{"name":"sessions_list","input_schema":{}},{"name":"session_get","input_schema":{}},{"name":"web_search","type":"web_search_20250305"}],"tool_choice":{"type":"tool","name":"sessions_list"}}`)
	rw := buildToolNameRewriteFromBody(body)
	require.NotNil(t, rw)
	require.Contains(t, rw.Forward, "sessions_list")
	require.Contains(t, rw.Forward, "session_get")
	require.NotContains(t, rw.Forward, "web_search")

	out := applyToolNameRewriteToBody(body, rw)
	require.Equal(t, "cc_sess_list", gjson.GetBytes(out, "tools.0.name").String())
	require.Equal(t, "cc_ses_get", gjson.GetBytes(out, "tools.1.name").String())
	require.Equal(t, "web_search", gjson.GetBytes(out, "tools.2.name").String())
	require.Equal(t, "cc_sess_list", gjson.GetBytes(out, "tool_choice.name").String())
	require.Equal(t, "tool", gjson.GetBytes(out, "tool_choice.type").String())
}

func TestApplyToolsLastCacheBreakpoint_InjectsDefault(t *testing.T) {
	body := []byte(`{"tools":[{"name":"a","input_schema":{}},{"name":"b","input_schema":{}}]}`)
	out := applyToolsLastCacheBreakpoint(body)

	require.Equal(t, "ephemeral", gjson.GetBytes(out, "tools.1.cache_control.type").String())
	require.Equal(t, claudeMimicDefaultCacheControlTTL, gjson.GetBytes(out, "tools.1.cache_control.ttl").String())
	require.False(t, gjson.GetBytes(out, "tools.0.cache_control").Exists())
}

func TestApplyToolsLastCacheBreakpoint_PreservesClientTTL(t *testing.T) {
	body := []byte(`{"tools":[{"name":"a","input_schema":{},"cache_control":{"type":"ephemeral","ttl":"1h"}}]}`)
	out := applyToolsLastCacheBreakpoint(body)
	require.Equal(t, "1h", gjson.GetBytes(out, "tools.0.cache_control.ttl").String())
}

func TestStripMessageCacheControl(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"hi","cache_control":{"type":"ephemeral"}}]}]}`)
	out := stripMessageCacheControl(body)
	require.False(t, gjson.GetBytes(out, "messages.0.content.0.cache_control").Exists())
}

func TestAddMessageCacheBreakpoints_LastMessageOnly(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	out := addMessageCacheBreakpoints(body)
	require.Equal(t, "ephemeral", gjson.GetBytes(out, "messages.0.content.0.cache_control.type").String())
	require.Equal(t, claudeMimicDefaultCacheControlTTL, gjson.GetBytes(out, "messages.0.content.0.cache_control.ttl").String())
}

func TestAddMessageCacheBreakpoints_SecondToLastUserTurn(t *testing.T) {
	body := []byte(`{"messages":[
		{"role":"user","content":[{"type":"text","text":"q1"}]},
		{"role":"assistant","content":[{"type":"text","text":"a1"}]},
		{"role":"user","content":[{"type":"text","text":"q2"}]},
		{"role":"assistant","content":[{"type":"text","text":"a2"}]}
	]}`)
	out := addMessageCacheBreakpoints(body)

	require.Equal(t, "ephemeral", gjson.GetBytes(out, "messages.3.content.0.cache_control.type").String())
	require.Equal(t, "ephemeral", gjson.GetBytes(out, "messages.0.content.0.cache_control.type").String())
	require.False(t, gjson.GetBytes(out, "messages.1.content.0.cache_control").Exists())
	require.False(t, gjson.GetBytes(out, "messages.2.content.0.cache_control").Exists())
}

func TestAddMessageCacheBreakpoints_StringContentPromoted(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	out := addMessageCacheBreakpoints(body)

	require.True(t, gjson.GetBytes(out, "messages.0.content").IsArray())
	require.Equal(t, "text", gjson.GetBytes(out, "messages.0.content.0.type").String())
	require.Equal(t, "hi", gjson.GetBytes(out, "messages.0.content.0.text").String())
	require.Equal(t, claudeMimicDefaultCacheControlTTL, gjson.GetBytes(out, "messages.0.content.0.cache_control.ttl").String())
}

func TestBuildToolNameRewriteFromBody_ReverseOrderedByLengthDesc(t *testing.T) {
	body := []byte(`{"tools":[
		{"name":"t1","input_schema":{}},
		{"name":"t2","input_schema":{}},
		{"name":"t3","input_schema":{}},
		{"name":"t4","input_schema":{}},
		{"name":"t5","input_schema":{}},
		{"name":"t6","input_schema":{}}
	]}`)
	rw := buildToolNameRewriteFromBody(body)
	require.NotNil(t, rw)
	for i := 1; i < len(rw.ReverseOrdered); i++ {
		require.GreaterOrEqual(t, len(rw.ReverseOrdered[i-1][0]), len(rw.ReverseOrdered[i][0]))
	}
}

func TestBuildDynamicToolMap_FakeNameShape(t *testing.T) {
	names := []string{"alphabet", "bravo", "charlie", "delta", "echo", "foxtrot"}
	mapping := buildDynamicToolMap(names)
	require.NotNil(t, mapping)

	for _, name := range names {
		fake := mapping[name]
		require.Regexp(t, `^[a-z]+_[a-z0-9]{1,3}\d{2}$`, fake)
		head := name
		if len(head) > 3 {
			head = head[:3]
		}
		require.True(t, strings.Contains(fake, head), "fake %q should contain %q", fake, head)
	}
}

func TestNormalizeClaudeOAuthRequestBody_PreservesToolChoiceWithTools(t *testing.T) {
	body := []byte(`{"model":"claude-3-7-sonnet","messages":[],"tools":[{"name":"sessions_list","input_schema":{}}],"tool_choice":{"type":"tool","name":"sessions_list"}}`)
	out, _ := normalizeClaudeOAuthRequestBody(body, "claude-3-7-sonnet", claudeOAuthNormalizeOptions{})

	require.True(t, gjson.GetBytes(out, "tool_choice").Exists())
	require.Equal(t, "sessions_list", gjson.GetBytes(out, "tool_choice.name").String())
}

func TestNormalizeClaudeOAuthRequestBody_DropsToolChoiceWithoutTools(t *testing.T) {
	body := []byte(`{"model":"claude-3-7-sonnet","messages":[],"tool_choice":{"type":"auto"}}`)
	out, _ := normalizeClaudeOAuthRequestBody(body, "claude-3-7-sonnet", claudeOAuthNormalizeOptions{})
	require.False(t, gjson.GetBytes(out, "tool_choice").Exists())
	require.True(t, gjson.GetBytes(out, "tools").IsArray())
}

func TestApplyClaudeCodeOAuthMimicryToParsedRequestBody_FullPipeline(t *testing.T) {
	body := []byte(`{
		"model":"claude-3-7-sonnet-20250219",
		"system":"You are OpenCode, the best coding agent on the planet.",
		"messages":[
			{"role":"user","content":[{"type":"text","text":"q1","cache_control":{"type":"ephemeral"}}]},
			{"role":"assistant","content":[{"type":"text","text":"a1"}]},
			{"role":"user","content":[{"type":"text","text":"q2"}]},
			{"role":"assistant","content":[{"type":"text","text":"a2"}]}
		],
		"tools":[{"name":"sessions_list","input_schema":{}}],
		"tool_choice":{"type":"tool","name":"sessions_list"}
	}`)
	parsed := &ParsedRequest{
		Body:   body,
		Model:  "claude-3-7-sonnet-20250219",
		System: "You are OpenCode, the best coding agent on the planet.",
	}
	account := &Account{
		ID:       42,
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"claude_user_id": strings.Repeat("a", 64),
			"account_uuid":   "acct-uuid",
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	gwctx := gatewayctx.NewNative(req, httptest.NewRecorder(), nil, "")

	out, model := (&GatewayService{}).applyClaudeCodeOAuthMimicryToParsedRequestBody(context.Background(), gwctx, account, parsed, body, parsed.Model)

	require.Equal(t, parsed.Model, model)
	require.NotEmpty(t, gjson.GetBytes(out, "metadata.user_id").String())
	require.Equal(t, "cc_sess_list", gjson.GetBytes(out, "tools.0.name").String())
	require.Equal(t, "cc_sess_list", gjson.GetBytes(out, "tool_choice.name").String())
	require.Equal(t, claudeMimicDefaultCacheControlTTL, gjson.GetBytes(out, "tools.0.cache_control.ttl").String())
	require.Equal(t, "ephemeral", gjson.GetBytes(out, "messages.3.content.0.cache_control.type").String())
	require.Equal(t, "ephemeral", gjson.GetBytes(out, "messages.0.content.0.cache_control.type").String())

	rawRewrite, ok := gwctx.Value(toolNameRewriteKey)
	require.True(t, ok)
	require.Equal(t, `"sessions_list"`, string(restoreToolNamesInBytes([]byte(`"cc_sess_list"`), rawRewrite.(*ToolNameRewrite))))
}
