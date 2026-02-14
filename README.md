# ccost

A fast, single-binary CLI tool to analyze [Claude Code](https://claude.com/claude-code) token usage and costs. No runtime dependencies, no network access — just reads your local JSONL logs.

An alternative to [ccusage](https://github.com/ryoppippi/ccusage) (npm) that doesn't require Node.js.

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
ccost -since 2026-02-01
ccost -since 2026-02-01 -until 2026-02-07

# Filter by project name (substring match)
ccost -project myapp

# Group by project
ccost -by-project

# Show per-model breakdown
ccost -models

# Combine flags
ccost -by-project -models -since 2026-02-01

# JSON output
ccost -json
ccost -json -models
```

## Examples

### Daily summary

```
╭─────────────┬────────┬─────────┬─────────────┬────────────┬────────╮
│ DATE (2026) │  INPUT │  OUTPUT │ CACHE WRITE │ CACHE READ │   COST │
├─────────────┼────────┼─────────┼─────────────┼────────────┼────────┤
│ 02-09       │ 19,772 │  92,844 │   2,024,707 │ 31,682,507 │ $33.59 │
│ 02-10       │ 12,940 │  89,386 │   1,275,034 │ 14,984,972 │ $18.09 │
│ 02-12       │     10 │      98 │      17,795 │          0 │  $0.11 │
│ 02-14       │ 27,027 │  39,998 │     833,920 │ 25,628,360 │ $19.08 │
├─────────────┼────────┼─────────┼─────────────┼────────────┼────────┤
│ TOTAL       │ 59,749 │ 222,326 │   4,151,456 │ 72,295,839 │ $70.87 │
╰─────────────┴────────┴─────────┴─────────────┴────────────┴────────╯
```

### Per-model breakdown (`-models`)

```
╭─────────────┬───────────────────┬────────┬─────────┬─────────────┬────────────┬────────╮
│ DATE (2026) │ MODEL             │  INPUT │  OUTPUT │ CACHE WRITE │ CACHE READ │   COST │
├─────────────┼───────────────────┼────────┼─────────┼─────────────┼────────────┼────────┤
│ 02-09       │ claude-haiku-4-5  │ 15,985 │     357 │     281,947 │  2,710,664 │  $0.85 │
│             │ claude-opus-4-6   │    910 │  85,080 │   1,470,650 │ 27,278,467 │ $30.48 │
│             │ claude-sonnet-4-5 │  2,877 │   7,407 │     272,110 │  1,693,376 │  $2.26 │
├─────────────┼───────────────────┼────────┼─────────┼─────────────┼────────────┼────────┤
│ 02-14       │ claude-opus-4-6   │ 24,090 │  39,682 │     568,164 │ 22,893,508 │ $18.24 │
├─────────────┼───────────────────┼────────┼─────────┼─────────────┼────────────┼────────┤
│ TOTAL       │                   │ 59,749 │ 222,326 │   4,151,456 │ 72,295,839 │ $70.87 │
╰─────────────┴───────────────────┴────────┴─────────┴─────────────┴────────────┴────────╯
```

## Flags

| Flag | Description |
|------|-------------|
| `-since YYYY-MM-DD` | Start date filter |
| `-until YYYY-MM-DD` | End date filter (inclusive) |
| `-project NAME` | Filter by project name (substring match) |
| `-by-project` | Group by project instead of date |
| `-models` | Show per-model token breakdown |
| `-json` | Output as JSON |

## How it works

`ccost` reads Claude Code's JSONL session logs from `~/.claude/projects/`, including subagent files. It:

1. Parses all `assistant` type entries with token usage
2. Deduplicates by message ID (keeps max `output_tokens` from streaming)
3. Normalizes model names (strips date suffixes like `-20250929`)
4. Calculates costs using [Anthropic's pricing](https://www.anthropic.com/pricing) with 1-hour ephemeral cache rates
5. Skips synthetic zero-token entries

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
