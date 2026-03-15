# Agent-efficiency skill

**Lightweight.** Apply only when it clearly fits.

## Purpose

Improve future sessions by recording when **lack of a skill or guidance in AGENTS.md** made the agent work harder than necessary. Not every session should produce an issue — only when there is an **obvious need** and the agent had to **do considerable figuring out** to complete the task.

## When to create an issue

- During or at the end of a session, note where you struggled (e.g. unclear docs, no skill for a domain, scattered info).
- If the gap is clear and fixing it would help future agents (e.g. a new skill or an AGENTS.md section), **create one GitHub issue** with label **`agent-efficiency`** describing the improvement.

**Do not** create an issue every session. **Do** create one when:
- The task touched a domain that has no skill or no clear guidance (e.g. “how the OTel zpages API works”, “how to write Bubbletea model tests”).
- You had to search the codebase, guess, or infer a lot to do something that a short skill or AGENTS.md update could have made straightforward.

## Issue format

- **Title:** Short improvement (e.g. “Skill: OTel zpages API reference” or “AGENTS.md: document golden file testing conventions”).
- **Body:** What was missing, what you had to figure out, and what to add (new skill in `docs/skills/`, new section in AGENTS.md, or link to existing doc). Keep it concise.
- **Label:** `agent-efficiency`

```bash
gh issue create --title "Skill: ..." --body "..." --label "agent-efficiency"
```

If the label does not exist yet: `gh label create "agent-efficiency" --description "Improvements for agent efficiency (skills, AGENTS.md)"`.

## Examples of improvements

- **New skill:** “OTel zpages API — endpoints, response format, how to parse pipeline topology.”
- **New skill:** “Elastic Agent gRPC control protocol — how to connect and query component status.”
- **AGENTS.md:** Add or expand a section (e.g. golden file testing conventions, TUI model patterns) so the next agent doesn’t have to rediscover it.

Stay lightweight: one issue per clear gap; no issue when the session was smooth or the gap is minor.

---

## Self-learning and documentation (do this)

When you solve a non-obvious problem or discover a gotcha:

1. **Capture the problem** briefly (what failed, error or behavior).
2. **Document the solution** in the right place:
   - Repo layout, adapters, CLI, TUI, pipeline model → the relevant package or `AGENTS.md`
   - AI workflow, testing, rules, skills → [AGENTS.md](../../AGENTS.md) or the relevant skill in `docs/skills/`
   - User-facing setup or runbooks → README.md
3. **Update cross-references** so the next agent or maintainer can find it (e.g. add a row to "Documentation Updates Required" in AGENTS.md, or add a troubleshooting entry in the right skill).
4. **If the gap was missing guidance** (no skill, unclear doc) → create an **agent-efficiency** issue (see above) so the fix is tracked.

Apply this every time you fix something non-obvious — do not skip. It keeps the codebase and docs from drifting.

