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
ccost                                        # daily breakdown
ccost -s 2026-02-01 -u 2026-02-07            # filter by date range
ccost -p myapp                               # filter by project
ccost --by-project                           # group by project
ccost -m                                     # per-model breakdown
ccost --by-project -m -s 2026-02-01          # combine flags
ccost --json                                 # JSON output
ccost -e                                     # exact token counts (no K/M)
```

## License

MIT
