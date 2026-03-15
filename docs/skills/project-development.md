# Skill: Project Development

**Use when**: Working on any agent-cli component and needing to build, test, lint, or run the project.

## Quick Reference

### Building

```bash
go build ./...                   # Build all packages
go build -o agent-cli ./cmd/agent-cli/  # Build binary
make build                       # Build via Makefile (once it exists)
```

### Testing

```bash
go test ./...                    # Run all unit tests
go test -race ./...              # Run with race detector (preferred in CI)
go test -v ./internal/...        # Verbose unit tests
go test -run TestFoo ./...       # Run specific test
```

### Linting

```bash
go vet ./...                     # Static analysis (always run)
golangci-lint run                # Full lint suite
golangci-lint run ./internal/... # Lint specific packages
```

Install golangci-lint: `brew install golangci-lint` or download binary from https://golangci-lint.run/

### Integration Tests

Requires Docker and Docker Compose.

```bash
cd test/integration
docker compose up -d             # Spin up real agents (EA, EDOT, OTel)
go test ./test/integration/...   # Run integration tests
docker compose down              # Teardown
```

### Running the binary

```bash
go run ./cmd/agent-cli -- --help            # Run without building
./agent-cli --help                          # After build
./agent-cli discover                        # Discover local agents
./agent-cli status                          # Status of all discovered agents
./agent-cli status --agent elastic-agent    # Target specific agent type
./agent-cli status --format json            # JSON output
./agent-cli tui                             # Launch interactive TUI
```

---

## Verify before declaring done (backpressure)

Always test your changes — do not assume code works.

| Check | Command |
|-------|---------|
| Build succeeds | `go build ./...` |
| All unit tests pass | `go test -race ./...` |
| No vet issues | `go vet ./...` |
| No lint issues | `golangci-lint run` |
| Binary works | `go run ./cmd/agent-cli -- --help` |
| Integration tests (if adapter/discovery changed) | `cd test/integration && docker compose up -d && go test ./test/integration/...` |

---

## Project Conventions

### Package structure
- `internal/` — private packages; not importable by external consumers
- `pkg/agentcli/` — public API; stable, well-designed; can be embedded by Elastic Agent or EDOT
- `cmd/agent-cli/` — main entry point only; thin wrapper around `internal/cli`

### Agent adapters
Each adapter in `internal/agent/` implements the `agent.Agent` interface defined in `internal/agent/agent.go`. New agent types get their own subdirectory (`elasticagent/`, `edot/`, `otel/`).

### TUI models
Follow the Bubbletea Elm architecture: each screen is a `Model` with `Init()`, `Update(msg)`, and `View()` methods. Use `tea.TestModel` for unit testing TUI models without a real terminal.

### Output formatters
All output formatters in `internal/output/` take a `*pipeline.Pipeline` and return a string. They must not write directly to `os.Stdout` — callers handle that.

### Testing patterns
- **Table-driven tests** for unit tests (Go idiomatic).
- **Mock HTTP servers** (`test/mocks/`) simulate agent APIs for adapter unit tests.
- **Golden files** (`test/fixtures/golden/`) for expected CLI output comparisons.

---

## Agent: Testing when terminal is restricted

When the runner rejects terminal commands:

1. **Try normal flow first** — `go build ./...` and `go test ./...`. If these succeed, verification is done.
2. **If restricted** — note in the PR: "Tests could not be run by the agent (terminal restricted). Please run `go test -race ./...` and `golangci-lint run` from repo root to verify."
3. **PR note required** — always state in the PR description whether tests were run or not.
