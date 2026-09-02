package ui

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"ai-usage/internal/config"
)

type KeyItem struct {
	Index       int
	ProviderID  string
	DisplayName string
	ModelTier   string
	Endpoint    string
	MaskedKey   string
	PlainKey    string
	IsRevealed  bool
	IsLocalAuth bool
}

// RunKeysManager provides an interactive CLI interface to view and toggle obfuscated/revealed keys.
func RunKeysManager(in io.Reader, out io.Writer, cfg *config.Config) {
	if cfg == nil || len(cfg.Providers) == 0 {
		fmt.Fprintln(out, "⚠️  No AI providers configured yet. Run 'usage scan new' or 'usage add-provider' to add credentials.")
		return
	}

	// Build sorted list of provider keys
	var items []*KeyItem
	idx := 1

	// Sort keys alphabetically for consistent numbering
	var providerIDs []string
	for id := range cfg.Providers {
		providerIDs = append(providerIDs, id)
	}
	sort.Strings(providerIDs)

	for _, id := range providerIDs {
		p := cfg.Providers[id]
		name := p.DisplayName
		if name == "" {
			name = id
		}

		plainKey := p.GetDecryptedKey()

		isLocal := false
		masked := "None / Unset"
		if plainKey == "" {
			if id == "antigravity" || id == "cursor" || id == "warp" || id == "ollama" {
				isLocal = true
				masked = "Local Session / Daemon Telemetry ✔"
			}
		} else {
			masked = MaskSecret(plainKey)
		}

		items = append(items, &KeyItem{
			Index:       idx,
			ProviderID:  id,
			DisplayName: name,
			ModelTier:   p.ModelTier,
			Endpoint:    p.Endpoint,
			MaskedKey:   masked,
			PlainKey:    plainKey,
			IsRevealed:  false,
			IsLocalAuth: isLocal,
		})
		idx++
	}

	scanner := bufio.NewScanner(in)

	for {
		renderKeysList(out, items)
		fmt.Fprintf(out, "👉 Enter key number [1-%d] to reveal/conceal, [a] to reveal all, [h] to hide all, or [0] to exit: ", len(items))

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "0" || input == "q" || input == "exit" || input == "quit" {
			fmt.Fprintln(out, "\n✔ Secure credentials session closed.")
			break
		}

		if strings.EqualFold(input, "a") || strings.EqualFold(input, "all") {
			for _, item := range items {
				if item.PlainKey != "" {
					item.IsRevealed = true
				}
			}
			continue
		}

		if strings.EqualFold(input, "h") || strings.EqualFold(input, "hide") {
			for _, item := range items {
				item.IsRevealed = false
			}
			continue
		}

		num, err := strconv.Atoi(input)
		if err == nil && num >= 1 && num <= len(items) {
			item := items[num-1]
			if item.PlainKey != "" {
				item.IsRevealed = !item.IsRevealed
			}
		} else {
			fmt.Fprintf(out, "⚠️  Invalid choice '%s'. Please enter a number between 1 and %d (or 0 to exit).\n\n", input, len(items))
		}
	}
}

func renderKeysList(w io.Writer, items []*KeyItem) {
	boxWidth := 96
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(FormatBoxTop(boxWidth))
	sb.WriteString(FormatBoxLine("🔐 SECURE AI CREDENTIALS & API KEY MANAGER", boxWidth))
	sb.WriteString(FormatBoxLine("🛡️  Keys stored locally with AES-256-GCM encryption", boxWidth))
	sb.WriteString(FormatBoxDivider(boxWidth))
	sb.WriteString(FormatBoxLine("• Press [1-N] to reveal or conceal a private key", boxWidth))
	sb.WriteString(FormatBoxLine("• Press [a] to reveal all, [h] to conceal all, or [0] to exit", boxWidth))
	sb.WriteString(FormatBoxBottom(boxWidth))
	sb.WriteString("\n")

	for _, item := range items {
		sb.WriteString(fmt.Sprintf("  \033[1;36m[%d]\033[0m \033[1m%s\033[0m (%s)\n", item.Index, item.DisplayName, item.ProviderID))
		if item.ModelTier != "" {
			sb.WriteString(fmt.Sprintf("      📍 Model / Tier  : %s\n", item.ModelTier))
		}
		if item.Endpoint != "" {
			sb.WriteString(fmt.Sprintf("      🌐 Endpoint      : %s\n", item.Endpoint))
		}

		if item.IsLocalAuth {
			sb.WriteString(fmt.Sprintf("      🔑 Auth Type     : \033[32m%s\033[0m\n", item.MaskedKey))
		} else if item.PlainKey == "" {
			sb.WriteString("      🔑 API Key       : \033[33m(Unset / None configured)\033[0m\n")
		} else if item.IsRevealed {
			sb.WriteString(fmt.Sprintf("      🔑 API Key       : \033[1;32m🔓 %s\033[0m \033[32m(REVEALED)\033[0m\n", item.PlainKey))
		} else {
			sb.WriteString(fmt.Sprintf("      🔑 API Key       : \033[38;5;244m🔒 %s\033[0m \033[2m(Hidden — press %d to reveal)\033[0m\n", item.MaskedKey, item.Index))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("────────────────────────────────────────────────────────────────────────────────────────────────\n")
	_, _ = w.Write([]byte(sb.String()))
}

// MaskSecret returns an obfuscated representation of a secret string.
func MaskSecret(key string) string {
	if len(key) == 0 {
		return ""
	}
	if len(key) <= 8 {
		return "••••••••"
	}
	if len(key) <= 16 {
		return key[:3] + "..." + key[len(key)-3:]
	}
	return key[:6] + "..." + key[len(key)-4:]
}
