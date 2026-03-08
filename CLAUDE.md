# ccost

Go CLI for analyzing Claude Code token usage and costs.

## Philosophy

Purposefully minimalist — do one thing well with no extra machinery.

- **Single-purpose offline tool** — reads local JSONL logs, no network, no config files, no persistent state
- **Minimal dependencies** — only pflag (flags) and go-pretty (tables); no frameworks, no DI, no interfaces where unnecessary
- **Flat architecture** — linear data flow: parse → aggregate → display; no subcommands, no plugins
- **Composable flags** — orthogonal flags that combine freely; sensible defaults (last 7 days)
- **Practical, not generic** — solves real problems (dedup, project disambiguation, parallel parsing) without abstractions for hypothetical needs
- **Graceful degradation** — unknown models → warning, malformed lines → skip, no records → clean exit

## Updating pricing

When model prices change, update `internal/pricing/pricing.go`:

1. Fetch current prices from https://platform.claude.com/docs/en/about-claude/pricing
2. Update the `models` map with new values. Pricing rules:
   - **Cache Write** = 2x Input price (Claude Code uses 1-hour ephemeral cache)
   - **Cache Read** = 0.1x Input price
3. Add entries for any new models
4. Update tests in `internal/pricing/pricing_test.go`
5. Run `go test ./...` to verify

## Releasing

Release is automated via GoReleaser + GitHub Actions (`.goreleaser.yaml`, `.github/workflows/release.yml`).

Follow [semver](https://semver.org/): bump **patch** for bug fixes (`v0.8.0` → `v0.8.1`), **minor** for new features (`v0.8.1` → `v0.9.0`), **major** for breaking changes.

To publish a new version:

```bash
git tag v0.X.Y
git push origin v0.X.Y
```

This triggers the `Release` workflow which:
1. Builds binaries for linux/darwin × amd64/arm64 (CGO_ENABLED=0)
2. Injects version via `-X main.version={{.Version}}`
3. Creates a GitHub Release with archives and checksums
4. Updates the Homebrew formula in `zulerne/homebrew-tap`

Required repository secrets: `GITHUB_TOKEN` (automatic), `HOMEBREW_TAP_GITHUB_TOKEN` (PAT with repo access to homebrew-tap).

## Project structure

- `cmd/ccost/main.go` — CLI entry point, flags, wiring
- `internal/pricing/` — model price table, cost calculation
- `internal/parser/` — JSONL file discovery (sessions + subagents), parsing, deduplication
- `internal/report/` — aggregation by date/project, with optional model detail
- `internal/display/` — table (go-pretty) and JSON output

## Code Quality

Always run `task lint` before committing. Run `task check` (lint + test) for full validation.

## Workflow

- Search for existing solutions (stdlib, well-maintained packages) before writing custom code
- Compact context at logical boundaries: after planning, after debugging — not mid-implementation
