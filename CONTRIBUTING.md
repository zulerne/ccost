# Contributing to ccost

Thanks for your interest in contributing!

## Quick start

```bash
git clone https://github.com/zulerne/ccost.git
cd ccost
task check   # runs lint + tests
```

## Development

Requires **Go 1.26+** and [Task](https://taskfile.dev).

| Command          | Description              |
|------------------|--------------------------|
| `task build`     | Build the binary         |
| `task test`      | Run tests with `-race`   |
| `task lint`      | Run golangci-lint        |
| `task check`     | Lint + test              |
| `task coverage`  | Tests with coverage      |

## Guidelines

- Run `task check` before pushing.
- Use [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `docs:`, `chore:`, etc.
- Keep PRs focused — one concern per PR.
- Add tests for new functionality.

## Reporting bugs

Open an [issue](https://github.com/zulerne/ccost/issues/new/choose) using the bug report template.
