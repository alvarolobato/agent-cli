# AGENTS.md — AI development guide

Guidance for AI assistants. Use the **skills** ([docs/skills/skills.md](docs/skills/skills.md)) for domain detail; this file is the skeleton, index, and meta-rules.

## Project Overview

**agent-cli** is a CLI & TUI tool for managing and inspecting the configuration of Elastic Agent, EDOT Collectors, and generic OpenTelemetry Collectors.

**Status:** Greenfield Go project — no existing tooling to migrate from.

Key capabilities:
1. **Inspect** the current state of an agent's pipeline (inputs → processors/transforms → outputs) at a glance.
2. **Assess health** — healthy / degraded / error state with reason per component.
3. **View metrics** — throughput, error rates, buffer pressure per pipeline stage.
4. **Surface misconfigurations** — disabled components, config errors, and warnings.
5. **Rich TUI** — interactive terminal UI (Phase 1) and **Config modification** (Phase 2).

**Full requirements:** [project-requirements.md](project-requirements.md)
**Implemented architecture:** [architecture.md](architecture.md)
**Decision history:** [decision-log.md](decision-log.md)

---

## Repository Structure

```
agent-cli/
├── cmd/agent-cli/          # Entry point (main.go)
├── internal/
│   ├── cli/                # Cobra command definitions
│   ├── tui/                # Bubbletea models & views
│   ├── agent/              # Agent abstraction layer + adapters (EA, EDOT, OTel)
│   ├── discovery/          # Agent discovery logic
│   ├── pipeline/           # Pipeline model (DAG), health, render
│   ├── config/             # Config parsing per agent type
│   ├── metrics/            # Metrics collection & aggregation
│   └── output/             # Output formatters (table, JSON, pipeline diagram)
├── pkg/agentcli/           # Public library (for hybrid/embedded use)
├── test/
│   ├── integration/        # Integration test suites + docker-compose.yml
│   ├── fixtures/           # Sample configs & golden files
│   └── mocks/              # Mock HTTP servers for unit tests
├── .github/workflows/      # CI (ci.yml) + release (release.yml)
├── docs/skills/            # AI agent skill documents
├── specs/                  # Spec index (issues are source of truth)
└── project-requirements.md # Full requirements document
```

---

## Technology Stack

| Area | Choice | Rationale |
|------|--------|-----------|
| Language | Go | Same as EA, OTel Collector — can import their packages directly |
| CLI | Cobra | Industry standard; used by EA and OTel Collector |
| TUI | Bubbletea + Lipgloss + Bubbles | Modern, testable, composable Charm ecosystem |
| Forms (Phase 2) | huh | Charm form/wizard framework |
| Testing | Standard `testing` + table-driven tests | Go idiomatic |
| CI | GitHub Actions + GoReleaser | Standard Go release pipeline |

---

## Development Commands

```bash
go build ./...                   # Build all packages
go test ./...                    # Run all unit tests
go test -race ./...              # Run tests with race detector
go vet ./...                     # Static analysis
golangci-lint run                # Linting (install via brew or binary)
make build                       # Build binary (once Makefile exists)
make test                        # Run tests
make lint                        # Run linter
```

**Integration tests** (requires Docker):
```bash
cd test/integration && docker compose up -d
go test ./test/integration/...
```

---

## Testing Strategy

### Unit Tests
- Mock HTTP servers simulating Elastic Agent status API, OTel zpages, Prometheus endpoints.
- Test all adapters against mocked responses for each agent type.
- Test discovery logic with mocked process lists and file system.
- Test health assessment logic with various metric scenarios.
- Test output formatters (JSON, table, pipeline diagram).
- Test TUI models in isolation using Bubbletea's test utilities.

### Integration Tests (CI)
- Docker Compose spins up real agent instances (EA 9.x standalone, EDOT, vanilla OTel contrib).
- `agent-cli` runs against live agents.
- Golden file comparisons for expected status output.

### CI Pipeline
```
lint → unit tests → build → integration tests (Docker) → release (on tag)
```

### Verify before declaring done (backpressure)

| Component | Validation command |
|-----------|-------------------|
| All Go code | `go build ./...` + `go test ./...` + `go vet ./...` |
| Linting | `golangci-lint run` |
| Integration tests | `cd test/integration && docker compose up -d && go test ./test/integration/...` |
| Binary works | `./agent-cli --help` |

