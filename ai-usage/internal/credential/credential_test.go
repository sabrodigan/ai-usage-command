package credential

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"
)

func jwt(payload string) string {
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return hdr + "." + body + ".c2ln"
}

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want Kind
	}{
		{"empty", "", Session},
		{"ollama sentinel", "ollama-local", Session},
		{"anthropic oauth", "sk-ant-oat01-abc123", OAuth},
		{"anthropic api key", "sk-ant-api03-abc123", APIKey},
		{"google oauth", "ya29.a0AdMbc123", OAuth},
		{"google api key", "AIzaSyABC123", APIKey},
		{"openai api key", "sk-proj-abc123", APIKey},
		{"github oauth token", "gho_abc123", APIKey},
		{"github pat", "github_pat_abc123", APIKey},
		{"bare jwt", jwt(`{"sub":"user"}`), OAuth},
		{"opaque string", "random-opaque-token", APIKey},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Detect(c.key); got != c.want {
				t.Errorf("Detect(%q) = %q, want %q", c.key, got, c.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	if got := Resolve("oauth", "sk-ant-api03-x"); got != OAuth {
		t.Errorf("stored type should win: got %q", got)
	}
	if got := Resolve("", "sk-ant-oat01-x"); got != OAuth {
		t.Errorf("fallback to Detect: got %q", got)
	}
	if got := Resolve("  ", "AIzaX"); got != APIKey {
		t.Errorf("blank stored type falls back: got %q", got)
	}
}

func TestJWTExpiryAndExpired(t *testing.T) {
	future := time.Now().Add(time.Hour).Unix()
	past := time.Now().Add(-time.Hour).Unix()

	tok := jwt(fmt.Sprintf(`{"exp":%d}`, future))
	if exp, ok := JWTExpiry(tok); !ok || time.Until(exp) <= 0 {
		t.Errorf("expected future expiry, ok=%v exp=%v", ok, exp)
	}
	if Expired(tok) {
		t.Errorf("token with future exp should not be expired")
	}

	stale := jwt(fmt.Sprintf(`{"exp":%d}`, past))
	if !Expired(stale) {
		t.Errorf("token with past exp should be expired")
	}

	if _, ok := JWTExpiry("not-a-jwt"); ok {
		t.Errorf("non-JWT should report ok=false")
	}
	if Expired("sk-ant-api03-x") {
		t.Errorf("non-JWT api key must not be treated as expired")
	}
}
