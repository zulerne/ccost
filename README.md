<picture>
  <source media="(prefers-color-scheme: dark)" srcset="ccost-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="ccost-light.svg">
  <img alt="ccost" src="ccost-dark.svg" width="480">
</picture>

[![CI](https://github.com/zulerne/ccost/actions/workflows/ci.yml/badge.svg)](https://github.com/zulerne/ccost/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/zulerne/ccost/graph/badge.svg?token=KMKEA9GGBO)](https://codecov.io/gh/zulerne/ccost)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Go Report Card](https://goreportcard.com/badge/github.com/zulerne/ccost)](https://goreportcard.com/report/github.com/zulerne/ccost)
[![Go Reference](https://pkg.go.dev/badge/github.com/zulerne/ccost.svg)](https://pkg.go.dev/github.com/zulerne/ccost)
[![Release](https://img.shields.io/github/v/release/zulerne/ccost)](https://github.com/zulerne/ccost/releases)

A CLI tool to analyze [Claude Code](https://claude.com/claude-code) token usage, costs, and session time.

Reads local JSONL session logs. No network access. Alternative to [ccusage](https://github.com/ryoppippi/ccusage).

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
task build
```

## Usage

```bash
ccost                                           # last 7 days (default)
ccost --since 2026-02-01 --until 2026-02-07     # custom date range
ccost --project myapp                           # filter by project
ccost --by-project                              # group by project
ccost --models                                  # per-model breakdown
ccost --by-project --models --since 2026-02-01  # combine flags
ccost --json                                    # JSON output
ccost --exact                                   # exact token counts (no K/M)
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

[MIT](LICENSE)