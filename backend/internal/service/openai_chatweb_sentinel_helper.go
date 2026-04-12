package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type openAIChatWebSentinelPersona struct {
	Platform              string  `json:"platform,omitempty"`
	Vendor                string  `json:"vendor,omitempty"`
	TimezoneOffsetMin     int     `json:"timezone_offset_min,omitempty"`
	SessionID             string  `json:"session_id,omitempty"`
	TimeOrigin            float64 `json:"time_origin,omitempty"`
	WindowFlags           []int   `json:"window_flags,omitempty"`
	WindowFlagsSet        bool    `json:"window_flags_set,omitempty"`
	EntropyA              float64 `json:"entropy_a,omitempty"`
	EntropyB              float64 `json:"entropy_b,omitempty"`
	DateString            string  `json:"date_string,omitempty"`
	RequirementsScriptURL string  `json:"requirements_script_url,omitempty"`
	NavigatorProbe        string  `json:"navigator_probe,omitempty"`
	DocumentProbe         string  `json:"document_probe,omitempty"`
	WindowProbe           string  `json:"window_probe,omitempty"`
	PerformanceNow        float64 `json:"performance_now,omitempty"`
	RequirementsElapsed   float64 `json:"requirements_elapsed,omitempty"`
}

type openAIChatWebSentinelSession struct {
	DeviceID            string                       `json:"device_id"`
	UserAgent           string                       `json:"user_agent"`
	ScreenWidth         int                          `json:"screen_width"`
	ScreenHeight        int                          `json:"screen_height"`
	HeapLimit           int64                        `json:"heap_limit"`
	HardwareConcurrency int                          `json:"hardware_concurrency"`
	Language            string                       `json:"language"`
	LanguagesJoin       string                       `json:"languages_join"`
	Persona             openAIChatWebSentinelPersona `json:"persona"`
}

type openAIChatWebSentinelHelper struct {
	pythonPath string
	helperPath string
}

type openAIChatWebSentinelHelperRunner interface {
	requirementsToken(ctx context.Context, session openAIChatWebSentinelSession) (string, error)
	enforcementToken(ctx context.Context, session openAIChatWebSentinelSession, required bool, seed string, difficulty string) (string, error)
	solveTurnstile(ctx context.Context, session openAIChatWebSentinelSession, requirementsToken string, dx string) (string, error)
	solveSessionObserver(ctx context.Context, session openAIChatWebSentinelSession, proofToken string, collectorDX string) (string, error)
}

var newOpenAIChatWebSentinelHelperRunner = func() openAIChatWebSentinelHelperRunner {
	return newOpenAIChatWebSentinelHelper()
}

func newOpenAIChatWebSentinelHelper() *openAIChatWebSentinelHelper {
	return &openAIChatWebSentinelHelper{
		pythonPath: resolveOpenAIChatWebPythonPath(),
		helperPath: resolveOpenAIChatWebHelperPath(),
	}
}

func resolveOpenAIChatWebPythonPath() string {
	if value := strings.TrimSpace(os.Getenv("SUB2API_OPENAI_CHATWEB_PYTHON")); value != "" {
		return value
	}
	return "python3"
}

func resolveOpenAIChatWebHelperPath() string {
	if value := strings.TrimSpace(os.Getenv("SUB2API_OPENAI_CHATWEB_HELPER")); value != "" {
		return value
	}
	candidates := []string{
		"/app/resources/openai_chatweb/helper.py",
		filepath.Join("backend", "resources", "openai_chatweb", "helper.py"),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func (h *openAIChatWebSentinelHelper) run(ctx context.Context, payload map[string]any, out any) error {
	if h == nil {
		return fmt.Errorf("chatweb sentinel helper is nil")
	}
	if strings.TrimSpace(h.helperPath) == "" {
		return fmt.Errorf("chatweb sentinel helper path is empty")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sentinel payload: %w", err)
	}
	if _, err := os.Stat(h.helperPath); err != nil {
		return fmt.Errorf("chatweb sentinel helper not found: %w", err)
	}

	timeoutCtx := ctx
	cancel := func() {}
	if _, ok := ctx.Deadline(); !ok {
		timeoutCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
	}
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, h.pythonPath, h.helperPath)
	cmd.Stdin = bytes.NewReader(raw)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run chatweb sentinel helper: %w stderr=%s", err, strings.TrimSpace(stderr.String()))
	}
	if err := json.Unmarshal(stdout.Bytes(), out); err != nil {
		return fmt.Errorf("decode chatweb sentinel helper output: %w", err)
	}
	return nil
}

func (h *openAIChatWebSentinelHelper) requirementsToken(ctx context.Context, session openAIChatWebSentinelSession) (string, error) {
	var result struct {
		RequirementsToken string `json:"requirements_token"`
	}
	if err := h.run(ctx, map[string]any{
		"op":      "requirements_token",
		"session": session,
	}, &result); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.RequirementsToken), nil
}

func (h *openAIChatWebSentinelHelper) enforcementToken(ctx context.Context, session openAIChatWebSentinelSession, required bool, seed string, difficulty string) (string, error) {
	var result struct {
		ProofToken string `json:"proof_token"`
	}
	if err := h.run(ctx, map[string]any{
		"op":         "enforcement_token",
		"session":    session,
		"required":   required,
		"seed":       seed,
		"difficulty": difficulty,
	}, &result); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.ProofToken), nil
}

func (h *openAIChatWebSentinelHelper) solveTurnstile(ctx context.Context, session openAIChatWebSentinelSession, requirementsToken string, dx string) (string, error) {
	var result struct {
		TurnstileToken string `json:"turnstile_token"`
	}
	if err := h.run(ctx, map[string]any{
		"op":                 "solve_turnstile_dx",
		"session":            session,
		"requirements_token": requirementsToken,
		"dx":                 dx,
	}, &result); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.TurnstileToken), nil
}

func (h *openAIChatWebSentinelHelper) solveSessionObserver(ctx context.Context, session openAIChatWebSentinelSession, proofToken string, collectorDX string) (string, error) {
	var result struct {
		SOToken string `json:"so_token"`
	}
	if err := h.run(ctx, map[string]any{
		"op":           "solve_session_observer",
		"session":      session,
		"proof_token":  proofToken,
		"collector_dx": collectorDX,
	}, &result); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.SOToken), nil
}
