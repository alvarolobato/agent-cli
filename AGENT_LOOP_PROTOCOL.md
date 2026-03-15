# Agent loop protocol

<!-- Based on: https://github.com/simianhacker/ralph-loop/blob/df320880e4c1557fcb50b702160eaf6bd7f15358/AGENT_LOOP_PROTOCOL.md -->

You are executing a spec in an iterative, multi-session loop.

## Required behavior

- Pick the **first unchecked** task(s) in `## Tasks`.
- Implement **up to {{TASKS_PER_TURN}} tasks** per turn (including their acceptance checks).
- **Before marking tasks complete**: run all tests, linting, type-checking, and formatting commands for the repo. Fix any failures before proceeding. If the spec includes a dedicated "run checks" task, defer to that task instead.
- Update the spec (the GitHub issue body):
  - Mark completed tasks as done (`[x]`).
  - Append discoveries/gotchas to `## Additional Context`.
  - Adjust remaining tasks if reality differs (split/merge/reword as needed).
  - Update `## Status`:
    - `in-progress` when the first implementation task begins
    - `done` only when the spec's "Definition of done" is met
- Exit after updating the spec so the next fresh session can continue.

If an `## Issue Comments` section is present below the spec, read it for guidance from the user. These comments are steering input posted during execution and may override or refine specific tasks.
If the issue references a pull request (in tasks, context, or comments), proactively inspect that PR's
review comments/threads every session and incorporate unresolved feedback before declaring handoff-ready.

## Spec source: GitHub Issues only

Specs live in GitHub issues, NOT local files. The prompt contains `<!-- Issue #N -->` and `<!-- URL: ... -->`.

- Update the issue body using `gh issue edit`:
  ```bash
  gh issue edit <number> --repo <owner/repo> --body-file <temp-file>
  ```
- To update: read current body, modify it, write to temp file, then edit.
- Extract owner/repo from the `<!-- URL: https://github.com/<owner>/<repo>/issues/<N> -->` comment.

## One-shot execution preference

Prefer completing **all tasks in a single session** rather than one-task-per-session. Implement as many tasks as possible, run checks, commit, and create the PR — all in one shot. Only exit early if you hit a blocker that requires human input.

## Backpressure

Before marking tasks complete, run the project's validation commands. Refer to `AGENTS.md` for the specific test/build/lint commands for each component. If the spec includes a dedicated "run checks" task, defer to it instead.

## Linked PR handling (required)

When a spec has a linked PR:

1. Proactively inspect review feedback every session, even if the user did not explicitly ask:
   ```bash
   gh pr view <number> --repo <owner/repo> --json reviews,comments,statusCheckRollup
   gh api repos/<owner>/<repo>/pulls/<number>/comments
   ```
2. Treat unresolved review threads/comments as in-scope work. Add tasks if needed, implement fixes,
   push updates, and reply/resolve threads when possible.
3. **Never merge PRs as the agent.** Once checks pass and feedback is addressed, post a handoff
   comment requesting user attention and set the issue to `needs-attention`.

## Asking questions

When you need clarification before proceeding:

1. Post a comment on the issue:
   ```
   ## Agent says...

   <Your question or clarification request>

   **Options:** (if applicable)
   1. Option A - description
   2. Option B - description

   ---
   Blocked on: Task N (task name)
   ```

2. Update the issue body `## Status` to `agent-question`.

3. Update labels:
   ```bash
   gh issue edit <number> --repo <owner/repo> --remove-label "status:in-progress" --add-label "status:agent-question"
   ```

4. **Exit the session.** Do NOT inject follow-up tasks or poll for replies. The user will reply, remove the blocking label, and re-trigger the agent when ready. This saves tokens and avoids unnecessary roundtrips.

## Adding tasks mid-execution

When you discover necessary work not in the original spec:

1. Insert new task(s) in the appropriate position in `## Tasks`.
2. Add the `agent-added-tasks` label for audit trail:
   ```bash
   gh issue edit <number> --repo <owner/repo> --add-label "agent-added-tasks"
   ```
3. Post a comment explaining what was added and why.
4. Continue execution.

## Stopping rules

- **`needs-attention`**: Set when all agent tasks are done and human action is required (for example, PR ready for human merge). Update **both** the GitHub label and the `## Status` line in the body, and post a comment asking for user attention. The agent does NOT re-check the issue — the user removes `needs-attention` and re-triggers the agent if further work is needed.
- **`agent-question`**: Set when blocked on a question (see above). Same rule — do not poll or re-check.
- **No-op exit**: If the issue already has `needs-attention` or `agent-question` in both label and body, and there are no new comments in `## Issue Comments`, exit immediately without editing.

## Status workflow

```
draft → ready → in-progress → needs-attention → done
                     ↓
               agent-question
```

| Transition | Who | Trigger |
|-----------|-----|---------|
| `draft` → `ready` | Human only | Human approves the spec |
| `ready` → `in-progress` | Agent | Agent starts first task |
| `in-progress` → `needs-attention` | Agent | All tasks done; PR open; waiting for merge |
| `in-progress` → `agent-question` | Agent | Blocked on a question |
| `agent-question` → `in-progress` | Human triggers, agent resumes | Human replies and removes blocking label |
| `needs-attention` → `done` | Human | PR merged; issue auto-closes |

## Labels

| Label | Purpose |
|-------|---------|
| `spec` | Standalone spec issue |
| `project` | Orchestrator issue (multiple specs) |
| `project:spec` | Spec belonging to a project |
| `status:draft` | Agent-created, being refined |
| `status:ready` | Human-approved, ready for execution. **Only a human sets this.** |
| `status:in-progress` | Agent is implementing |
| `status:needs-attention` | Agent finished or blocked; needs human action |
| `status:agent-question` | Agent blocked, waiting for clarification |
| `status:done` | Completed; PR merged |
| `agent-added-tasks` | Audit trail: agent added tasks during execution |

## Creating a new spec issue

When drafting a new spec:

1. Set `## Status` to **draft**.
2. Apply labels: `spec` (or `project:spec`), `status:draft`, plus appropriate `component:*` and `type:*`.
3. **Never** apply `ready` or `status:ready`. Only a human promotes a spec.

## Spec issue template

```markdown
# <Feature name>

## Status
draft | ready | in-progress | needs-attention | done

## Context
- **Problem**: <what's wrong / missing; why it matters>
- **Worktree**: <required: worktree name for isolated execution, e.g. my-feature>
- **Scope**: <what is in / out>
- **Constraints**: <perf, compatibility, deps, no-breaking-changes, etc.>
- **Repo touchpoints**: <files/dirs likely involved>
- **Definition of done**: <builds + tests pass; plus feature-specific checks. Must include a PR that is ready for human merge.>

## Tasks
- [ ] 1) <task> (owner: agent)
  - **Change**: <precise behavior/code change>
  - **Files**: <exact paths>
  - **Acceptance**: <how to verify; exact commands/output>

- [ ] N-2) Run all checks and fix issues (owner: agent)
  - **Change**: Run all tests, linting, type-checking, and formatting; fix any failures
  - **Acceptance**: All repo checks pass

- [ ] N-1) Create commit (owner: agent)
  - **Change**: Stage all changes and create a descriptive commit
  - **Acceptance**: `git status` shows clean working tree

- [ ] N) Create or update pull request (owner: agent)
  - **Change**: Push branch, open/update PR targeting default branch. Reference spec issue (`Closes #<number>`)
  - **Acceptance**: PR exists, references issue, targets default branch

## Additional Context
<append-only notes>
```
