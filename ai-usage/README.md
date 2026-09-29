# `usage` — Multi-Provider AI Model Consumption & Quota Tracker

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20Windows%20%7C%20macOS-blue?style=flat)](#installation--build)
[![Architecture](https://img.shields.io/badge/Arch-amd64%20%7C%20arm64-orange?style=flat)](#installation--build)
[![Storage](https://img.shields.io/badge/Storage-Embedded%20SQLite%20(WAL)-4169E1?style=flat&logo=sqlite)](#storage--database-architecture)
[![Security](https://img.shields.io/badge/Security-AES--256--GCM%20Vault-green?style=flat)](#security--cryptography-architecture)
[![License](https://img.shields.io/badge/License-AGPL--3.0-purple?style=flat)](LICENSE.md)

**`usage`** is a high-performance, single-binary, cross-platform CLI tool and interactive Terminal User Interface (TUI) written in Go. It monitors, aggregates, forecasts, and ranks developer AI model and token consumption across **Anthropic (Claude)**, **OpenAI (Codex / GPT-4o)**, **Google (Gemini)**, **GitHub Copilot**, **Antigravity Agent**, **Meta Muse**, **Cursor IDE**, **Warp Terminal**, **Ollama**, **DeepSeek**, and any custom OpenAI-compatible endpoint (Groq, OpenRouter, Mistral, Perplexity, vLLM, LM Studio).

Starts up in under **5ms**, requires **zero runtime dependencies**, encrypts credentials with **machine-local AES-256-GCM**, and logs telemetry into an **embedded SQLite** database with WAL mode.

---

## 📑 Table of Contents

- [Why `usage`?](#why-usage)
- [⚡ Key Features](#key-features)
- [📦 Project Architecture](#project-architecture)
- [🚀 Quick Start Guide (60 Seconds)](#quick-start-guide-60-seconds)
- [📥 Installation & Build](#installation--build)
- [🤖 Supported Providers & Detection Matrix](#supported-providers--detection-matrix)
- [🛠️ Complete Command Reference](#complete-command-reference)
  - [`usage` / `usage live` / `usage now`](#usage--usage-live--usage-now)
  - [`usage local`](#usage-local)
  - [`usage example` / `usage demo`](#usage-example--usage-demo)
  - [`usage watch` / `usage tui` (Interactive Live Dashboard)](#usage-watch--usage-tui-interactive-live-dashboard)
  - [`usage scan` & `usage scan new` (Smart Diff Scanner)](#usage-scan--usage-scan-new-smart-diff-scanner)
  - [`usage recommend` (AI Intelligence & Run-Rate Forecasting)](#usage-recommend-ai-intelligence--run-rate-forecasting)
  - [`usage stats` (Historical SQLite Analytics)](#usage-stats-historical-sqlite-analytics)
  - [`usage keys` (Interactive AES-256 Vault Manager)](#usage-keys-interactive-aes-256-vault-manager)
  - [`usage providers` (Inventory & Encryption Status)](#usage-providers-inventory--encryption-status)
  - [`usage add-provider` (Manual Configuration)](#usage-add-provider-manual-configuration)
  - [`usage update-key` (Secret Rotation)](#usage-update-key-secret-rotation)
  - [`usage set-cycle` (Billing Reset Day Configuration)](#usage-set-cycle-billing-reset-day-configuration)
  - [`usage prune-stale` (Inactivity Garbage Collection)](#usage-prune-stale-inactivity-garbage-collection)
  - [`usage remove-provider` (Deletion)](#usage-remove-provider-deletion)
  - [`usage history` (Snapshot Log)](#usage-history-snapshot-log)
- [📅 Billing Cycle Anchors Explained](#billing-cycle-anchors-explained)
- [🔐 Security & Cryptography Architecture](#security--cryptography-architecture)
- [🗄️ Storage & Database Architecture](#storage--database-architecture)
- [💡 Power-User Recipes & Automation](#power-user-recipes--automation)
  - [1. Shell Prompt Widget (Starship / Bash / Zsh)](#1-shell-prompt-widget-starship--bash--zsh)
  - [2. Tmux Status Line Integration](#2-tmux-status-line-integration)
  - [3. Automated Daily Cron Snapshot](#3-automated-daily-cron-snapshot)
  - [4. Quota Exhaustion Desktop Alert Script](#4-quota-exhaustion-desktop-alert-script)
  - [5. Scripting with `jq` & JSON Export](#5-scripting-with-jq--json-export)
  - [6. CSV Financial & Tax Reconciliation](#6-csv-financial--tax-reconciliation)
  - [7. Tracking Local LLMs via Ollama](#7-tracking-local-llms-via-ollama)
- [⚙️ Configuration Specification (`config.json`)](#configuration-specification-configjson)
- [❓ Troubleshooting & FAQ](#troubleshooting--faq)
- [🧪 Testing & Contributing](#testing--contributing)
- [👤 Author & License](#author--license)

---

## 🎯 Why `usage`?

Modern software engineering involves multiple AI assistants running simultaneously across different workflows:
- Coding inside **Cursor IDE** or **VS Code with GitHub Copilot**.
- Running terminal coding agents like **Claude Code**, **Antigravity CLI**, or **Codex CLI**.
- Executing local privacy-focused models in **Ollama** or **vLLM**.
- Running custom scripts or batch jobs against **Anthropic**, **OpenAI**, **Gemini**, or **DeepSeek** APIs.

### The Problem
1. **Scattered Metrics**: Quotas are tracked in disparate units (tokens, fast requests, completion credits, or subscription tiers).
2. **Mismatched Billing Cycles**: Claude Pro may renew on the 14th of the month, OpenAI on the 23rd, and Copilot on the 1st. Standard calendar-month tools provide inaccurate percentages.
3. **Unexpected Quota Exhaustion**: Hitting hard rate limits or running out of tokens mid-task disrupts developer flow.
4. **Credential Security Risks**: API keys are often left unencrypted in shell histories or plaintext dotfiles.
5. **Inefficient Routing**: High-spend flagship models (e.g. Claude 3.5 Opus or GPT-4o) are frequently used for trivial tasks where lightweight models (Gemini Flash, Claude Haiku, or local Llama) could achieve identical results at 70% lower cost.

### The Solution: `usage`
`usage` unifies all your AI consumption into a single, cohesive developer tool. It detects your installed tools, tracks exact billing windows per provider, forecasts monthly burn rates before you exhaust your quota, encrypts all credentials at rest, and provides a real-time dashboard right in your terminal.

---

## ⚡ Key Features

- ⚡ **Sub-5ms Execution Time**: Written in optimized Go with concurrent goroutines. Providers degrade gracefully with strict timeouts without blocking the rest of the report.
- 📊 **Ranked Consumption Overview**: Automatically ranks models from most consumed to least consumed, showing remaining allowances, percentage used, and estimated USD cost.
- 🖥️ **Live Interactive TUI Dashboard (`usage watch`)**: Full terminal dashboard with auto-refresh, keyboard navigation, dynamic filtering, multi-column sorting, and provider inspection cards.
- 🔍 **Zero-Config Machine Scanner (`usage scan new`)**: Scans local configs, active sessions, and environment variables across 15+ AI ecosystems with interactive or automated 1-click import.
- 🧠 **AI Intelligence & Quota Forecasting (`usage recommend`)**: Analyzes daily burn rate and billing cycles to forecast exact exhaustion dates, identify underutilized subscriptions, and calculate cost savings.
- 📈 **SQLite Historical Analytics (`usage stats`)**: Automatically persists snapshots to a pure Go embedded SQLite database with Write-Ahead Logging (WAL). Query daily time-series and peak consumption.
- 🔐 **AES-256-GCM Encrypted Vault (`usage keys`)**: All keys and tokens are stored encrypted using a machine-local 256-bit vault key. Interactive key manager reveals secrets only on explicit operator request.
- 📅 **Per-Provider Billing Cycle Anchors (`usage set-cycle`)**: Assign individual billing renewal days (1–28) per provider or set a global anchor.
- 🧹 **Stale Provider Garbage Collection (`usage prune-stale`)**: Flags and removes inactive provider configurations with zero activity over 30+ days.
- 📤 **Machine-Readable Exports**: One-flag export to formatted JSON (`--json`) or CSV (`--csv`) for piping into `jq`, reporting scripts, or financial spreadsheets.

---

## 📦 Project Architecture

```
ai-usage/
├── cmd/
│   └── usage/
│       ├── main.go                 # CLI routing, flag parsing & subcommands
│       └── main_test.go            # CLI integration tests
├── internal/
│   ├── analytics/
│   │   ├── engine.go               # AI intelligence, quota run-rate & cost optimization
│   │   └── engine_test.go          # Analytics rule engine test suite
│   ├── config/
│   │   ├── config.go               # Config loader (~/.config/ai-usage/config.json)
│   │   ├── config_test.go          # Config unit tests & migration validation
│   │   ├── crypto.go               # AES-256-GCM encryption & vault key derivation
│   │   └── crypto_test.go          # Cryptography unit tests
│   ├── core/
│   │   ├── model.go                # Domain models, formatting & progress bar math
│   │   ├── billing.go              # Billing window calculation & leap-year handling
│   │   ├── billing_test.go         # Billing window test suite
│   │   ├── aggregator.go           # Concurrent goroutine worker pool & ranking logic
│   │   └── aggregator_test.go      # Aggregator concurrency test suite
│   ├── credential/
│   │   ├── credential.go           # Secret classification (API Key, OAuth, Session) & JWT parser
│   │   └── credential_test.go      # Credential detection test suite
│   ├── provider/
│   │   ├── provider.go             # ProviderAdapter interface & resilient HTTP client
│   │   ├── antigravity.go          # Local Antigravity Agent session telemetry adapter
│   │   ├── claude.go               # Anthropic Claude adapter (API Key & OAuth beta)
│   │   ├── codex.go                # OpenAI / Codex adapter (API Key & JWT session)
│   │   ├── gemini.go               # Google Gemini adapter (API Key & OAuth bearer)
│   │   ├── copilot.go              # GitHub Copilot adapter (gh CLI & OAuth)
│   │   ├── cursor.go               # Cursor IDE local SQLite tracking adapter
│   │   ├── warp.go                 # Warp Terminal local telemetry adapter
│   │   ├── custom.go               # Generic OpenAI-compatible adapter (DeepSeek, Groq, etc.)
│   │   └── provider_test.go        # Provider adapter test suite
│   ├── scanner/
│   │   ├── scanner.go              # Filesystem, service & environment diff scanner
│   │   └── scanner_test.go         # Scanner test suite & hermetic mock tests
│   ├── storage/
│   │   ├── sqlite.go               # Embedded SQLite store (WAL mode & daily trends)
│   │   ├── storage.go              # JSON file fallback & history store interface
│   │   └── storage_test.go         # Storage test suite
│   └── ui/
│       ├── table.go                # Terminal table & Unicode progress bar renderer
│       ├── dashboard.go            # Interactive live TUI dashboard & event loop
│       ├── dashboard_test.go       # TUI layout & sorting test suite
│       ├── keys.go                 # Interactive secure key manager UI
│       ├── recommend.go            # AI recommendations card renderer
│       ├── scan.go                 # Scanner review & interactive import cards
│       ├── format.go               # Historical trends & summary tables
│       ├── json.go                 # JSON and CSV output serializers
│       ├── rawmode_unix.go         # Unix terminal raw mode controls
│       └── rawmode_windows.go      # Windows console raw mode controls
├── dist/                           # Compiled binaries (generated on build)
├── Makefile                        # Multi-platform build & test recipes
└── go.mod                          # Go module definition
```

---

## 🚀 Quick Start Guide (60 Seconds)

### Step 1: Scan & Import Your AI Tools
Scan your system for installed tools, config files, and environment variables:
```bash
usage scan new
```
`usage` will discover your credentials (e.g. Claude, OpenAI, Gemini, Copilot, Cursor, Warp, Ollama) and prompt you to import them with recommended quotas and billing cycles:
```text
👉 Would you like to add provider 'Anthropic Claude' (Claude 3.5 Sonnet / Opus, Quota: 5.00M tokens, Cycle: 1st)?
   [Y]es / [n]o / [e]dit / [a]ll / [s]kip (Default: Y): y
   ✔ Added and encrypted 'Anthropic Claude' (Cycle resets on 1st).
```
*(Tip: Use `usage scan new --yes` to import all discovered providers non-interactively).*

### Step 2: View Your Live Consumption
Run `usage` with no arguments to get an instant ranked usage table:
```bash
usage
```
Output:
```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│  🚀 AI MODEL USAGE MONITOR                                                                    │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│  📅 Current Time   : 2026-09-04 17:36:21 BST       📊 Total Providers: 7 (Active: 7)         │
│  💳 Default Window : 2026-09-01 ➔ 2026-09-30       💰 Total Est Cost : $49.20                │
└──────────────────────────────────────────────────────────────────────────────────────────────┘

RANK   PROVIDER            MODEL / TIER                 CYCLE WINDOW   CONSUMED        QUOTA    REMAINING   USAGE %   PROGRESS BAR     COST (USD)   SOURCE / STATUS
----   --------            ------------                 ------------   --------        -----    ---------   -------   ------------     ----------   ---------------
#1     OpenAI Codex        GPT-4o / Codex               09/01➔09/30    2.15M tokens    4.00M    1.85M        53.8%    [■■■■■■······]   $ 21.50      ✔ Live API
#2     GitHub Copilot      Individual Plan              09/01➔09/30    1240 requests   3000     1760         41.3%    [■■■■■·······]   $ 10.00      ✔ Live API
#3     Anthropic Claude    Claude 3.5 Sonnet / Opus     09/01➔09/30    1.85M tokens    5.00M    3.15M        36.9%    [■■■■········]   $ 27.68      ✔ Live API
#4     Cursor IDE          Cursor Pro (Fast Requests)   09/01➔09/30    138 requests    500      362          27.6%    [■■■·········]   $ 20.00      ✔ Cursor Local DB
#5     Google Gemini       Gemini 1.5 Pro / Flash       09/01➔09/30    920.0K tokens   4.00M    3.08M        23.0%    [■■■·········]   $  3.22      ✔ Live API
#6     Antigravity Agent   Agent Pro Runtime            09/01➔09/30    1.02M tokens    10.00M   8.98M        10.2%    [■···········]   $  0.00      ✔ Local Session Logs
#7     Warp Terminal       Warp AI / Agent              09/01➔09/30    24 requests     100      76           24.0%    [■■■·········]   $  0.00      ✔ Warp Session
```

### Step 3: Launch the Interactive Dashboard
Monitor your usage in real-time with an auto-refreshing TUI:
```bash
usage watch
```

### Step 4: Check Intelligence & Run-Rate Alerts
Check if you are burning through quota too fast before the month ends:
```bash
usage recommend
```

---

## 📥 Installation & Build

### Prerequisites
- **Go 1.27+** (the version in `go.mod`) if compiling from source.
- Standard Linux, macOS, or Windows 10/11 environment.

### Download a Release
Prebuilt, CGO-free binaries for Linux (amd64, arm64), macOS (arm64) and Windows
(amd64), with a `SHA256SUMS` file, are attached to each
[GitHub release](https://github.com/sabrodigan/ai-usage-command/releases):
```bash
gh release download --repo sabrodigan/ai-usage-command --pattern usage-linux-amd64 --pattern SHA256SUMS
sha256sum --check --ignore-missing SHA256SUMS
install -m 755 usage-linux-amd64 ~/.local/bin/usage
```

### Compile and Install Locally
Clone the repository and run:
```bash
# 1. Build binary for your host operating system
make build

# 2. Install to ~/.local/bin/usage and ~/go/bin/usage
make install
```

### Cross-Compile for All Platforms
```bash
make build-all
```
Binaries and `SHA256SUMS` are written to `dist/` (git-ignored):
`usage-linux-amd64`, `usage-linux-arm64`, `usage-darwin-arm64`, `usage-windows-amd64.exe`.

### Tests, CI and Releases
| Command | What it does |
|---|---|
| `make check` | `gofmt` check, `go vet`, and `go test -race` — the same gate CI runs on every push and PR |
| `make plugin-test` | Builds `usage` and runs the [Omarchy widget](https://github.com/sabrodigan/omarchy-ai-usage)'s tests against it, including an end-to-end contract test of `usage local --json` (checkout expected at `~/Projects/omarchy-ai-usage`, override with `PLUGIN_DIR=`) |
| `make version` | Prints the version from `const Version` in `cmd/usage/main.go` |
| `make release` | Runs both test suites, refuses a dirty tree or an existing tag, then tags `v<version>` and pushes it |

Pushing a `v*` tag triggers `.github/workflows/release.yml`, which checks that the
tag matches `const Version`, reruns the tests, cross-compiles, and publishes the
GitHub release with binaries and checksums. To cut a release: bump `const Version`,
commit, then `make release`.

### Configure Your Shell PATH
Ensure `~/.local/bin` is in your shell `PATH`:

#### Bash (`~/.bashrc`) / Zsh (`~/.zshrc`):
```bash
export PATH="$HOME/.local/bin:$PATH"
```
Then reload:
```bash
source ~/.bashrc  # or source ~/.zshrc
```

#### Fish (`~/.config/fish/config.fish`):
```fish
fish_add_path ~/.local/bin
```

#### Windows (PowerShell):
```powershell
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$HOME\.local\bin", "User")
```

---

## 🤖 Supported Providers & Detection Matrix

`usage` comes pre-configured with intelligent adapters for major AI developer tools:

| Provider | Adapter ID | Metric Unit | Detection Sources | Credential Types Handled | Default Quota |
|---|---|---|---|---|---|
| **Antigravity Agent** | `antigravity` | Tokens | `~/.gemini/antigravity-cli/brain/*.jsonl` | Local Session Telemetry | 10,000,000 tokens ($0 bundled) |
| **Anthropic Claude** | `claude` | Tokens | `~/.claude/.credentials.json`<br>`~/.claude.json`<br>`$ANTHROPIC_API_KEY` | API Key (`sk-ant-api...`)<br>OAuth Bearer (`sk-ant-oat...`) | 5,000,000 tokens |
| **OpenAI Codex** | `codex` | Tokens | `~/.codex/auth.json`<br>`~/.openai/api_key`<br>`$OPENAI_API_KEY` | API Key (`sk-...`)<br>OAuth JWT Access Token | 4,000,000 tokens |
| **Google Gemini** | `gemini` | Tokens | `~/.gemini/oauth_creds.json`<br>`$GEMINI_API_KEY` | Google API Key (`AIza...`)<br>OAuth Bearer (`ya29...`) | 4,000,000 tokens |
| **GitHub Copilot** | `copilot` | Requests | `~/.config/gh/hosts.yml`<br>`$GITHUB_TOKEN` | GitHub OAuth / PAT (`gho_`, `ghp_`) | 3,000 requests |
| **Cursor IDE** | `cursor` | Requests (per model, e.g. `grok-4.6`, `auto`) | `~/.config/Cursor/User/globalStorage/state.vscdb` (chat prompts)<br>`~/.cursor/ai-tracking/ai-code-tracking.db` (fallback) | Local IDE SQLite Database | 500 fast requests |
| **Meta Muse** | `muse` | Tokens (per model, priced from Muse's model catalog) | `~/.local/share/muse/sessions/**/session.jsonl`<br>`~/.config/muse/settings.json` | Local Session Telemetry (Meta token never imported) | 50,000,000 tokens |
| **Warp Terminal** | `warp` | Requests | `~/.warp` | Local Terminal Session | 100 requests |
| **Ollama Local** | `ollama` | Tokens | Local HTTP daemon: `http://127.0.0.1:11434` or `$OLLAMA_HOST` | Local Session (No Secret) | 50,000,000 tokens |
| **Continue.dev** | *(dynamic)* | Tokens | `~/.continue/config.json` | API Keys for configured models | 5,000,000 tokens |
| **DeepSeek AI** | `deepseek` | Tokens | `$DEEPSEEK_API_KEY` | First-party API Key | 10,000,000 tokens |
| **Groq Cloud** | `groq` | Tokens | `$GROQ_API_KEY` | First-party API Key | 10,000,000 tokens |
| **OpenRouter** | `openrouter` | Tokens | `$OPENROUTER_API_KEY` | First-party API Key | 10,000,000 tokens |
| **Mistral AI** | `mistral` | Tokens | `$MISTRAL_API_KEY` | First-party API Key | 10,000,000 tokens |
| **Perplexity AI**| `perplexity`| Tokens | `$PERPLEXITY_API_KEY` | First-party API Key | 5,000,000 tokens |
| **xAI (Grok)** | `xai` | Tokens | `$XAI_API_KEY` or `$GROK_API_KEY` | First-party API Key | 10,000,000 tokens |
| **Together AI** | `together` | Tokens | `$TOGETHER_API_KEY` | First-party API Key | 10,000,000 tokens |
| **Cohere** | `cohere` | Tokens | `$COHERE_API_KEY` | First-party API Key | 5,000,000 tokens |
| **Custom Endpoint**| Any string | Any | Custom endpoint URL via `add-provider` | API Key / Bearer Token | Custom |

---

## 🛠️ Complete Command Reference

### `usage` / `usage live` / `usage now`

Fetches and displays live consumption data from all configured providers.

```bash
# Standard live table
usage

# Alias commands
usage live
usage now

# Adjust query timeout (default: 6 seconds)
usage now --timeout 10

# Output as formatted JSON
usage now --json

# Output as CSV for spreadsheets
usage now --csv
```

### `usage local`

Reports usage read only from local agent session data — Meta Muse, Cursor, Antigravity and Warp — with no stored keys decrypted and no network calls. Tools are picked up as soon as their data exists on disk, even before `usage scan --import`. Local providers include a `models` breakdown in JSON (per-model requests, input/cached/output tokens and estimated cost), and the Model/Tier column lists the models used, e.g. `grok-4.6 (8) · auto (1)`.

```bash
usage local
usage local --json

# Ignore ~/.config/ai-usage entirely (calendar month, default quotas)
usage local --json --no-config
```

**Table Columns**:
- **RANK**: Priority ranking calculated by percentage of quota consumed.
- **PROVIDER**: Name of the AI provider or service.
- **MODEL / TIER**: Active model family or subscription plan.
- **CYCLE WINDOW**: The active billing window (`MM/DD➔MM/DD`) calculated for this provider.
- **CONSUMED**: Current usage in thousands (`K`), millions (`M`), or request counts.
- **QUOTA**: Configured monthly quota.
- **REMAINING**: Available quota remaining before hitting rate limit / exhaustion.
- **USAGE %**: Consumed divided by Quota.
- **PROGRESS BAR**: Visual Unicode representation (`[■■■■■·······]`).
- **COST (USD)**: Estimated monthly cost incurred.
- **SOURCE / STATUS**: Data source origin (Live API, Local SQLite, Session Log) and health status (`✔ OK`, `⚠ Quota Exceeded`, `✖ Offline`).

---

### `usage example` / `usage demo`

Displays representative benchmark simulation data across 9 sample providers. Useful for testing terminal font compatibility, verifying layout widths, or demonstration without initiating network API calls.

```bash
usage example
# or
usage demo
```

---

### `usage watch` / `usage tui` (Interactive Live Dashboard)

Launches an interactive, auto-refreshing Terminal User Interface (TUI) with real-time telemetry.

```bash
# Default (auto-refreshes every 3 seconds)
usage watch

# Aliases
usage tui
usage dashboard

# Set custom refresh interval (e.g. 5 seconds)
usage watch --interval 5

# Pre-filter by provider name or ID
usage watch --filter claude
```

#### 🎮 Interactive Keyboard Controls

| Key | Action | Description |
|:---:|---|---|
| `q` / `Ctrl+C` | **Quit** | Cleanly exits the dashboard and restores terminal settings |
| `r` | **Refresh** | Immediately queries all adapters and refreshes telemetry |
| `↑` / `k` | **Select Up** | Moves highlight cursor to the previous provider |
| `↓` / `j` | **Select Down** | Moves highlight cursor to the next provider |
| `Space` / `Enter` | **Inspect Card** | Toggles expanded modal card with detailed quota metrics and diagnostics |
| `s` | **Cycle Sort** | Cycles sort order: Usage % ➔ Consumed ➔ Quota ➔ Remaining ➔ Cost ➔ Provider Name |
| `/` or `f` | **Filter** | Opens dynamic search prompt to filter providers by name or model tier in real time |
| `+` / `=` | **Faster Rate** | Decreases refresh interval by 1 second (down to 1s minimum) |
| `-` | **Slower Rate** | Increases refresh interval by 1 second (up to 60s maximum) |
| `h` or `?` | **Help Overlay** | Toggles the in-dashboard shortcut and legend overlay |

*(Note: On launch, `usage watch` automatically runs a silent background discovery scan to detect any newly added AI tools or CLI updates).*

---

### `usage scan` & `usage scan new` (Smart Diff Scanner)

Inspects the host machine for AI configuration files, OAuth sessions, and environment variables.

#### 1. Full Environment Inventory (`usage scan`)
Displays all detected credentials alongside currently configured providers:
```bash
usage scan

# Automatically import all detected credentials
usage scan --import
```

#### 2. Smart Diff Scanner (`usage scan new`)
Focuses specifically on newly discovered or updated credentials that are not yet in your configuration:
```bash
usage scan new
```

**Interactive Step-by-Step Prompt**:
For each discovered tool, `usage scan new` offers full control:
```text
👉 Would you like to add provider 'DeepSeek AI' (DeepSeek-V3, Quota: 10.00M tokens, Cycle: 1st)?
   [Y]es / [n]o / [e]dit / [a]ll / [s]kip (Default: Y):
```
- `Y` or `Enter`: Imports the provider with recommended defaults.
- `n` or `s`: Skips this provider.
- `e` (`edit`): Prompts you to customize the monthly quota and billing reset anchor day (1–28).
- `a` (`all`): Automatically imports all remaining discovered providers.

#### Non-Interactive Import:
```bash
# Accept all new providers immediately without prompting
usage scan new --yes
# or
usage scan new -y
```

#### Output Discovered Credentials as JSON:
```bash
usage scan new --json
```

---

### `usage recommend` (AI Intelligence & Run-Rate Forecasting)

Executes an intelligent analysis of your current consumption, historical trajectory, and billing cycles to deliver actionable insights.

```bash
usage recommend
# Aliases: usage recommendations, usage insights
```

Example output:
```text
┌────────────────────────────────────────────────────────────────────────┐
│  🧠 AI USAGE INTELLIGENCE & RECOMMENDATIONS                            │
└────────────────────────────────────────────────────────────────────────┘

🚨 [Quota Risk] Antigravity Agent Quota Run-Rate Alert
   At your current burn rate (890,369 tokens/day), you are projected to reach 26,711,083 tokens (267% of quota) and exhaust remaining allowance in ~7.5 days (Cycle resets on Sep 30).

🚨 [Quota Risk] Cursor IDE Quota Run-Rate Alert
   At your current burn rate (40 requests/day), you are projected to reach 1,205 requests (241% of quota) and exhaust remaining allowance in ~8.7 days (Cycle resets on Sep 30).

💡 [Cost Optimization] Cost Optimization for Anthropic Claude
   You have consumed 1.85M tokens on Claude 3.5 Sonnet ($27.68 est. cost). Routing automated summaries and background workflows to fast tier models (e.g. Gemini 1.5 Flash or Claude 3.5 Haiku) can reduce cost by up to 70%.

💡 [Workload Routing] Leverage Included Antigravity Agent Quota
   Your Antigravity agent interactions are bundled in your Google subscription ($0.00 extra cost). Offloading large refactors and codebase searches here preserves paid tokens on other APIs.
```

#### Recommendation Categories:
1. **Quota Risk (🚨 HIGH)**: Alerts when current burn rate will exhaust quota before the billing reset day.
2. **Cost Optimization (💡 MEDIUM)**: Identifies expensive model consumption and calculates exact USD savings by shifting sub-tasks to efficient models.
3. **Subscription Underutilization (📉 LOW)**: Flags paid monthly plans where less than 5% of the quota has been used near the end of the billing period.
4. **Workload Routing (🎯 TIP)**: Reminds developers of zero-cost bundled quotas (e.g. Antigravity Agent) to offload large batch operations.

---

### `usage stats` (Historical SQLite Analytics)

Queries the local embedded SQLite database (`usage.db`) to display daily historical consumption trends and peak usage per provider.

```bash
# Default lookback (last 30 days)
usage stats

# Custom lookback window (e.g. 7 days or 90 days)
usage stats --days 7
usage stats --days 90
```

Example output:
```text
📊 Historical Daily AI Consumption (Local SQLite Database):
────────────────────────────────────────────────────────────────────────
DATE         PROVIDER            PEAK CONSUMED   EST. COST   USAGE %
----         --------            -------------   ---------   -------
2026-09-04   Antigravity Agent   3,321,104       $0.00        33.2%
2026-09-04   Cursor IDE          150             $20.00       30.0%
2026-09-03   Antigravity Agent   3,166,538       $0.00        31.7%
2026-09-03   Cursor IDE          150             $20.00       30.0%
2026-09-02   Anthropic Claude    58,818          $0.19         1.2%
2026-09-01   OpenAI Codex        2,150,000       $21.50       53.8%
2026-09-01   GitHub Copilot      1,240           $10.00       41.3%
2026-09-01   Anthropic Claude    1,845,200       $27.68       36.9%
```

---

### `usage keys` (Interactive AES-256 Vault Manager)

Inspects configured API keys securely in your terminal. All keys are encrypted at rest with AES-256-GCM and masked by default in the terminal.

```bash
usage keys
# Aliases: usage key, usage show-keys
```

**Interactive Interface**:
```text
┌────────────────────────────────────────────────────────────────────────┐
│  🔐 SECURE API KEY MANAGER (AES-256 ENCRYPTED)                         │
└────────────────────────────────────────────────────────────────────────┘

[1] Anthropic Claude  : sk-ant...0wAA  [ENCRYPTED]
[2] Google Gemini     : ya29.a...0211  [ENCRYPTED]
[3] Groq Cloud        : gsk_74...9X4a  [ENCRYPTED]
[4] OpenAI Codex      : eyJhbG...irtg  [ENCRYPTED]

Controls:
  • Press [1-4] to reveal or hide a specific key in plaintext
  • Press [0] or [q] to exit
```
- Pressing `1` decrypts and temporarily reveals the Claude key in memory on screen. Pressing `1` again re-masks it.
- Secrets never touch disk in unencrypted form.

---

### `usage providers` (Inventory & Encryption Status)

Lists all currently configured providers, their active status, quotas, billing reset days, credential authentication classification, and encryption verification.

```bash
usage providers
```

Example output:
```text
Configured AI Model Providers:
───────────────────────────────────────────────────────────────────────────────────────────────────
ID              NAME                 STATUS     QUOTA        RESET DAY  CRED      ENCRYPTED    MODEL / TIER
───────────────────────────────────────────────────────────────────────────────────────────────────
antigravity     Antigravity Agent    Active     10.00M       1st        session   No Key       Agent Pro Runtime
claude          Anthropic Claude     Active     5.00M        1st        oauth     AES-256 ✔    Claude 3.5 Sonnet / Opus
codex           OpenAI Codex         Active     4.00M        1st        oauth     AES-256 ✔    GPT-4o / Codex
copilot         GitHub Copilot       Active     3000         1st        session   No Key       Individual Plan
cursor          Cursor IDE           Active     500          1st        session   No Key       Cursor Pro (Fast Requests)
gemini          Google Gemini        Active     4.00M        1st        oauth     AES-256 ✔    Gemini 1.5 Pro / Flash
groq            Groq Cloud           Active     10.00M       1st        api_key   AES-256 ✔    Llama 3.3
mistral         Mistral AI           Active     5.00M        5th        api_key   AES-256 ✔    Codestral
warp            Warp Terminal        Active     100          1st        session   No Key       Warp AI / Agent
───────────────────────────────────────────────────────────────────────────────────────────────────
Default global billing cycle anchor: 1st of each month.
```

---

### `usage add-provider` (Manual Configuration)

Adds or updates an AI provider manually.

```bash
# Add custom DeepSeek endpoint with custom quota and cycle reset date
usage add-provider \
  --id deepseek \
  --name "DeepSeek AI" \
  --endpoint "https://api.deepseek.com" \
  --key "sk-dseek-xxx" \
  --quota 15000000 \
  --tier "DeepSeek-V3" \
  --cycle 14

# Add local Ollama endpoint
usage add-provider \
  --id ollama \
  --name "Ollama Local" \
  --endpoint "http://localhost:11434" \
  --quota 50000000 \
  --tier "Llama-3.3-70B"

# Add OpenAI-compatible private vLLM or LM Studio gateway
usage add-provider \
  --id vllm \
  --name "Internal vLLM Cluster" \
  --endpoint "http://ai-gateway.internal:8000/v1" \
  --key "cluster-token-xxx" \
  --quota 25000000 \
  --tier "Qwen-2.5-Coder-32B"

# Positional shortcut syntax: usage add-provider <id> <key> [<quota>]
usage add-provider groq gsk_xxx 10000000
```

---

### `usage update-key` (Secret Rotation)

Replaces an existing provider's API key, re-encrypts the secret with AES-256-GCM, and resets the provider's activity timestamp.

```bash
usage update-key claude sk-ant-api03-new-secret-key-xxx

# Or using flag syntax
usage update-key --id codex --key sk-proj-new-token-xxx
```

---

### `usage set-cycle` (Billing Reset Day Configuration)

Sets the monthly billing cycle reset day (1st through 28th).

```bash
# Set custom billing reset anchor for a specific provider
usage set-cycle claude 14     # Claude renews on the 14th of each month
usage set-cycle mistral 5     # Mistral renews on the 5th of each month

# Set global default billing reset anchor for all providers
usage set-cycle 1             # Default to calendar month (1st)
```

---

### `usage prune-stale` (Inactivity Garbage Collection)

Removes configured providers that have had no telemetry or API activity for an extended period (default: 30 days). Antigravity Agent is always preserved.

```bash
# Prune providers inactive for 30+ days
usage prune-stale

# Specify custom inactivity threshold (e.g. 14 days or 60 days)
usage prune-stale --days 14
usage prune-stale --days 60
```

---

### `usage remove-provider` (Deletion)

Deletes a configured provider from your configuration file.

```bash
usage remove-provider deepseek
usage remove-provider vllm
```

---

### `usage history` (Snapshot Log)

Displays recent recorded snapshot logs, including total cost and active provider counts.

```bash
usage history
```

---

## 📅 Billing Cycle Anchors Explained

Most billing monitors assume every service follows the standard calendar month (the 1st to the 30th/31st). In reality, developer subscriptions renew on arbitrary dates:
- **Anthropic Claude Pro**: Renews on the day you subscribed (e.g. the 14th).
- **OpenAI API / ChatGPT**: Often resets on the 23rd or account creation day.
- **GitHub Copilot**: Billed monthly on your GitHub invoice date.

### How `usage` Calculates Exact Windows
Given a target anchor day $D$ (between 1 and 28):
1. If the current date is on or after day $D$, the window starts on day $D$ of the **current month** and ends at 23:59:59 on day $D - 1$ of the **next month**.
2. If the current date is before day $D$, the window starts on day $D$ of the **previous month** and ends at 23:59:59 on day $D - 1$ of the **current month**.
3. All leap years (February 29), month-length variations (28/30/31 days), and local time zones are handled deterministically.

```
Example: Provider with Anchor Day = 14
Today: September 4th
Billing Window: August 14th ➔ September 13th (23:59:59)
```

This guarantees that:
- Quotas and percentages reflect **actual usage within the current billing cycle**.
- Daily burn-rate projections correctly calculate the exact days remaining until your quota resets.

---

## 🔐 Security & Cryptography Architecture

`usage` treats developer API keys as sensitive zero-trust secrets.

```
                   ┌──────────────────────────────────────────────┐
                   │               Plaintext Key                  │
                   │           (e.g., sk-ant-api...)              │
                   └──────────────────────┬───────────────────────┘
                                          │
                                          ▼
                      ┌───────────────────────────────────────┐
                      │ AES-256-GCM Authenticated Encryption  │
                      │   - 96-bit cryptographically random   │
                      │     nonce per encryption invocation   │
                      │   - Machine-local 256-bit vault key   │
                      │     (~/.config/ai-usage/.vault.key)   │
                      └───────────────────┬───────────────────┘
                                          │
                                          ▼
                   ┌──────────────────────────────────────────────┐
                   │    enc:v1:<base64-encoded-ciphertext>        │
                   │        Stored safely in config.json          │
                   └──────────────────────────────────────────────┘
```

1. **At-Rest Encryption**: Plaintext keys are never stored in `config.json`. All secrets are encrypted with `AES-256-GCM` and prefixed with `enc:v1:`.
2. **Local Vault Key**: A 256-bit machine-local master key is generated from `crypto/rand` upon first run and stored at `~/.config/ai-usage/.vault.key` with strict `0600` (read/write only by file owner) filesystem permissions.
3. **Secret Classification**: Categorizes credentials into `api_key`, `oauth`, or `session`. For example, OAuth tokens use `Authorization: Bearer` and beta headers, while API keys use provider-specific headers (e.g. `x-api-key`).
4. **JWT Expiry Validation**: Inspects JWT tokens (such as ChatGPT OAuth tokens or Google access tokens) in memory to verify the `exp` claim before making network calls, preventing failed API invocations.
5. **No External Leakage**: Telemetry is logged strictly into local SQLite storage. `usage` does not contain telemetry beacons, metrics uploaders, or phone-home calls.

---

## 🗄️ Storage & Database Architecture

Telemetry is stored entirely locally using two tiers of persistence:

### 1. Embedded SQLite Database (`usage.db`)
Located at `~/.config/ai-usage/usage.db` (or `%APPDATA%\ai-usage\usage.db` on Windows).
- Utilizes pure Go SQLite (`modernc.org/sqlite`) requiring **no external C compiler (`CGO_ENABLED=0`)** and zero SQLite system packages.
- Enabled with **Write-Ahead Logging (`PRAGMA journal_mode=WAL`)** and `PRAGMA synchronous=NORMAL` for concurrent query support and high write throughput.

#### SQLite Schema:
- **`usage_snapshots`**: Records overall snapshot summaries (`timestamp`, `total_providers`, `active_ok_count`, `total_tokens`, `total_cost_usd`).
- **`provider_records`**: Indexed per-provider breakdowns (`snapshot_id`, `provider_id`, `model_tier`, `consumed`, `quota`, `percent_used`, `cost_usd`, `cycle_start`, `cycle_end`, `status`).
- **`ai_recommendations`**: Stored optimization insights and run-rate projections.

### 2. Configuration & Vault Storage
- **Linux**: `~/.config/ai-usage/config.json`
- **Windows**: `%APPDATA%\ai-usage\config.json`
- **macOS**: `~/Library/Application Support/ai-usage/config.json`

File permissions are enforced at `0600` for both `config.json` and `.vault.key`.

---

## 💡 Power-User Recipes & Automation

### 1. Shell Prompt Widget (Starship / Bash / Zsh)
Display your top AI model's quota percentage in your terminal prompt.

#### Using `jq` and `usage now --json`:
```bash
# Add to ~/.bashrc or ~/.zshrc
ai_usage_prompt() {
  local top
  top=$(usage now --json 2>/dev/null | jq -r '.providers[0] | "\(.display_name): \(.percent_used | round)%"' 2>/dev/null)
  if [ -n "$top" ]; then
    echo "🤖 $top "
  fi
}
```

#### Starship Prompt (`~/.config/starship.toml`):
```toml
[custom.ai_usage]
command = "usage now --json | jq -r '\"[\" + .providers[0].provider_id + \":\" + (.providers[0].percent_used | round | tostring) + \"%]\"'"
when = "command -v usage >/dev/null"
interval = 300
format = "[$output]($style) "
style = "bold cyan"
```

---

### 2. Tmux Status Line Integration
Keep an eye on monthly AI costs directly in your tmux status bar.

Add to `~/.tmux.conf`:
```tmux
set -g status-right '#(usage now --json 2>/dev/null | jq -r "\"AI: $\" + (.total_cost_usd | tostring) + \" (\" + (.active_ok_count | tostring) + \" active)\"") | %H:%M %d-%b'
```

---

### 3. Automated Daily Cron Snapshot
Run `usage now` automatically every day at 08:00 AM to build historical SQLite analytics:

```bash
crontab -e
```
Add the cron rule:
```cron
0 8 * * * /home/yourusername/.local/bin/usage now >/dev/null 2>&1
```

---

### 4. Quota Exhaustion Desktop Alert Script
Alert yourself with a desktop notification when any provider exceeds 80% of its quota.

Save as `~/.local/bin/ai-quota-alert.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail

usage now --json | jq -c '.providers[] | select(.percent_used >= 80.0)' | while read -r provider; do
  name=$(echo "$provider" | jq -r '.display_name')
  pct=$(echo "$provider" | jq -r '.percent_used | round')
  notify-send -u critical "⚠️ AI Quota Alert" "$name is at ${pct}% of its monthly quota!"
done
```
Make it executable:
```bash
chmod +x ~/.local/bin/ai-quota-alert.sh
```

---

### 5. Scripting with `jq` & JSON Export
Extract structured metrics for downstream automation:

```bash
# Total estimated USD spend across all models
usage now --json | jq '.total_cost_usd'

# List providers where quota consumed is greater than 50%
usage now --json | jq '.providers[] | select(.percent_used > 50) | {id: .provider_id, usage: .percent_used}'

# Extract active billing cycle for Claude
usage now --json | jq '.providers[] | select(.provider_id == "claude") | {start: .billing_start, end: .billing_end}'
```

---

### 6. CSV Financial & Tax Reconciliation
Export a monthly snapshot directly into accounting software (Excel, LibreOffice Calc, Google Sheets):

```bash
# Export to CSV
usage now --csv > ai-expenses-$(date +%Y-%m).csv
```

---

### 7. Tracking Local LLMs via Ollama
Track local model queries at zero cost:
```bash
# 1. Start Ollama
ollama serve &

# 2. Add Ollama to tracking
usage add-provider \
  --id ollama \
  --name "Ollama Local" \
  --endpoint "http://127.0.0.1:11434" \
  --quota 50000000 \
  --tier "Llama-3.3-70B"

# 3. View Ollama status
usage
```

---

## ⚙️ Configuration Specification (`config.json`)

Configuration is stored in JSON format at `~/.config/ai-usage/config.json`. Below is an annotated specification:

```json
{
  "anchor_billing_day": 1,
  "timeout_seconds": 6,
  "stale_warning_days": 30,
  "providers": {
    "claude": {
      "id": "claude",
      "display_name": "Anthropic Claude",
      "api_key": "enc:v1:qKzP8...==",
      "credential_type": "oauth",
      "custom_quota": 5000000,
      "model_tier": "Claude 3.5 Sonnet / Opus",
      "endpoint": "https://api.anthropic.com",
      "anchor_billing_day": 14,
      "enabled": true,
      "created_at": "2026-09-01T12:00:00Z",
      "last_activity": "2026-09-04T16:30:00Z"
    },
    "codex": {
      "id": "codex",
      "display_name": "OpenAI Codex",
      "api_key": "enc:v1:mNpR2...==",
      "credential_type": "api_key",
      "custom_quota": 4000000,
      "model_tier": "GPT-4o / Codex",
      "endpoint": "https://api.openai.com/v1",
      "anchor_billing_day": 1,
      "enabled": true,
      "created_at": "2026-09-01T12:00:00Z",
      "last_activity": "2026-09-04T16:30:00Z"
    },
    "cursor": {
      "id": "cursor",
      "display_name": "Cursor IDE",
      "custom_quota": 500,
      "model_tier": "Cursor Pro (Fast Requests)",
      "anchor_billing_day": 1,
      "enabled": true,
      "created_at": "2026-09-01T12:00:00Z",
      "last_activity": "2026-09-04T16:30:00Z"
    }
  }
}
```

---

## ❓ Troubleshooting & FAQ

### Q: Why does my quota show 4.00M or 5.00M by default?
**A**: Many provider APIs (such as Anthropic or standard OpenAI API accounts) report rate limits rather than a fixed prepaid spending ceiling. `usage` assigns sensible default monthly quotas (e.g. 5M tokens for Claude, 4M for Codex, 500 fast requests for Cursor Pro). You can customize this anytime:
```bash
usage add-provider --id claude --quota 15000000
```

### Q: What should I do if Claude reports "OAuth token has expired"?
**A**: If you authenticated Claude via the Claude Code CLI or browser OAuth, the local bearer token eventually expires. Re-authenticate with:
```bash
claude login
# or run
claude
```
`usage` will automatically detect the renewed access token on its next scan.

### Q: How does Antigravity Agent calculate token counts without an API key?
**A**: Antigravity runs locally via the Google Antigravity CLI. `usage` directly inspects the local session logs in `~/.gemini/antigravity-cli/brain/*.jsonl`, filtering strictly for interactions that occurred within your active billing window.

### Q: How do I back up or transfer my configuration to a new machine?
**A**: Because credentials are encrypted using a machine-local key, copy both files together:
```bash
# Source machine:
tar -czvf ai-usage-backup.tar.gz -C ~/.config/ai-usage config.json .vault.key

# Destination machine:
mkdir -p ~/.config/ai-usage
tar -xzvf ai-usage-backup.tar.gz -C ~/.config/ai-usage/
```

### Q: How do I completely reset my encryption key and configuration?
**A**: Remove the configuration directory:
```bash
rm -rf ~/.config/ai-usage
```
Running `usage` will recreate a fresh vault key and default settings.

---

## 🧪 Testing & Contributing

All unit tests and integration tests can be run using the standard Go test harness:

```bash
# Run test suite with race condition detector
make test

# Or run go test directly
go test -v -race ./...
```

Contributions, issue reports, and feature requests are welcome! When submitting a Pull Request:
1. Ensure `make test` passes without race conditions.
2. Maintain documentation integrity and update command tables if adding new flags.
3. Follow idiomatic Go guidelines and existing repository formatting.

---

## 👤 Author & License

- **Author**: **Stephen Brodigan**
- **License**: GNU Affero General Public License v3 ([AGPL-3.0](LICENSE.md))
