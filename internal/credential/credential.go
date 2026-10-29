// Package credential classifies AI-provider secrets by how they must be
// presented to an API: a first-party API key, a short-lived OAuth token, or a
// local session with no secret at all. Adapters use the classification to pick
// the correct auth scheme (e.g. "x-api-key" vs "Authorization: Bearer").
package credential

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// Kind is the authentication category of a stored provider secret.
type Kind string

const (
	// Unknown means no explicit classification is recorded; callers should
	// fall back to Detect.
	Unknown Kind = ""
	// APIKey is a long-lived first-party key (sk-ant-api…, sk-…, AIza…, gho_…).
	APIKey Kind = "api_key"
	// OAuth is a short-lived bearer token minted by an interactive login
	// (sk-ant-oat…, ya29.…, a bare JWT).
	OAuth Kind = "oauth"
	// Session is a local tool with usage telemetry on disk and no secret
	// (Cursor, Warp, Antigravity, a local Ollama daemon).
	Session Kind = "session"
)

// Detect infers a Kind from the shape of a raw secret. It is a best-effort
// fallback for configs written before credential types were tagged; an explicit
// stored type always wins.
func Detect(rawKey string) Kind {
	k := strings.TrimSpace(rawKey)
	switch {
	case k == "", k == "ollama-local":
		return Session
	case strings.HasPrefix(k, "sk-ant-oat"): // Anthropic OAuth access token
		return OAuth
	case strings.HasPrefix(k, "ya29."): // Google OAuth access token
		return OAuth
	case strings.HasPrefix(k, "sk-ant-api"): // Anthropic API key
		return APIKey
	case strings.HasPrefix(k, "AIza"): // Google API key
		return APIKey
	case strings.HasPrefix(k, "sk-"): // OpenAI-style API key
		return APIKey
	case strings.HasPrefix(k, "gho_"), strings.HasPrefix(k, "ghp_"),
		strings.HasPrefix(k, "ghu_"), strings.HasPrefix(k, "github_pat_"):
		return APIKey // GitHub tokens authenticate like an API key
	case looksLikeJWT(k): // bare JWT with no api-key prefix → OIDC/OAuth token
		return OAuth
	default:
		return APIKey
	}
}

// Resolve returns stored if it is set, otherwise the Kind inferred from rawKey.
func Resolve(stored string, rawKey string) Kind {
	if k := Kind(strings.TrimSpace(stored)); k != Unknown {
		return k
	}
	return Detect(rawKey)
}

// looksLikeJWT reports whether s is a three-segment token whose first segment
// base64url-decodes to a JSON object containing an "alg" field.
func looksLikeJWT(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
		return false
	}
	hdr, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal(hdr, &m); err != nil {
		return false
	}
	_, ok := m["alg"]
	return ok
}

// JWTExpiry returns the expiry time encoded in a JWT's "exp" claim. ok is false
// when s is not a JWT or carries no numeric exp claim.
func JWTExpiry(s string) (expiry time.Time, ok bool) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}

// Expired reports whether s is a JWT whose exp claim is in the past. A token
// that is not a JWT, or has no exp, is treated as not expired (false).
func Expired(s string) bool {
	exp, ok := JWTExpiry(s)
	if !ok {
		return false
	}
	return time.Now().After(exp)
}
