# Skills in this folder

This folder contains **skill documents** for AI agents working on the agent-cli project. Each skill is a single reference for a specific domain. **Read this file to see what skills exist and when to use them.**

## Project Skills

| Skill | Purpose | Use when |
|-------|---------|----------|
| **[project-development.md](project-development.md)** | How to build, test, lint, and run agent-cli. | Running `go test`, `go build`, `golangci-lint`, integration tests — the day-to-day Go workflow. |

## Meta Skills

| Skill | Purpose | Use when |
|-------|---------|----------|
| **[agent-loop-protocol.md](agent-loop-protocol.md)** | How the `AGENT_LOOP_PROTOCOL.md` override works, what we customize from upstream ralph-loop, and how to maintain it. | Updating the agent protocol, syncing with upstream, understanding why we override specific behaviors. |
| **[agent-efficiency.md](agent-efficiency.md)** | Self-learning and documentation (where to document gotchas, update cross-refs); create `agent-efficiency` issues when guidance is missing. | After fixing non-obvious bugs or discovering gotchas; when a clear doc/skill gap appears. |
| **[gh-attach.md](gh-attach.md)** | **Uploading images to GitHub issues and pull requests** from the CLI (via gh-attach + Playwright). | Attaching screenshots or diagrams to an issue or PR comment. |

## Summary

- **Spec-driven development**: Protocol lives in `AGENT_LOOP_PROTOCOL.md` (repo root). Use **agent-loop-protocol** to understand the override mechanism and how to maintain it.
- **Go development**: Use **project-development** for the standard Go build/test/lint workflow.
- **PR/issue images**: Use **gh-attach** to upload screenshots to GitHub.
