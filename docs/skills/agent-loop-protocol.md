# Agent loop protocol override

This project uses [simianhacker/ralph-loop](https://github.com/simianhacker/ralph-loop) for spec-driven agent execution. The `ralph-loop` runner concatenates `AGENT_LOOP_PROTOCOL.md` with the spec content to form the agent prompt each iteration.

## How the override works

The runner resolves the protocol file with this precedence:

1. **`--protocol` flag** — explicit path; highest priority
2. **`AGENT_LOOP_PROTOCOL.md` in project root** — auto-detected local override
3. **Built-in protocol** (inside `ralph-loop` package) — default fallback

Additionally, `--extra-instructions <file>` appends content to whichever base protocol is used.

**This project uses option 2**: a local `AGENT_LOOP_PROTOCOL.md` at the repo root that replaces the built-in protocol entirely. The runner logs `local (./AGENT_LOOP_PROTOCOL.md)` confirming the override.

## What we override

Our protocol diverges from upstream in these areas:

| Upstream behavior | Our override | Rationale |
|---|---|---|
| Supports file-based and issue-based specs | **Issue-based only** | GitHub issues are our single source of truth |
| One task per turn (configurable) | **One-shot: complete all tasks in a single session** | Saves tokens and roundtrips; issues are usually small enough |
| Injects "Follow up on user feedback" task when blocked | **No task injection; agent exits cleanly** | User removes the blocking label and re-triggers; avoids polling |
| `Processed Comments` section in issue body | **Removed** — uses `## Issue Comments` section injected by runner | Simpler; ralph-loop handles comment injection |
| `needs-attention` triggers re-check loop | **Agent does NOT re-check** — human removes label to re-trigger | Saves tokens |

**Important:** The `- **Worktree**: <name>` field in the spec Context section is **required** by ralph-loop. Specs without it are skipped. The worktree provides an isolated git worktree for each spec execution, preventing parallel agents from stepping on each other. This requirement comes from upstream ralph-loop and must be included in the spec template.

**Note:** The protocol is intentionally project-agnostic. Project-specific validation commands (backpressure) live in `AGENTS.md` under "Verify before declaring done", not in the protocol file. This allows the protocol to be reused across projects.

## Maintaining the override

When upstream `AGENT_LOOP_PROTOCOL.md` changes:

1. Check what changed: `gh api repos/simianhacker/ralph-loop/contents/AGENT_LOOP_PROTOCOL.md --jq '.content' | base64 -d > /tmp/upstream-protocol.md && diff AGENT_LOOP_PROTOCOL.md /tmp/upstream-protocol.md`
2. Cherry-pick improvements that apply to our workflow.
3. Ignore changes related to file-based specs or feedback-loop polling.

## Key files

| File | Purpose |
|------|---------|
| `AGENT_LOOP_PROTOCOL.md` | The protocol override (read by `ralph-loop` runner) |
| `AGENTS.md` | Project-level agent guide (loaded per-session via `CLAUDE.md`/cursor rules) |
| `docs/skills/skills.md` | Skill index — points to this doc and all domain skills |

