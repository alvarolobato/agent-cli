# agent-cli Architecture

This document describes the implemented architecture and the target direction for `agent-cli`.
It is grounded in:

- `project-requirements.md` (product definition and constraints)
- merged PRs `#10` (foundation), `#11` (agent workflow/docs), `#12` (Elastic Agent support)
- active Phase 1B work on EDOT support (PR `#13`)

## 1) System Goals

`agent-cli` provides a terminal-native way to inspect local observability agents:

- Elastic Agent
- EDOT Collector (Elastic distribution of OTel)
- Generic OTel Collector (planned; partial groundwork is in place)

Primary outcomes:

1. Pipeline-centric status visibility
2. Component health and runtime issues
3. Key throughput/error metrics
4. Human-friendly TUI + scriptable CLI outputs

## 2) High-Level Architecture

The codebase follows a layered, adapter-driven design:

1. **Presentation layer**
   - Cobra CLI commands in `internal/cli/`
   - Bubbletea TUI models/views in `internal/tui/`
2. **Domain model layer**
   - Shared pipeline DAG and health logic in `internal/pipeline/`
3. **Adapter layer**
   - Agent-specific implementations in `internal/agent/<type>/`
4. **Data access layer**
   - Config parsers in `internal/config/`
   - Runtime metrics clients in `internal/metrics/`
5. **Rendering layer**
   - Table/JSON formatting in `internal/output/`
6. **Integration and verification**
   - Unit tests across packages
   - Docker-based integration tests in `test/integration/`

## 3) Package Responsibilities

### Entry Point and Command Tree

- `cmd/agent-cli/main.go`: process entry point
- `internal/cli/root.go`: root command registration
- `internal/cli/status.go`: `status` command orchestration
- `internal/cli/discover.go`: discovery command surface
- `internal/cli/tui.go`: TUI launch path and live-refresh flags

### Shared Models and Health Logic

- `internal/pipeline/model.go`: canonical `Pipeline`, `Node`, `Edge`, metric payloads
- `internal/pipeline/health.go`: normalization/mapping of runtime status to common health states

### Agent Adapters

- `internal/agent/elasticagent/`
  - Parses Elastic config wiring
  - Calls Elastic status endpoint
  - Maps component units and Beat-derived metrics into shared model
- `internal/agent/edot/`
  - Parses OTel/EDOT config-defined topology
  - Reads zpages topology (`/debug/pipelinez`, `/debug/tracez`)
  - Scrapes Prometheus self-telemetry (`/metrics`)
  - Applies OTel/EDOT health evaluation and Elastic-specific component labeling

### Config and Metrics Services

- `internal/config/elastic.go`: Elastic Agent config parser/mapping
- `internal/config/otel.go`: OTel/EDOT parser (receivers/processors/exporters/extensions/service pipelines)
- `internal/metrics/collector.go`: generic HTTP collector utilities
- `internal/metrics/prometheus.go`: OTel metric extraction for key counters

### Output and TUI

- `internal/output/`: table and JSON rendering
- `internal/tui/app.go`: root Bubbletea model
- `internal/tui/dashboard/`: dashboard screen and OTel/Elastic column adaptation

## 4) Runtime Data Flow

## 4.1 `agent-cli status`

1. CLI parses flags and selects target adapter (`elastic-agent`, `edot`)
2. Adapter loads config as baseline topology
3. Adapter queries runtime endpoints (status/zpages/metrics/health)
4. Adapter merges static + runtime data into `pipeline.Pipeline`
5. Renderer outputs table or JSON

## 4.2 `agent-cli tui`

1. Reuses status pipeline construction path
2. Injects built pipeline into dashboard model
3. Supports snapshot mode and periodic refresh loop (`--live`, `--refresh`)

## 4.3 `agent-cli discover`

- Uses discovery strategies to locate local agents and candidate endpoints/config paths
- Explicit CLI flags override discovery output

## 5) Health and Metrics Model

Health states across agent types:

- `healthy`
- `degraded`
- `error`
- `disabled`
- `unknown`

Current P0/P1 OTel metrics collected:

- `otelcol_receiver_accepted_*`
- `otelcol_exporter_sent_*`
- `otelcol_processor_dropped_*`
- `otelcol_exporter_send_failed_*`

Implementation note: Prometheus parsing now correctly reads the sample value token even when lines include optional trailing timestamps.

## 6) EDOT/OTel Specific Notes

- Processor ordering is semantically significant and preserved from config-defined order when constructing pipeline edges.
- zpages pipeline parsing prefers JSON responses; HTML fallback is intentionally limited to deterministic test fixtures that use `data-*` attributes.
- Health combines runtime status with metrics-derived degradation/error signals.

## 7) Testing and Quality Gates

Quality gates (backpressure) expected before task completion:

- `go build ./...`
- `go test -race ./...`
- `go vet ./...`
- `golangci-lint run`

Integration tests:

- `test/integration/docker-compose.yml` boots real services
- `test/integration/*_test.go` validates adapter behavior against live containers

## 8) CI/CD and Release Flow

- CI workflow: lint -> tests -> build (plus integration path)
- Release workflow uses GoReleaser for multi-platform artifacts
- Local Make targets wrap build/test/lint actions

## 9) Current Boundaries and Planned Evolution

Implemented:

- Phase 0 foundation
- Phase 1A Elastic Agent status
- Phase 1B EDOT status (in review)

Planned:

- Full generic OTel adapter path
- Deeper TUI drill-down screens
- Phase 2 config modification workflows
- Remote agent support with authentication/TLS

## 10) Documentation Ownership

This file is a living architecture contract. Any PR that changes architecture, data flow, package responsibilities, major runtime behavior, or technical constraints must update:

- `architecture.md`
- `decision-log.md`
