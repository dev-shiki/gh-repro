# gh-repro

**Deterministic 1-line local reproducers from failing GitHub Actions runs — without Docker.**

`gh-repro` reads a failing GitHub Actions run (`gh run view --log-failed`) alongside your `.github/workflows/*.yml`, strips timestamps/ANSI/group noise, redacts accidental tokens, and outputs the exact minimal local command (`pytest`, `go test`, `vitest`, `jest`, `cargo test`) with the failing step's `working-directory` and `env`.

## Why `gh-repro`?

- **`nektos/act`** spins up Docker to re-run entire workflows (slow, heavy, and often breaks on GitHub secrets or hosted services).
- **`gh run view --log-failed`** dumps thousands of raw log lines mixed with UTC timestamps and ANSI codes.
- **`gh-repro`** gives you the exact failing test node and `cwd`/`env` in < 1 second — both for humans (`gh repro`) and coding agents (`gh repro --mcp`).

## Install

```bash
gh extension install dev-shiki/gh-repro
```

Or build from source (Go 1.24+):

```bash
go install github.com/dev-shiki/gh-repro@latest
```

## Usage

```bash
# Inspect the latest failed workflow run in the current repo
gh repro

# Inspect failing checks on PR #123
gh repro --pr 123

# Inspect a specific run ID and emit JSON
gh repro --run-id 123456789 --json

# Offline mode from a saved `gh run view --log-failed` file
gh repro --log-file failed.log

# Execute the extracted minimal repro command locally (never runs without --run)
gh repro --pr 123 --run
```

Example output:

```text
[1] test-python / Run pytest (pytest)
    cwd: backend
    env: map[APP_ENV:test DATABASE_URL:sqlite:///:memory:]
    cmd: pytest tests/test_auth.py::TestToken::test_refresh
    err: FAILED tests/test_auth.py::TestToken::test_refresh - AssertionError: expired token [REDACTED]
```

## MCP Server Mode (Claude Code / Cursor / Windsurf / Antigravity)

The same single binary exposes a zero-dependency stdio MCP server (`reproduce_ci_failure`, read-only):

```json
{
  "mcpServers": {
    "gh-repro": {
      "command": "gh",
      "args": ["repro", "--mcp"]
    }
  }
}
```

## Security Design

- **Safe by default:** Never executes extracted commands unless `--run` is explicitly passed.
- **No shell injection:** `--run` invokes `exec.Command(bin, args...)` directly without `sh -c` interpolation.
- **Secret redaction:** Masks `ghp_`, `github_pat_`, `AKIA`, `sk-`, and `Bearer` tokens in extracted error tails.
- **Safe YAML:** Uses `gopkg.in/yaml.v3` structured decoding on `.github/workflows/*.yml`.
