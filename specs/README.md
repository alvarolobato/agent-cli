# Specs

All feature specs live as **GitHub issues** in this repository. Issues are the single source of truth.

The agent loop protocol is defined in [`AGENT_LOOP_PROTOCOL.md`](../AGENT_LOOP_PROTOCOL.md) at the repo root.

## Quick reference

- **Create a spec**: File a GitHub issue using the template in `AGENT_LOOP_PROTOCOL.md`. Label it `spec` + `status:draft` + appropriate `component:*` and `type:*`. The `- **Worktree**: <name>` field in the Context section is **required**.
- **Promote to ready**: Only a human removes `status:draft` and adds `status:ready`.
- **Execute**: Agent implements all tasks in one shot, runs checks, commits, creates PR.
- **Done**: PR merged → issue auto-closes.

## Labels

| Label | Purpose |
|-------|---------|
| `spec` | Standalone spec |
| `project` | Orchestrator issue (multiple specs) |
| `project:spec` | Spec belonging to a project |
| `status:draft` / `ready` / `in-progress` / `needs-attention` / `done` | Workflow state |
| `status:agent-question` | Agent blocked, needs human input |
| `agent-added-tasks` | Agent added tasks during execution |
| `component:cli` | CLI commands (Cobra) |
| `component:tui` | TUI (Bubbletea) |
| `component:agent-adapter` | Agent adapters (EA, EDOT, OTel) |
| `component:discovery` | Agent discovery logic |
| `component:pipeline` | Pipeline model and rendering |
| `component:config` | Config parsing |
| `component:metrics` | Metrics collection |
| `component:output` | Output formatters |
| `component:ci` | CI/CD, release, tooling |
| `type:feature` | New feature |
| `type:bug` | Bug fix |
| `type:refactor` | Refactoring |
| `type:docs` | Documentation |
| `type:test` | Testing |
| `type:infra` | Infrastructure, CI, tooling |
