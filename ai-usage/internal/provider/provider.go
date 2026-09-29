package provider

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"
)

// NewHTTPClient creates a resilient HTTP client with appropriate timeouts.
func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 6 * time.Second
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			TLSClientConfig:    &tls.Config{MinVersion: tls.VersionTLS12},
			DisableCompression: false,
		},
	}
}

// IsAPIKeyConfigured returns true if the key is non-empty and not a placeholder.
func IsAPIKeyConfigured(key string) bool {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return false
	}
	// Check for common placeholders
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "placeholder") || strings.Contains(lower, "your_api_key") || strings.Contains(lower, "sk-xxx") {
		return false
	}
	return true
}
