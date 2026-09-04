package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ai-usage/internal/credential"
)

// ProviderConfig holds credentials, cycle reset dates, and metadata for a provider.
type ProviderConfig struct {
	ID                 string    `json:"id"`
	DisplayName        string    `json:"display_name,omitempty"`
	APIKey             string    `json:"api_key,omitempty"`         // Stored encrypted on disk
	CredentialType     string    `json:"credential_type,omitempty"` // "api_key" | "oauth" | "session"
	CustomQuota        float64   `json:"custom_quota"`
	ModelTier          string    `json:"model_tier,omitempty"`
	Endpoint           string    `json:"endpoint,omitempty"`
	Enabled            bool      `json:"enabled"`
	AnchorBillingDay   int       `json:"anchor_billing_day,omitempty"` // 1-28 per-provider cycle reset day
	CreatedAt          time.Time `json:"created_at"`
	LastActivity       time.Time `json:"last_activity"`
	StaleDaysThreshold int       `json:"stale_days_threshold,omitempty"` // Default 30 days
}

// GetDecryptedKey returns the decrypted plaintext API key.
func (p ProviderConfig) GetDecryptedKey() string {
	if p.APIKey == "" {
		return ""
	}
	dec, err := DecryptString(p.APIKey)
	if err != nil {
		return p.APIKey
	}
	return dec
}

// ResolvedCredentialType returns the recorded credential type, inferring it from
// the key's shape when the config predates credential-type tagging.
func (p ProviderConfig) ResolvedCredentialType() string {
	return string(credential.Resolve(p.CredentialType, p.GetDecryptedKey()))
}

// SetPlainKey encrypts and sets the API key.
func (p *ProviderConfig) SetPlainKey(plain string) error {
	if plain == "" {
		p.APIKey = ""
		return nil
	}
	enc, err := EncryptString(plain)
	if err != nil {
		return err
	}
	p.APIKey = enc
	p.LastActivity = time.Now().UTC()
	return nil
}

// IsStale checks if the provider has had no activity for the stale threshold (default 30 days).
func (p ProviderConfig) IsStale() (bool, int) {
	threshold := p.StaleDaysThreshold
	if threshold <= 0 {
		threshold = 30
	}

	refTime := p.LastActivity
	if refTime.IsZero() {
		refTime = p.CreatedAt
	}
	if refTime.IsZero() {
		return false, 0
	}

	daysInactive := int(time.Since(refTime).Hours() / 24)
	return daysInactive >= threshold, daysInactive
}

// Config represents the application configuration.
type Config struct {
	AnchorBillingDay int                       `json:"anchor_billing_day"` // Default global cycle day
	TimeoutSeconds   int                       `json:"timeout_seconds"`    // Per-provider timeout
	StaleWarningDays int                       `json:"stale_warning_days"` // Global inactivity threshold (default: 30)
	MongoURI         string                    `json:"mongo_uri,omitempty"`
	MongoDBName      string                    `json:"mongo_db_name,omitempty"`
	Providers        map[string]ProviderConfig `json:"providers"`

	mu sync.RWMutex `json:"-"`
}

// ConfigDir returns the platform-appropriate configuration directory.
func ConfigDir() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		home, hErr := os.UserHomeDir()
		if hErr != nil {
			return "", fmt.Errorf("unable to resolve config directory: %w", err)
		}
		baseDir = filepath.Join(home, ".config")
	}

	appDir := filepath.Join(baseDir, "ai-usage")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", appDir, err)
	}
	return appDir, nil
}

// ConfigFilePath returns the absolute path to config.json.
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// DefaultConfig provides default settings and built-in provider stubs.
func DefaultConfig() *Config {
	now := time.Now().UTC()
	return &Config{
		AnchorBillingDay: 1,
		TimeoutSeconds:   6,
		StaleWarningDays: 30,
		Providers: map[string]ProviderConfig{
			"antigravity": {
				ID:               "antigravity",
				DisplayName:      "Antigravity Agent",
				CustomQuota:      10_000_000,
				ModelTier:        "Agent Pro Runtime",
				AnchorBillingDay: 1,
				Enabled:          true,
				CreatedAt:        now,
				LastActivity:     now,
			},
			"claude": {
				ID:               "claude",
				DisplayName:      "Anthropic Claude",
				CustomQuota:      5_000_000,
				ModelTier:        "Claude 3.5 Sonnet / Opus",
				AnchorBillingDay: 1,
				Enabled:          true,
				CreatedAt:        now,
				LastActivity:     now,
			},
			"codex": {
				ID:               "codex",
				DisplayName:      "OpenAI Codex",
				CustomQuota:      4_000_000,
				ModelTier:        "GPT-4o / Codex",
				AnchorBillingDay: 1,
				Enabled:          true,
				CreatedAt:        now,
				LastActivity:     now,
			},
			"gemini": {
				ID:               "gemini",
				DisplayName:      "Google Gemini",
				CustomQuota:      4_000_000,
				ModelTier:        "Gemini 1.5 Pro / Flash",
				AnchorBillingDay: 1,
				Enabled:          true,
				CreatedAt:        now,
				LastActivity:     now,
			},
			"copilot": {
				ID:               "copilot",
				DisplayName:      "GitHub Copilot",
				CustomQuota:      3000,
				ModelTier:        "Individual Plan",
				AnchorBillingDay: 1,
				Enabled:          true,
				CreatedAt:        now,
				LastActivity:     now,
			},
		},
	}
}

