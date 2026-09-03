# `usage` — Multi-Provider AI Model Consumption Tracker

A high-performance, lightweight, cross-platform CLI tool written in Go to monitor, aggregate, and rank developer AI model and token consumption across Anthropic (Claude), OpenAI (Codex / GPT-4o), Google (Gemini), GitHub Copilot, Antigravity Agent, and custom endpoints (DeepSeek, Ollama, Groq, vLLM, OpenRouter).

---

## ⚡ Features

- **Ranked Consumption Overview**: Automatically computes percentage used, remaining quotas, and sorts models from most used to least used.
- **Visual Progress Bars**: Cross-platform Unicode progress bars (`[■■■■········]`) compatible with Windows Terminal, PowerShell 7, Command Prompt, and Linux shells (`xterm`, `kitty`, `ghostty`, `foot`).
- **Flexible Billing Cycles**: Supports standard calendar months (1st to end-of-month) or custom anchor reset days (e.g., 15th to 14th).
- **Non-blocking & Concurrent**: Queries all provider adapters simultaneously with timeout guarantees; unreachable providers degrade gracefully without blocking execution.
- **Zero Heavy Dependencies**: Starts up in under **5ms** and uses negligible memory (< 10MB).
- **Export & History**: Export snapshots in structured **JSON** or **CSV** formats, with local historical caching.
- **Cross-Platform**: Native binaries for Linux (amd64 / arm64) and Windows 11 (amd64).

---

## 📦 Project Structure

```
ai-usage/
├── cmd/
│   └── usage/
│       └── main.go                 # CLI commands & flags routing
├── internal/
│   ├── config/
│   │   ├── config.go               # Config loader (~/.config/ai-usage or %APPDATA%\ai-usage)
│   │   └── config_test.go          # Config unit tests
│   ├── core/
│   │   ├── model.go                # Domain models & formatting
│   │   ├── billing.go              # Billing cycle date window calculation
│   │   ├── billing_test.go         # Billing window test suite
│   │   ├── aggregator.go           # Goroutine worker pool & ranking algorithm
│   │   └── aggregator_test.go      # Aggregator test suite
│   ├── provider/
│   │   ├── provider.go             # ProviderAdapter interface & HTTP client
│   │   ├── antigravity.go          # Local Antigravity Agent telemetry adapter
│   │   ├── claude.go               # Anthropic Claude adapter
│   │   ├── codex.go                # OpenAI / Codex adapter
│   │   ├── gemini.go               # Google Gemini adapter
│   │   ├── copilot.go              # GitHub Copilot adapter
│   │   ├── custom.go               # Generic OpenAI-compatible endpoint adapter
│   │   └── provider_test.go        # Provider adapter test suite
│   ├── storage/
│   │   ├── storage.go              # FileStore & snapshot document definitions
│   │   └── storage_test.go         # Storage test suite
│   └── ui/
│       ├── table.go                # Terminal table & progress bar renderer
│       ├── json.go                 # JSON / CSV export renderers
│       └── ui_test.go              # UI formatters test suite
├── Makefile                        # Multi-platform build & test recipes
└── go.mod
```

---

## 🚀 Quick Start

### 1. Build & Install

```bash
# Build binary for current platform
make build

# Or cross-compile for Linux + Windows 11
make build-all

# Or install to $GOPATH/bin
make install
```

### 2. View Current Usage

```bash
./usage
# or
./usage now
```

Example output:
```text
┌────────────────────────────────────────────────────────────────────────┐
│  🚀 AI MODEL USAGE MONITOR                                             │
├────────────────────────────────────────────────────────────────────────┤
│  📅 Current Time   : 2026-09-01 17:19:15 BST                           │
│  💳 Billing Window : 2026-09-01 ➔ 2026-09-30                           │
│  📊 Total Providers: 5  (Active: 5 )                                  │
│  💰 Total Est Cost : $62.40                                            │
└────────────────────────────────────────────────────────────────────────┘

RANK   PROVIDER            MODEL / TIER               CONSUMED        QUOTA    REMAINING   USAGE %   PROGRESS BAR     COST (USD)   STATUS
----   --------            ------------               --------        -----    ---------   -------   ------------     ----------   ------
#1     OpenAI Codex        GPT-4o / Codex             2.15M tokens    4.00M    1.85M        53.8%    [■■■■■■······]   $ 21.50      ✔ OK
#2     GitHub Copilot      Individual Plan            1240 requests   3000     1760         41.3%    [■■■■■·······]   $ 10.00      ✔ OK
#3     Anthropic Claude    Claude 3.5 Sonnet / Opus   1.85M tokens    5.00M    3.15M        36.9%    [■■■■········]   $ 27.68      ✔ OK
#4     Google Gemini       Gemini 1.5 Pro / Flash     920.0K tokens   4.00M    3.08M        23.0%    [■■■·········]   $  3.22      ✔ OK
#5     Antigravity Agent   Agent Pro Runtime          130.1K tokens   10.00M   9.87M         1.3%    [············]   $  0.00      ✔ OK
```

