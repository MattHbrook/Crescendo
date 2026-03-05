# Crescendo — Working Agreement

## What Is This

Music discovery & download app (Go). Bridges hifi-api (Tidal proxy) with a local FLAC library. Mobile-optimized web UI.

## Quick Reference

```
make all        # fmt + lint + test + sec + build (full quality gate)
make build      # compile to build/crescendo
make test       # go test -race -count=1 ./...
make lint       # golangci-lint run
make sec        # govulncheck ./...
make fmt        # gofmt + goimports
make docker     # local Docker image build
```

## Git Workflow

- `main` is always deployable
- Branch naming: `<type>/<short-description>` — feat/, fix/, refactor/, docs/, test/, chore/, ci/
- Commit format: `<type>: <imperative summary>` (lowercase, no period)
- Claude never commits or pushes without being explicitly asked
- One logical change per commit
- Feature work on branches → Claude opens PRs via `gh` CLI → human reviews and merges

## Quality Gates

All must pass before merge:

| Gate     | Command     | Checks                                         |
|----------|-------------|-------------------------------------------------|
| Format   | `make fmt`  | gofmt + goimports                               |
| Lint     | `make lint` | golangci-lint (staticcheck, errcheck, gosec...) |
| Test     | `make test` | All tests pass with `-race`                     |
| Security | `make sec`  | govulncheck — no known CVEs                     |
| Build    | `make build`| Compiles cleanly                                |

## Code Standards

- Always check errors (no `_ = err`)
- Wrap errors with context: `fmt.Errorf("doing thing: %w", err)`
- Table-driven tests with `t.Run()` subtests, stdlib `testing` only
- Integration tests use `//go:build integration` tag
- Comments explain WHY, not WHAT
- No dead code, no commented-out code, no TODO without issue number

## Dependency Policy

- Stdlib first
- Approved: chi, modernc.org/sqlite, godotenv
- New deps require discussion and justification
- Run `govulncheck` before merging any `go.mod` change

## How Claude Works

- Read before writing. Follow existing patterns.
- Run `make all` after changes. Fix what fails.
- One feature/fix per branch. Flag security issues immediately.
- Admit uncertainty rather than guessing.

## Context Window Management

- Orchestrator stays high-level: planning, decisions, wiring, reviewing results
- Delegate to subagents for implementation, research, and verbose output
- Launch independent agents in parallel to maximize throughput
- Never duplicate work between orchestrator and agent

## Project Layout

```
cmd/crescendo/main.go       — entry point
internal/                    — all app packages
  config/                    — env-based configuration
  db/                        — SQLite connection, migrations, queries
  hifi/                      — HTTP client for hifi-api
  library/                   — scanner, path building, index queries
  manifest/                  — base64 decode → BTS JSON or DASH MPD
  downloader/                — download workers, queue, FLAC tagging
  discovery/                 — recommendations from library + Tidal
  handlers/                  — HTTP handlers (home, search, artist, album, etc.)
templates/                   — HTML templates (HTMX + Pico CSS)
static/                      — CSS overrides
migrations/                  — SQL migration files
```

## Key Architecture

- Single Go binary, Chi router, HTMX + Pico CSS, SQLite (modernc.org/sqlite)
- crescendo container (:8888) → hifi-api container (:8000)
- /music volume (bind mount to NAS) + /data volume (Docker managed, SQLite)
- ffmpeg for HI_RES_LOSSLESS DASH remuxing (Alpine image)