---

## Spec-driven development

This project uses **spec-driven development**. Feature work is planned and tracked as **GitHub issues** (specs). The agent loop protocol lives in [`AGENT_LOOP_PROTOCOL.md`](AGENT_LOOP_PROTOCOL.md) at the repo root.

Key points:
- **Specs are GitHub issues** with structured context, tasks, and acceptance criteria.
- **Workflow**: `draft` → `ready` (human-approved) → `in-progress` → `needs-attention` → `done` (PR merged).
- **PR ownership**: agents open/update PRs and address feedback; **users merge PRs**.
- **One-shot execution**: complete all tasks, run checks, commit, and create the PR in a single session.
- **Human-gated**: agents never promote a spec to `ready`; only humans do.

---

## Important Rules for AI Assistants

### Working on issues
When asked to work on an issue: review the issue and all comments; if incomplete, update the description; implement in a separate worktree; verify the fix/feature and gather evidence; create a PR with details; wait until the PR builds green (fix if not); resolve merge conflicts; and proactively check linked PR review comments for unresolved feedback. The PR must reference the issue for auto-close. **Do not merge the PR as the agent** — hand off with a user-attention comment and `status:needs-attention`.

### Working on specs
When asked to work on a **spec issue** (label `spec` or `project:spec`), follow [`AGENT_LOOP_PROTOCOL.md`](AGENT_LOOP_PROTOCOL.md): implement all tasks with backpressure, update the issue body, commit, and create/update the PR — all in one shot. When blocked, set `agent-question` label, post a comment, and exit.

### Working on PR reviews
When asked to address PR feedback: fetch the PR reviews and all conversation threads. For each **unresolved** comment, triage and either implement or justify skipping. After processing all feedback, leave a **general PR comment**: "PR feedback addressed." Commit and push.

### Documentation Updates Required

| Change Type | Update Required |
|-------------|-----------------|
| Add/remove package or major component | AGENTS.md (structure), README.md |
| Architecture, data-flow, or technical boundary changes | architecture.md, decision-log.md, AGENTS.md (if guidance changes) |
| Material technical decision (new pattern, tradeoff, limitation, fallback) | decision-log.md |
| New CLI command | AGENTS.md (commands), internal/cli/ |
| New agent adapter | AGENTS.md (supported agents), pkg/agentcli/ public API |
| New TUI screen | AGENTS.md (TUI screens) |
| CI/CD changes | AGENTS.md (CI pipeline section) |
| New skill area | docs/skills/skills.md |
| Spec-driven development / agent protocol | AGENT_LOOP_PROTOCOL.md, docs/skills/agent-loop-protocol.md, AGENTS.md |

`architecture.md` and `decision-log.md` are living documents and must be updated in every PR that changes architecture or technical decisions.

---

## Self-learning and documentation

When you fix a non-obvious bug or discover a gotcha, document it and update cross-references. Procedure: [agent-efficiency.md](docs/skills/agent-efficiency.md).

---

## AI Assistant Configuration

This project supports **Cursor**, **GitHub Copilot**, **Claude Code**, and **OpenCode**. All follow the same guideline:

- **Entry point:** AGENTS.md (this file) for skeleton, index, and meta-rules.
- **Domain detail:** [docs/skills/skills.md](docs/skills/skills.md) to choose the right skill.
- **Self-learning:** [docs/skills/agent-efficiency.md](docs/skills/agent-efficiency.md).

### Configuration files (minimal pointers only)

| File | Editor | Purpose |
|------|--------|---------|
| `CLAUDE.md` | Claude Code | Imports AGENTS.md + skills |

---

## GitHub access

Use the [GitHub CLI](https://cli.github.com/) (`gh`) for all GitHub operations.

**Cloud sessions:** Pass `-R owner/repo` on every `gh` command. Authentication via `GH_TOKEN` env var.

```bash
gh issue view <number> -R alvarolobato/agent-cli
gh issue create -R alvarolobato/agent-cli --title "..." --body "..."
gh pr create -R alvarolobato/agent-cli --title "..." --body "..."
```

### Issue and PR labeling policy

- **Required labels:** at least one `component:*` and one `type:*`; add `status:*` for specs/workflows.
- **Quick check:** `gh issue list -R alvarolobato/agent-cli --search "is:open no:label"`