// Load loads configuration from disk and ensures keys are encrypted.
func Load() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg := DefaultConfig()
		_ = Save(cfg)
		return cfg, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config json: %w", err)
	}

	if cfg.AnchorBillingDay < 1 || cfg.AnchorBillingDay > 28 {
		cfg.AnchorBillingDay = 1
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 6
	}
	if cfg.StaleWarningDays <= 0 {
		cfg.StaleWarningDays = 30
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}

	return &cfg, nil
}

// Save writes configuration to disk with encrypted credentials.
func Save(cfg *Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// SetProvider sets or updates a provider configuration, ensuring the key is encrypted.
func (c *Config) SetProvider(p ProviderConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}

	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	p.LastActivity = time.Now().UTC()

	// Classify the credential from its plaintext shape before it is encrypted,
	// unless the caller already supplied an explicit type.
	if p.CredentialType == "" {
		p.CredentialType = string(credential.Detect(p.APIKey))
	}

	// Ensure API key is encrypted
	if p.APIKey != "" {
		enc, err := EncryptString(p.APIKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt key for provider %s: %w", p.ID, err)
		}
		p.APIKey = enc
	}

	if p.AnchorBillingDay < 1 || p.AnchorBillingDay > 28 {
		if c.AnchorBillingDay > 0 {
			p.AnchorBillingDay = c.AnchorBillingDay
		} else {
			p.AnchorBillingDay = 1
		}
	}

	c.Providers[p.ID] = p
	return nil
}

// SetProviderCycle sets the reset day for a specific provider.
func (c *Config) SetProviderCycle(id string, day int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if day < 1 || day > 28 {
		return fmt.Errorf("billing anchor day must be between 1 and 28, got %d", day)
	}

	p, exists := c.Providers[id]
	if !exists {
		return fmt.Errorf("provider '%s' not found", id)
	}

	p.AnchorBillingDay = day
	c.Providers[id] = p
	return nil
}

// UpdateKey replaces the API key for an existing provider and resets activity time.
func (c *Config) UpdateKey(id, newPlainKey string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	p, exists := c.Providers[id]
	if !exists {
		return fmt.Errorf("provider '%s' not found", id)
	}

	if err := p.SetPlainKey(newPlainKey); err != nil {
		return err
	}
	p.CredentialType = string(credential.Detect(newPlainKey))
	p.Enabled = true
	c.Providers[id] = p
	return nil
}

// TouchActivity marks active usage timestamp for a provider.
func (c *Config) TouchActivity(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if p, exists := c.Providers[id]; exists {
		p.LastActivity = time.Now().UTC()
		c.Providers[id] = p
	}
}

// RemoveProvider deletes a provider by ID.
func (c *Config) RemoveProvider(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.Providers[id]; exists {
		delete(c.Providers, id)
		return true
	}
	return false
}

// PruneStaleProviders removes providers that have exceeded the inactivity threshold.
func (c *Config) PruneStaleProviders(thresholdDays int) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if thresholdDays <= 0 {
		thresholdDays = c.StaleWarningDays
	}
	if thresholdDays <= 0 {
		thresholdDays = 30
	}

	var pruned []string
	for id, p := range c.Providers {
		if id == "antigravity" {
			continue
		}
		refTime := p.LastActivity
		if refTime.IsZero() {
			refTime = p.CreatedAt
		}
		if !refTime.IsZero() && int(time.Since(refTime).Hours()/24) >= thresholdDays {
			delete(c.Providers, id)
			pruned = append(pruned, id)
		}
	}
	return pruned
}
