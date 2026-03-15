# agent-cli Decision Log

This log records architectural and technical decisions for traceability.
Update this file in every PR that introduces or changes meaningful project decisions.

## Decision Table

| ID | Date | Status | Decision | Rationale | Source |
|---|---|---|---|---|---|
| D-001 | 2026-03-15 | accepted | Use Go as implementation language | Aligns with Elastic Agent and OTel collector ecosystem; enables direct package reuse and static binaries | `project-requirements.md` |
| D-002 | 2026-03-15 | accepted | Use Cobra for CLI command tree | Widely adopted standard for Go CLIs and supports command/flag growth cleanly | `project-requirements.md`, PR `#10` |
| D-003 | 2026-03-15 | accepted | Use Bubbletea + Charm stack for TUI | Composable/testable model with mature terminal UX ecosystem | `project-requirements.md`, PR `#10` |
| D-004 | 2026-03-15 | accepted | Adopt spec-driven issue workflow with agent loop protocol | Ensures explicit status/task tracking and repeatable autonomous execution | PR `#11`, `AGENT_LOOP_PROTOCOL.md` |
| D-005 | 2026-03-15 | accepted | Keep architecture as layered adapter model | Isolates agent-specific behavior behind shared pipeline interfaces and enables phased agent support | `project-requirements.md`, PR `#10` |
| D-006 | 2026-03-15 | accepted | Prioritize local-host access in Phase 1 | Reduces auth/network complexity and accelerates delivery of status visibility | `project-requirements.md` |
| D-007 | 2026-03-15 | accepted | Implement Elastic Agent support before EDOT/OTel | Matches product priority and provides baseline abstractions for later adapters | PR `#12`, issues `#4`, `#5` |
| D-008 | 2026-03-15 | accepted | Parse OTel/EDOT config using typed collector model | Needed to map `service.pipelines` accurately and detect component wiring reliably | PR `#13`, issue `#5` |
| D-009 | 2026-03-15 | accepted | Preserve OTel processor order from config when building edges | Processor order changes behavior; alphabetical sorting can produce incorrect runtime representation | PR `#13` follow-up commit `369a06a` |
| D-010 | 2026-03-15 | accepted | Cache dashboard OTel-kind detection in model construction | Avoids duplicate node scans and keeps UI logic consistent across view paths | PR `#13` follow-up commit `369a06a` |
| D-011 | 2026-03-15 | accepted | Parse Prometheus metric sample from token index 1 (not trailing token) | Prometheus lines may include optional timestamps; parsing last token can misread timestamps as values | PR `#13` review follow-up |
| D-012 | 2026-03-15 | accepted | Treat zpages HTML parsing as limited fallback and prefer JSON | Real pipelinez HTML format is not stable for regex scraping; JSON path is more robust and testable | PR `#13` review follow-up |
| D-013 | 2026-03-15 | accepted | Require architecture and decision docs updates in ongoing PRs | Keeps long-running project context current as implementation evolves across phases | issue `#5` execution comment trail |

## Notes

- IDs are append-only; do not reuse or reorder historical IDs.
- If a decision is superseded, mark prior row as `superseded` and add a new row with the replacement.
- Keep rationale short and technical; link to PRs/issues/files that contain implementation evidence.
