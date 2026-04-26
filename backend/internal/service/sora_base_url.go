package service

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// NormalizeSoraAPIKeyBaseURL validates and normalizes the explicit upstream
// base_url used by Sora API key accounts.
func NormalizeSoraAPIKeyBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("base_url is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid url: %s", trimmed)
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("invalid url scheme: %s", parsed.Scheme)
	}
	if parsed.User != nil {
		return "", errors.New("base_url cannot contain userinfo")
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return "", errors.New("invalid host")
	}
	if strings.HasSuffix(host, ".localhost") {
		return "", errors.New("host is not allowed")
	}
	if _, blocked := soraBlockedHostnames[host]; blocked {
		return "", errors.New("host is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil && isSoraBlockedIP(ip) {
		return "", errors.New("host is not allowed")
	}

	if port := parsed.Port(); port != "" {
		num, err := strconv.Atoi(port)
		if err != nil || num <= 0 || num > 65535 {
			return "", fmt.Errorf("invalid port: %s", port)
		}
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("base_url cannot contain query or fragment")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}
