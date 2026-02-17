# ccost

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Code Size](https://img.shields.io/github/languages/code-size/zulerne/ccost)](https://github.com/zulerne/ccost)
[![Release](https://img.shields.io/github/v/release/zulerne/ccost)](https://github.com/zulerne/ccost/releases)

A fast, standalone CLI tool to analyze [Claude Code](https://claude.com/claude-code) token usage, costs, and session time.

Built with Go as a single binary — no runtime dependencies, no network access. Just reads your local JSONL session logs.

Alternative to [ccusage](https://github.com/ryoppippi/ccusage).

![demo](demo.gif)

## Installation

### Homebrew

```bash
brew install zulerne/tap/ccost
```

### Go install

```bash
go install github.com/zulerne/ccost/cmd/ccost@latest
```

### Build from source

```bash
git clone https://github.com/zulerne/ccost.git
cd ccost
make build
```

## Usage

```bash
# Daily breakdown (default)
ccost

# Filter by date range
ccost --since 2026-02-01
ccost -s 2026-02-01 -u 2026-02-07

# Filter by project name (substring match)
ccost --project myapp
ccost -p myapp

# Group by project
ccost --by-project

# Show per-model breakdown
ccost --models
ccost -m

# Combine flags
ccost --by-project --models --since 2026-02-01

# JSON output
ccost --json
ccost --json --models
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--since YYYY-MM-DD` | `-s` | Start date filter |
| `--until YYYY-MM-DD` | `-u` | End date filter (inclusive) |
| `--project NAME` | `-p` | Filter by project name (substring match) |
| `--by-project` | | Group by project instead of date |
| `--models` | `-m` | Show per-model token breakdown |
| `--exact` | `-e` | Show exact token counts instead of compact (K/M) |
| `--json` | | Output as JSON |

## How it works

`ccost` reads Claude Code's JSONL session logs from `~/.claude/projects/`, including subagent files. It:

1. Parses all `assistant` type entries with token usage
2. Deduplicates by message ID (keeps max `output_tokens` from streaming)
3. Normalizes model names (strips date suffixes like `-20250929`)
4. Calculates costs using [Anthropic's pricing](https://www.anthropic.com/pricing) with 1-hour ephemeral cache rates
5. Tracks session duration (first to last timestamp per main session file, excluding subagents)
6. Skips synthetic zero-token entries

## Supported models

| Model | Input | Output | Cache Write | Cache Read |
|-------|------:|-------:|------------:|-----------:|
| claude-opus-4-6 | $5.00 | $25.00 | $10.00 | $0.50 |
| claude-opus-4-5 | $5.00 | $25.00 | $10.00 | $0.50 |
| claude-sonnet-4-5 | $3.00 | $15.00 | $6.00 | $0.30 |
| claude-sonnet-4 | $3.00 | $15.00 | $6.00 | $0.30 |
| claude-haiku-4-5 | $1.00 | $5.00 | $2.00 | $0.10 |

*Prices per 1M tokens. Cache write/read use Claude Code's 1-hour ephemeral cache rates.*

Unknown models show token counts with cost as `N/A` and a warning on stderr.

## License

MIT