---

## 🛠️ CLI Commands & Reference

| Command | Description |
|---|---|
| `usage` / `usage live` | Display real-time live usage table queried from active logs, daemons & APIs |
| `usage example` / `usage demo` | Display representative simulation / benchmark demo data |
| `usage keys` / `usage key` | Interactive secure key manager (masked by default, press [1-N] to reveal, [0] to exit) |
| `usage watch` / `usage tui` | Open interactive live auto-refreshing dashboard (auto-syncs machine tools on start) |
| `usage scan` | Scan machine for local AI files/env credentials vs configured inventory |
| `usage scan new` | Diff scan against known providers & interactively review/add with cycle & quota |
| `usage scan new --yes` | Automatically import all newly detected credentials |
| `usage recommend` | AI-powered cost optimization insights & recommendations |
| `usage stats` | Historical daily consumption time-series trends from local SQLite DB |
| `usage now --json` / `--csv` | Output snapshot in structured JSON or CSV format |
| `usage providers` | List all configured providers, reset days, and AES-256 encryption status |
| `usage add-provider` | Add or configure a provider API key, quota & cycle |
| `usage update-key <id> <key>` | Replace/update an existing provider's API key |
| `usage set-cycle <id> <1-28>` | Set billing reset day for a specific provider |
| `usage set-cycle <1-28>` | Set default global billing reset day |
| `usage prune-stale` | Clean up providers with no usage for 30+ days |
| `usage remove-provider <id>` | Remove a configured provider |
| `usage history` | Display recent snapshot history |
| `usage version` | Print version information |

### 🖥️ Interactive Live Dashboard (`usage watch` / `usage tui`)

Monitor real-time token and request consumption across all providers in an auto-refreshing TUI:

- **Keyboard Shortcuts**:
  - `q` or `Ctrl+C`: Quit dashboard and restore terminal.
  - `r`: Force immediate telemetry refresh.
  - `↑` / `k` / `↓` / `j`: Select and inspect individual providers.
  - `s`: Cycle sort order (by Usage %, Consumed, Quota, Remaining, Cost, or Provider Name).
  - `/` or `f`: Live interactive provider filter.
  - `Space` / `Enter`: Toggle expanded inspection card.
  - `+` / `-`: Increase or decrease auto-refresh interval (1s–60s).
  - `h` / `?`: Toggle help overlay.

### 🔍 Smart Diff Scanner (`usage scan new`)

Automatically detects local AI credentials (Claude, OpenAI, Gemini, Copilot, Cursor, Warp, Continue, Ollama, OpenRouter, DeepSeek, Groq, Mistral, Perplexity, etc.) and compares against your known configuration:

```bash
# Interactively review newly discovered providers and configure custom quota & cycle dates
usage scan new

# Or import all newly detected providers non-interactively
usage scan new --yes
```

### Configuring Providers Manually

```bash
# Configure Anthropic Claude API Key and 10M token monthly quota
usage add-provider --id claude --key sk-ant-api03-... --quota 10000000 --tier "Claude 3.5 Sonnet"

# Configure DeepSeek custom endpoint
usage add-provider --id deepseek --name "DeepSeek AI" --endpoint "https://api.deepseek.com" --key sk-... --quota 8000000 --tier "DeepSeek-V3" --cycle 15

# Configure local Ollama endpoint
usage add-provider --id ollama --name "Ollama Local" --endpoint "http://localhost:11434" --quota 50000000 --tier "Llama-3.3-70B"
```

---

## ⚙️ Configuration File Location

- **Linux**: `~/.config/ai-usage/config.json`
- **Windows 11**: `%APPDATA%\ai-usage\config.json`

File permissions are automatically restricted (`0600`) to protect API keys and tokens.

---

## 🧪 Running Tests

```bash
make test
```
All unit tests run with Go's race detector enabled (`-race`).

---

## 👤 Author

**Stephen Brodigan**

