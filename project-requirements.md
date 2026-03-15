# agent-cli

> CLI & TUI tool for managing and inspecting the configuration of Elastic Agent, EDOT Collectors, and generic OpenTelemetry Collectors.

**Status:** Greenfield project — no existing tooling to migrate from or integrate with.

---

## 1. Problem Statement

Managing the configuration of observability agents (Elastic Agent, EDOT Collector, vanilla OTel Collector) is complex and error-prone. Operators need a way to:

- **Inspect** the current state of an agent's pipeline (inputs → processors/transforms → outputs) at a glance.
- **Assess health** — know whether each component is working, degraded, or errored.
- **View metrics** — throughput, error rates, buffer pressure, etc., per pipeline stage.
- **Spot issues** — disabled components, configuration errors, and mismatches should be surfaced proactively.
- **Modify configuration** (Phase 2) — add, edit, or remove inputs, processors, and outputs without hand-editing YAML.

There is currently no unified tool that works across all three agent types with both a rich TUI and scriptable CLI interface.

---

## 2. Objectives & Success Criteria

| # | Objective | Success Metric |
|---|-----------|---------------|
| 1 | Unified status view across agent types | Single command produces a pipeline-oriented status report for EA, EDOT, and OTel collectors |
| 2 | Health assessment | Each pipeline node shows a clear healthy / degraded / error state with reason |
| 3 | Metrics at a glance | Key throughput & error metrics displayed inline per pipeline stage |
| 4 | Surface misconfigurations | Disabled inputs, invalid config blocks, and warnings are highlighted |
| 5 | Rich TUI experience | Interactive terminal UI with navigation, filtering, drill-down |
| 6 | Scriptable CLI output | Machine-readable output (JSON, table) for CI/CD and automation |
| 7 | Phase 2: Config modification | Add/edit/remove inputs, processors, outputs via guided TUI or CLI flags |

---

## 3. Target Agent Types

### 3.1 Elastic Agent (standalone & Fleet-managed)
- Configuration: `elastic-agent.yml` + Fleet policies
- Runs Beats-based inputs (filebeat, metricbeat, etc.) as sub-processes
- Control protocol: gRPC-based internal control plane
- Status API: `elastic-agent status` (existing), HTTP diagnostics endpoint

### 3.2 EDOT Collector (Elastic Distribution of OpenTelemetry)
- Configuration: OTel-style YAML with Elastic-specific extensions
- Pipeline model: receivers → processors → exporters (with connectors & extensions)
- Potential status sources: `zpages` extension, health check extension, internal telemetry

### 3.3 Generic OTel Collector
- Same pipeline model as EDOT but without Elastic extensions
- Status via: zpages, health_check extension, OTLP self-telemetry
- Wide variety of community receivers/processors/exporters

### 3.4 Minimum Supported Versions

| Agent | Minimum Version | Rationale |
|-------|----------------|-----------|
| **Elastic Agent** | 9.x | Current major version; aligns with latest APIs and config format |
| **EDOT Collector** | Latest compatible with EA 9.x | Elastic's OTel distribution tracks EA releases |
| **OTel Collector** | 0.100+ | Modern component status API, stable zpages, recent enough for EDOT parity |

> Older versions are explicitly out of scope. This avoids dealing with legacy API formats and deprecated config structures.

---

## 4. Architecture

### 4.1 Deployment Model: Hybrid (Library + Standalone Binary)

```
┌───────────────────┐        ┌──────────────────────┐
│  agent-cli binary  │──HTTP──▶│  Agent (EA/EDOT/OTel) │
│  (standalone)      │──gRPC──▶│  running on localhost  │
└───────────────────┘        └──────────────────────┘

┌───────────────────┐
│  elastic-agent     │  (future: embeds pkg/agentcli
│  inspect subcommand│   for deeper introspection)
└───────────────────┘
```

**Rationale:** The core logic lives in `pkg/agentcli` as a reusable Go library. A standalone `agent-cli` binary is the primary distribution for operators and SREs. Agents (Elastic Agent, EDOT) *can* embed the library as a subcommand (e.g., `elastic-agent inspect`) for deeper introspection in future iterations.

**Implications:**
- Public API surface in `pkg/agentcli` must be well-designed from day one
- Agent adapters must work both in-process (embedded) and out-of-process (standalone)
- Standalone binary is the primary focus; embedded integration is a future enhancement

### 4.2 Network Scope
- **Phase 1:** Localhost only. All agent communication via `127.0.0.1` or unix sockets.
- **Phase 2:** Remote agents via `--remote host:port` with TLS and auth support.

---

## 5. Technology Stack

### 5.1 Language: Go
**Rationale:** Elastic Agent, the OTel Collector, and Beats are all written in Go. Using Go allows:
- Direct import of config parsing logic from `elastic/elastic-agent`, `open-telemetry/opentelemetry-collector`
- Familiar toolchain for the Elastic agent team
- Single static binary distribution

### 5.2 CLI Framework

| Library | Stars | Approach | Pros | Cons |
|---------|-------|----------|------|------|
| **[cobra](https://github.com/spf13/cobra)** | 39k+ | Command tree + flags | Industry standard (used by kubectl, docker, gh). Huge ecosystem, auto-completion, man pages | Verbose boilerplate |
| **[urfave/cli](https://github.com/urfave/cli)** | 22k+ | Struct-based | Simpler API, less boilerplate | Smaller plugin ecosystem |
| **[kong](https://github.com/alecthomas/kong)** | 2k+ | Struct tags | Very concise, type-safe | Less widely adopted |

**Decision: Cobra** — de facto standard for Go CLIs, both Elastic Agent and the OTel Collector already use it, and it makes shell completion trivial.

### 5.3 TUI Framework

| Library | Stars | Approach | Pros | Cons |
|---------|-------|----------|------|------|
| **[bubbletea](https://github.com/charmbracelet/bubbletea)** | 29k+ | Elm architecture (Model-Update-View) | Composable, testable, huge component library (bubbles), beautiful styling (lipgloss) | Learning curve for Elm pattern |
| **[tview](https://github.com/rivo/tview)** | 11k+ | Widget-based (traditional) | Familiar widget model (tables, forms, trees), built-in layouts | Harder to compose custom views, less modern feel |
| **[termui](https://github.com/gizak/termui)** | 13k+ | Dashboard widgets | Great for dashboards, sparklines, gauges | Less flexible for interactive forms |
| **[gocui](https://github.com/jroimartin/gocui)** | 10k+ | Low-level views | Maximum control | Lots of manual work |

**Decision: Bubbletea + Lipgloss + Bubbles** — The Charm ecosystem is the modern standard for Go TUIs. The Elm architecture makes it highly testable and composable. Components like tables, spinners, viewports, and text inputs are available out of the box via `bubbles`. Lipgloss handles styling.

**Additional Charm libraries:**
- **[huh](https://github.com/charmbracelet/huh)** — Form/wizard framework (Phase 2 config editing)
- **[glamour](https://github.com/charmbracelet/glamour)** — Markdown rendering in terminal
- **[log](https://github.com/charmbracelet/log)** — Structured logging with pretty output

### 5.4 Output Formats (CLI mode)
- **Table** (default for humans) — using `lipgloss/table`
- **JSON** (for scripting/piping)
- **YAML** (for config-native workflows)
- **Pipeline diagram** (ASCII art for `status` command)

---

## 6. Agent Discovery

### 6.1 Strategy: Multi-layered with Auto-Detect

`agent-cli` uses a layered discovery approach. Explicit flags always win; auto-detect is the fallback.

**Priority (highest to lowest):**

1. **Explicit flags** — `--config /path/to/config.yml` or `--endpoint localhost:55679`
2. **Process scan** — Scan running processes for known binaries (`elastic-agent`, `otelcol`, `otelcol-contrib`, `edot-collector`). Extract config paths and API ports from process arguments.
3. **Well-known paths** — Check default config file locations:
   - Elastic Agent: `/opt/Elastic/Agent/elastic-agent.yml`, `C:\Program Files\Elastic\Agent\elastic-agent.yml`
   - OTel Collector: `/etc/otelcol/config.yaml`, `/etc/otel-collector/config.yaml`
   - EDOT: `/etc/edot/config.yaml`
4. **Well-known ports** — Probe default API endpoints:
   - Elastic Agent: `localhost:6791` (status API)
   - OTel zpages: `localhost:55679`
   - OTel health_check: `localhost:13133`
   - Prometheus metrics: `localhost:8888`

### 6.2 Discovery Command

```bash
agent-cli discover
# Output:
# Found 2 agents:
#   1. elastic-agent (PID 1234) — /opt/Elastic/Agent/elastic-agent.yml
#   2. otelcol (PID 5678) — /etc/otelcol/config.yaml

agent-cli status                          # status of all discovered agents
agent-cli status --agent elastic-agent    # filter to specific type
agent-cli status --config /path/to/cfg    # explicit config
```

### 6.3 Authentication

> Based on documentation review — to be validated with hands-on testing.

| Agent | Local API Auth | Notes |
|-------|---------------|-------|
| **Elastic Agent** | Minimal by default | Status API on `localhost:6791` is unauthenticated in standalone mode. Fleet-managed agents may require Fleet enrollment token for policy reads. gRPC control socket is unix domain socket (OS-level auth). |
| **OTel Collector** | None by default | zpages, health_check, and Prometheus `/metrics` endpoints are unauthenticated by default. Auth can be added via `configauth` extension but uncommon for localhost. |
| **EDOT Collector** | Same as OTel | Follows upstream OTel defaults. Elastic extensions don't add auth to local endpoints. |

**Phase 1 approach:** Assume unauthenticated localhost access. Add `--token` / `--tls-cert` / `--tls-key` flags as optional overrides. Full auth support (mTLS, OAuth, API keys) deferred to Phase 2 with remote agent support.

---

## 7. Distribution

| Channel | Target | Notes |
|---------|--------|-------|
| **GitHub Releases** | All platforms | Cross-compiled binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64. GoReleaser for automation. Checksums + signatures. |
| **Homebrew** | macOS / Linux | `brew install elastic/tap/agent-cli` |
| **apt / yum** | Linux servers | Elastic's existing package repos or dedicated PPA |
| **Docker image** | CI/CD, containers | `docker run --net=host elastic/agent-cli status` (needs host network for localhost agent access) |

---

## 8. Phase 1 — Status Report

### 8.1 Core Concept: Pipeline Visualization

The status report models the agent's configuration as a directed graph of pipeline stages:

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│  INPUTS      │───▶│  PROCESSORS   │───▶│  OUTPUTS     │
│              │    │  /TRANSFORMS  │    │              │
│ ● filebeat   │    │ ● add_fields  │    │ ● elasticsearch│
│   ✓ running  │    │   ✓ active    │    │   ✓ connected  │
│   1.2k eps   │    │   0 dropped   │    │   1.1k eps     │
│              │    │               │    │                │
│ ● metricbeat │    │ ● drop_event  │    │ ● logstash     │
│   ✓ running  │    │   ✓ active    │    │   ⚠ slow       │
│   340 eps    │    │   12 dropped  │    │   290 eps      │
│              │    │               │    │                │
│ ● httpjson   │    │               │    │                │
│   ✗ ERROR    │    │               │    │                │
│   "conn refused"│ │               │    │                │
└─────────────┘    └──────────────┘    └─────────────┘
```

### 8.2 Metrics Priority

Metrics displayed per-component in order of importance:

| Priority | Metric | Display | Source |
|----------|--------|---------|--------|
| **P0** | Events/sec throughput (in/out) | `1.2k eps ▶ 1.1k eps` | Beat metrics, OTel `otelcol_receiver_accepted_*` / `otelcol_exporter_sent_*` |
| **P1** | Error & drop rates | `⚠ 12 dropped/min`, `✗ 3 errors/min` | Beat metrics, `otelcol_processor_dropped_*`, `otelcol_exporter_send_failed_*` |
| **P2** | Resource usage | `MEM 142MB`, `CPU 2.3%`, `FDs 84` | Process stats, `/proc`, agent self-telemetry |
| **P3** | Latency | `p99: 12ms export`, `avg: 2ms proc` | OTel span metrics, Beat internal timing |

In the TUI dashboard, P0 and P1 are always visible. P2 and P3 shown in detail drill-down screens.

### 8.3 Data to Collect Per Agent Type

#### Elastic Agent
| Data Point | Source |
|-----------|--------|
| Running inputs (beats) | `elastic-agent status` API / gRPC control |
| Input configuration | `elastic-agent.yml` or Fleet policy |
| Per-input metrics | Beat internal metrics (HTTP endpoint) |
| Output status | Beat output metrics |
| Errors / warnings | Agent logs, status API |
| Disabled inputs | Config file parse (enabled: false) |

#### EDOT / OTel Collector
| Data Point | Source |
|-----------|--------|
| Active receivers | zpages `/debug/tracez`, `/debug/pipelinez` |
| Active processors | zpages pipeline view |
| Active exporters | zpages, exporter metrics |
| Per-component metrics | Prometheus self-telemetry (`/metrics`) |
| Pipeline topology | Configuration file parse + service.pipelines |
| Health status | `health_check` extension |
| Errors | Component status from `componentstatus` package |

### 8.4 Status Command UX

```bash
# CLI mode
agent-cli status                          # auto-detect local agent
agent-cli status --agent elastic-agent    # target specific agent type
agent-cli status --agent otel             # target OTel collector
agent-cli status --format json            # machine-readable
agent-cli status --format yaml            # YAML output

# TUI mode
agent-cli tui                             # launch interactive dashboard
agent-cli tui --agent elastic-agent
agent-cli tui --live                      # auto-refresh mode
agent-cli tui --refresh 5s               # custom refresh interval
```

### 8.5 TUI Screens (Phase 1)

1. **Dashboard** — High-level pipeline diagram with health indicators
2. **Input Detail** — Drill into a specific input: config, metrics, recent errors
3. **Output Detail** — Connection status, throughput, backpressure
4. **Processor Detail** — Drop rates, transformation stats
5. **Errors & Warnings** — Aggregated view of all issues
6. **Raw Config** — View the effective configuration with syntax highlighting

### 8.6 TUI Refresh Behavior

- **Default: Snapshot mode** — data fetched once on launch and on manual refresh (`r` key)
- **Live mode: `agent-cli tui --live` or `agent-cli tui --refresh 5s`** — auto-refreshes at configurable interval
- Live mode displays a refresh indicator and last-updated timestamp
- User can toggle between snapshot/live within the TUI (`L` key)
- Rate limiting on refresh to avoid overwhelming agent APIs

### 8.7 Health Assessment Logic

| State | Condition | Display |
|-------|-----------|---------|
| ✓ Healthy | Component running, no errors, metrics within thresholds | Green |
| ⚠ Degraded | Component running but with warnings (high latency, retries, partial errors) | Yellow |
| ✗ Error | Component failed, not running, or config invalid | Red |
| ○ Disabled | Component present in config but explicitly disabled | Gray |
| ? Unknown | Cannot determine status (API unreachable, metrics unavailable) | Dim |

---

## 9. Phase 2 — Configuration Modification (Future)

### 9.1 Capabilities
- **Add** a new input/processor/output via guided wizard (TUI) or flags (CLI)
- **Edit** existing component configuration
- **Remove** a component from the pipeline
- **Enable/Disable** a component without removing it
- **Validate** configuration before applying
- **Diff** — show what will change before committing
- **Backup** — auto-backup config before modification

### 9.2 Safety
- Dry-run mode by default
- Config validation against agent-specific schemas
- Rollback support (keep N previous configs)
- Fleet-managed agents: read-only warning (modifications must go through Fleet)

---

## 10. Testing Strategy

### 10.1 Unit Tests (Mock Agent APIs)
- Mock HTTP servers simulating Elastic Agent status API, OTel zpages, Prometheus endpoints
- Test all adapters against mocked responses for each agent type
- Test discovery logic with mocked process lists and file system
- Test health assessment logic with various metric scenarios
- Test output formatters (JSON, table, pipeline diagram)
- Test TUI models in isolation using Bubbletea's test utilities (`tea.TestModel`)

### 10.2 Integration Tests (Real Agents in CI)
- Docker Compose environment spinning up real agent instances:
  - Elastic Agent 9.x (standalone mode) with sample inputs
  - EDOT Collector with sample receivers/exporters
  - Vanilla OTel Collector (contrib) with sample pipelines
- `agent-cli` runs against these live agents in CI
- Golden file comparisons for expected status output
- Tests cover: discovery, status collection, health assessment, all output formats
- Run on every PR; full matrix on merge to main

### 10.3 CI Pipeline
```
lint → unit tests → build → integration tests (Docker) → release (on tag)
```

Tools: GitHub Actions, GoReleaser, Docker Compose, `golangci-lint`

---

## 11. Project Structure

```
agent-cli/
├── cmd/
│   └── agent-cli/
│       └── main.go                 # Entry point
├── internal/
│   ├── cli/                        # Cobra command definitions
│   │   ├── root.go
│   │   ├── status.go
│   │   ├── discover.go
│   │   └── tui.go
│   ├── tui/                        # Bubbletea models & views
│   │   ├── app.go                  # Root TUI model
│   │   ├── dashboard/              # Dashboard screen
│   │   ├── detail/                 # Detail drill-down screens
│   │   └── components/             # Reusable TUI components
│   ├── agent/                      # Agent abstraction layer
│   │   ├── agent.go                # Interface definition
│   │   ├── elasticagent/           # Elastic Agent adapter
│   │   ├── edot/                   # EDOT collector adapter
│   │   └── otel/                   # Generic OTel adapter
│   ├── discovery/                  # Agent discovery logic
│   │   ├── discover.go             # Orchestrator
│   │   ├── process.go              # Process scanning
│   │   ├── paths.go                # Well-known paths
│   │   └── probe.go                # Port probing
│   ├── pipeline/                   # Pipeline model (DAG of components)
│   │   ├── model.go                # Pipeline, Node, Edge types
│   │   ├── health.go               # Health assessment logic
│   │   └── render.go               # ASCII/TUI pipeline rendering
│   ├── config/                     # Config parsing per agent type
│   │   ├── elastic.go
│   │   ├── otel.go
│   │   └── schema.go
│   ├── metrics/                    # Metrics collection & aggregation
│   │   ├── collector.go
│   │   ├── prometheus.go           # Scrape Prometheus endpoints
│   │   └── grpc.go                 # gRPC status queries
│   └── output/                     # Output formatters
│       ├── table.go
│       ├── json.go
│       └── pipeline.go             # ASCII pipeline diagram
├── pkg/                            # Public library (for hybrid/embedded use)
│   └── agentcli/
│       ├── status.go
│       ├── discovery.go
│       └── types.go
├── test/
│   ├── integration/                # Integration test suites
│   │   └── docker-compose.yml      # Real agent environments
│   ├── fixtures/                   # Sample configs & golden files
│   │   ├── elastic-agent.yml
│   │   ├── otel-config.yaml
│   │   └── golden/
│   └── mocks/                      # Mock HTTP servers for unit tests
├── .github/
│   └── workflows/
│       ├── ci.yml                  # Lint + test + build
│       └── release.yml             # GoReleaser on tag
├── .goreleaser.yml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 12. Phased Delivery Plan

### Phase 0 — Foundation (Weeks 1–2)
- [ ] Project scaffolding (Go module, CI, linting, Makefile, GoReleaser)
- [ ] Define core interfaces: `Agent`, `Pipeline`, `Component`, `HealthStatus`
- [ ] Cobra CLI skeleton with `status`, `discover`, and `tui` commands
- [ ] Basic Bubbletea app shell with navigation
- [ ] Output formatters (JSON, table)
- [ ] Agent discovery framework (process scan, well-known paths, port probe)
- [ ] Mock agent HTTP servers for unit testing
- [ ] Docker Compose integration test scaffold

### Phase 1A — Elastic Agent Status [P0 — Must Have] (Weeks 3–5)
- [ ] Parse `elastic-agent.yml` configuration (9.x format)
- [ ] Connect to Elastic Agent status API (HTTP/gRPC)
- [ ] Map Elastic Agent components to pipeline model
- [ ] Collect per-input/output metrics from Beat HTTP endpoints
- [ ] Health assessment logic for EA components
- [ ] CLI table + JSON output for `agent-cli status --agent elastic-agent`
- [ ] TUI dashboard screen for Elastic Agent
- [ ] Unit tests with mocked EA APIs
- [ ] Integration tests with real EA 9.x in Docker

### Phase 1B — EDOT Collector Status [P1 — High Priority] (Weeks 5–7)
- [ ] Parse EDOT/OTel collector YAML configuration
- [ ] Connect to zpages extension for pipeline topology
- [ ] Scrape Prometheus self-telemetry endpoint
- [ ] Map OTel components to pipeline model
- [ ] Handle Elastic-specific receivers/exporters/extensions
- [ ] Health assessment for EDOT components
- [ ] CLI + TUI support for `agent-cli status --agent edot`
- [ ] Unit + integration tests for EDOT

### Phase 1C — Generic OTel Collector Status [P2 — Nice to Have] (Weeks 7–8)
- [ ] Generalize EDOT adapter to work with vanilla OTel Collector
- [ ] Handle community components not present in EDOT
- [ ] CLI + TUI support for `agent-cli status --agent otel`
- [ ] Integration tests with otelcol-contrib in Docker

### Phase 1D — Elastic Agent Fleet-Managed [P3 — Future] (Weeks 8–9)
- [ ] Read-only inspection of Fleet-managed agent policies
- [ ] Fleet API integration for policy retrieval
- [ ] Clear UX indicators that config is Fleet-managed (read-only)

### Phase 1E — Polish & Cross-cutting (Weeks 9–10)
- [ ] TUI drill-down screens (input detail, output detail, errors)
- [ ] Pipeline ASCII diagram renderer
- [ ] Shell completion (bash, zsh, fish, powershell)
- [ ] TUI live mode (auto-refresh with `--live` / `--refresh` flags)
- [ ] Documentation & README
- [ ] First GitHub release via GoReleaser
- [ ] Homebrew formula, Docker image

### Phase 2 — Configuration Modification (Weeks 11+)
- [ ] Config modification data model & validation
- [ ] TUI wizard flows using `huh` forms
- [ ] CLI flags for add/edit/remove operations
- [ ] Dry-run & diff display
- [ ] Backup & rollback mechanism
- [ ] Fleet-managed detection & warnings
- [ ] Remote agent support (`--remote host:port` + TLS/auth)

---

## 13. Decisions Log

All key decisions made during planning, for reference:

| # | Decision | Choice | Rationale |
|---|----------|--------|-----------|
| 1 | Deployment model | Hybrid (library + standalone binary) | Independent release cycle; agents can embed later |
| 2 | Tool name | `agent-cli` | Clean, descriptive, agent-type-agnostic |
| 3 | Agent priority | EA standalone → EDOT → OTel → EA Fleet | Focus on highest-value, most-controlled target first |
| 4 | TUI refresh | Snapshot default, live opt-in | Avoid API pressure; user chooses when to go live |
| 5 | Agent discovery | Auto-detect (process scan + paths + ports) | Frictionless first run; explicit flags as override |
| 6 | CLI framework | Cobra | Industry standard; already used by EA and OTel |
| 7 | TUI framework | Bubbletea + Charm ecosystem | Most modern, testable, composable Go TUI stack |
| 8 | Metrics priority | Throughput → Errors → Resources → Latency | Operators care about data flow first |
| 9 | Auth (Phase 1) | Unauthenticated localhost | Simplifies P1; auth flags available as override |
| 10 | Network scope (Phase 1) | Localhost only | Remote support deferred to Phase 2 |
| 11 | Min versions | Elastic Agent 9.x, OTel 0.100+ | Current gen only; avoid legacy API baggage |
| 12 | Distribution | GitHub Releases, Homebrew, apt/yum, Docker | Cover all common installation paths |
| 13 | Testing | Mocked APIs + real agents in Docker CI | Comprehensive coverage without flaky tests |
| 14 | Existing tooling | Greenfield | No migration or integration concerns |

---

## 14. Config File Research (On-Disk Formats)

### 14.1 Elastic Agent 9.x Standalone (`elastic-agent.yml`)

**Location on disk:**
- Linux: `/opt/Elastic/Agent/elastic-agent.yml`
- macOS: `/Library/Elastic/Agent/elastic-agent.yml`
- Windows: `C:\Program Files\Elastic\Agent\elastic-agent.yml`
- Fleet-managed: same path, but the file is written/overwritten by Fleet policy sync

**Top-level structure:**

```yaml
# ─── OUTPUTS ───────────────────────────────────────
outputs:
  default:                          # named output (can have multiple)
    type: elasticsearch
    hosts: ["https://es:9200"]
    api_key: "id:key"
    preset: balanced                # performance preset (new in 8.x+)
  
  monitoring:
    type: elasticsearch
    hosts: ["https://es-mon:9200"]
    api_key: "id:key2"

# ─── INPUTS ────────────────────────────────────────
inputs:
  - id: system-logs                 # unique identifier
    type: filestream                # input type (was "log" pre-8.x)
    enabled: true                   # can be explicitly disabled
    use_output: default             # maps to named output above
    streams:
      - id: syslog
        paths: ["/var/log/syslog", "/var/log/messages"]
        exclude_lines: ['^DBG']
        parsers:
          - ndjson:
              target: ""
        processors:                 # per-stream processors
          - add_fields:
              target: ""
              fields:
                environment: production

  - id: system-metrics
    type: system/metrics
    use_output: default
    streams:
      - id: cpu
        metricsets: ["cpu"]
        period: 10s
      - id: memory
        metricsets: ["memory"]
        period: 10s
      - id: filesystem
        metricsets: ["filesystem"]
        period: 60s

  - id: apm-traces                  # APM / trace input
    type: apm
    use_output: default
    apm-server:
      host: "0.0.0.0:8200"
      rum.enabled: true

  - id: custom-http
    type: httpjson
    enabled: false                  # ← DISABLED input
    use_output: default
    config_version: 2
    request.url: "https://api.example.com/events"
    interval: 60s

# ─── AGENT SETTINGS ───────────────────────────────
agent:
  monitoring:
    enabled: true
    logs: true
    metrics: true
    use_output: monitoring          # self-monitoring to separate output
  download:
    sourceURI: "https://artifacts.elastic.co/downloads/"
  logging:
    level: info
    to_files: true
```

**Key observations for agent-cli:**
- `inputs[].type` determines the signal: `filestream`/`log`/`httpjson`/`journald` → logs, `system/metrics`/`*beat` → metrics, `apm` → traces
- `inputs[].use_output` links an input to a named output — this is the pipeline wiring
- `inputs[].enabled: false` is how configs are disabled but kept on disk
- `inputs[].streams[].processors` are the per-stream transforms
- `outputs` is a named map — multiple outputs supported
- No explicit "pipeline" concept like OTel — the wiring is implicit via `use_output`
- `agent.monitoring` is the self-monitoring config — separate from data pipelines
- Errors on disk: malformed YAML, missing required fields, referencing non-existent output names, duplicate IDs

### 14.2 EDOT / OTel Collector (`otel-collector.yml` or `config.yaml`)

**Location on disk:**
- Linux: `/etc/otelcol/config.yaml` or `/etc/edot/config.yaml`
- Docker: mounted via `-v`, typically at `/etc/otelcol-contrib/config.yaml`
- Custom: passed via `--config` flag

**Top-level structure:**

```yaml
# ─── RECEIVERS (inputs) ───────────────────────────
receivers:
  otlp:                             # accepts traces, metrics, logs via OTLP
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

  filelog:                          # log file tailing (like filebeat)
    include: ["/var/log/*.log"]
    operators:
      - type: regex_parser
        regex: '^(?P<time>\S+) (?P<level>\S+) (?P<msg>.*)'

  hostmetrics:                      # system metrics (like metricbeat)
    collection_interval: 10s
    scrapers:
      cpu:
      memory:
      disk:
      network:

  prometheus:                       # scrape Prometheus targets
    config:
      scrape_configs:
        - job_name: 'app'
          static_configs:
            - targets: ['app:8080']

# ─── PROCESSORS (transforms) ──────────────────────
processors:
  batch:                            # batches before export
    timeout: 5s
    send_batch_size: 1024

  attributes:                       # add/modify/delete attributes
    actions:
      - key: environment
        value: production
        action: upsert

  filter:                           # drop unwanted telemetry
    error_mode: ignore
    metrics:
      exclude:
        match_type: strict
        metric_names: ["system.disk.io"]

  transform:                        # OTTL-based transformations
    log_statements:
      - context: log
        statements:
          - 'set(severity_text, "INFO") where severity_number == 9'

  resourcedetection:                # auto-detect cloud/host metadata
    detectors: [env, system, docker, ec2, gcp]

# ─── EXPORTERS (outputs) ──────────────────────────
exporters:
  elasticsearch:                    # Elastic-specific (EDOT)
    endpoints: ["https://es:9200"]
    api_key: "id:key"
    logs_index: logs-generic-default
    mapping:
      mode: ecs

  otlphttp:                         # generic OTLP export
    endpoint: "https://other-backend:4318"

  debug:                            # console debug output
    verbosity: detailed

# ─── CONNECTORS ───────────────────────────────────
connectors:
  spanmetrics:                      # generates metrics from trace spans
    dimensions:
      - name: http.method
      - name: http.status_code

# ─── EXTENSIONS ───────────────────────────────────
extensions:
  zpages:
    endpoint: localhost:55679
  health_check:
    endpoint: localhost:13133
  pprof:
    endpoint: localhost:1777

# ─── SERVICE (pipeline wiring) ────────────────────
service:
  extensions: [zpages, health_check, pprof]
  
  pipelines:                        # ← THIS IS THE KEY SECTION
    traces:
      receivers: [otlp]
      processors: [resourcedetection, batch]
      exporters: [elasticsearch]

    metrics:
      receivers: [hostmetrics, otlp, spanmetrics]   # spanmetrics connector is a receiver here
      processors: [resourcedetection, batch, filter]
      exporters: [elasticsearch]

    logs:
      receivers: [filelog, otlp]
      processors: [resourcedetection, transform, attributes, batch]
      exporters: [elasticsearch, debug]

  telemetry:                        # self-monitoring
    logs:
      level: info
    metrics:
      address: localhost:8888
```

**Key observations for agent-cli:**
- `service.pipelines` is the explicit wiring — each signal type (traces, metrics, logs) defines exactly which receivers → processors → exporters
- A component defined in `receivers`/`processors`/`exporters` but NOT referenced in any pipeline is effectively **disabled/orphaned**
- `connectors` bridge pipelines (e.g., `spanmetrics` is an exporter in traces pipeline and a receiver in metrics pipeline)
- Processors are ordered — order in the `processors` array matters (executed left to right)
- Common error patterns: referencing undefined components, typos in component names, invalid config fields, port conflicts
- `extensions` are sidecar services (zpages, health_check) — not part of data pipelines but important for status

### 14.3 Config Comparison Matrix

| Concept | Elastic Agent | EDOT / OTel Collector |
|---------|--------------|----------------------|
| **Config file** | `elastic-agent.yml` | `config.yaml` |
| **Inputs** | `inputs[].type` | `receivers:` |
| **Transforms** | `inputs[].streams[].processors` | `processors:` |
| **Outputs** | `outputs:` (named map) | `exporters:` |
| **Pipeline wiring** | Implicit via `use_output` | Explicit via `service.pipelines` |
| **Signal separation** | Inferred from input type | Explicit pipeline names (traces/metrics/logs) |
| **Disabled config** | `enabled: false` on input | Component defined but not in any pipeline |
| **Cross-signal bridging** | N/A | `connectors:` |
| **Self-monitoring** | `agent.monitoring` | `service.telemetry` |
| **Sidecars/extensions** | N/A | `extensions:` |
| **Processor ordering** | Per-stream, ordered array | Per-pipeline, ordered array |

### 14.4 Validation Checks (What agent-cli Should Flag)

**Both agent types:**
- ❌ Malformed YAML (parse errors)
- ❌ Empty config file
- ⚠️ No inputs/receivers defined
- ⚠️ No outputs/exporters defined

**Elastic Agent specific:**
- ❌ `use_output` references non-existent output name
- ❌ Duplicate input `id` values
- ⚠️ `enabled: false` — show as disabled, not error
- ⚠️ Input type not recognized / not installed
- ⚠️ Missing required fields per input type (e.g., `paths` for filestream)

**OTel / EDOT specific:**
- ❌ Component referenced in pipeline but not defined in top-level section
- ❌ Component defined but not referenced in any pipeline (orphaned)
- ❌ Port conflicts between extensions/receivers
- ⚠️ Empty pipeline (no receivers or no exporters)
- ⚠️ Connector used as receiver without corresponding exporter pipeline (or vice versa)
- ⚠️ Debug exporter in production config

---

## 15. TUI Wireframes — Lane-Based Pipeline View

### 15.1 EDOT / OTel Collector — Full Pipeline View

The three-lane layout: RECEIVERS (left) → PROCESSORS (center) → EXPORTERS (right).
Each signal (logs, metrics, traces) gets its own row with connecting lines.

```
 agent-cli status — edot-collector (PID 4521)        config: /etc/edot/config.yaml
 ═══════════════════════════════════════════════════════════════════════════════════

  RECEIVERS              PROCESSORS                        EXPORTERS
 ─────────────────────────────────────────────────────────────────────────────────

  traces
  ┌────────────────┐     ┌───────────────────┐  ┌───────┐  ┌────────────────┐
  │ otlp           │────▶│ resourcedetection  │─▶│ batch │─▶│ elasticsearch  │
  │ ✓ grpc :4317   │     │ ✓ ok              │  │ ✓ ok  │  │ ✓ connected    │
  │ ✓ http :4318   │     └───────────────────┘  └───────┘  │   1.2k eps     │
  │   820 spans/s  │                                        └────────────────┘
  └────────────────┘
                                                       ┌ ─ ─ ─ ─ ─ ─ ─ ─ ─┐
                                                         spanmetrics        
                                                       │ (connector ▶ metrics)│
                                                       └ ─ ─ ─ ─ ─ ─ ─ ─ ─┘
  metrics
  ┌────────────────┐     ┌───────────────────┐  ┌───────┐  ┌────────────────┐
  │ hostmetrics    │──┐  │ resourcedetection  │─▶│ batch │─▶│ elasticsearch  │
  │ ✓ cpu,mem,disk │  │  │ ✓ ok              │  │ ✓ ok  │  │ ✓ connected    │
  │   340 pts/s    │  ├─▶│                   │  │       │  │   890 pts/s    │
  ├────────────────┤  │  └───────────────────┘  │       │  └────────────────┘
  │ otlp           │──┤                         │       │
  │   200 pts/s    │  │  ┌───────────────────┐  │       │
  ├────────────────┤  └─▶│ filter            │─▶│       │
  │ spanmetrics    │──┘  │ ✓ ok              │  │       │
  │   350 pts/s    │     │   ⚠ 12 dropped/m  │  └───────┘
  └────────────────┘     └───────────────────┘

  logs
  ┌────────────────┐     ┌───────────────────┐  ┌────────────┐  ┌────────────────┐
  │ filelog        │──┐  │ resourcedetection  │─▶│ transform   │  │ elasticsearch  │
  │ ✓ /var/log/*   │  │  │ ✓ ok              │  │ ✓ ok        │  │ ✓ connected    │
  │   1.5k evts/s  │  ├─▶│                   │  │             │  │   2.1k eps     │
  ├────────────────┤  │  └───────────────────┘  └─────┬──────┘  └───────┬────────┘
  │ otlp           │──┘                               │                 │
  │   600 evts/s   │     ┌───────────────────┐  ┌─────▼──────┐  ┌──────▼─────────┐
  └────────────────┘     │ attributes        │─▶│ batch       │─▶│ debug          │
                         │ ✓ ok              │  │ ✓ ok        │  │ ✓ verbose      │
                         └───────────────────┘  └─────────────┘  └────────────────┘

 ─────────────────────────────────────────────────────────────────────────────────
  extensions
  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐
  │ zpages  :55679 │  │ health   :13133│  │ pprof    :1777 │
  │ ✓ running      │  │ ✓ running      │  │ ✓ running      │
  └────────────────┘  └────────────────┘  └────────────────┘

 ─────────────────────────────────────────────────────────────────────────────────
  ISSUES (2)
  ⚠ filter: 12 metrics dropped/min in metrics pipeline
  ⚠ debug exporter active — consider removing in production
```

### 15.2 Elastic Agent — Full Pipeline View

Elastic Agent's implicit wiring mapped to the same three-lane layout:

```
 agent-cli status — elastic-agent (PID 1234)     config: /opt/Elastic/Agent/elastic-agent.yml
 ═══════════════════════════════════════════════════════════════════════════════════

  INPUTS                 PROCESSORS                        OUTPUTS
 ─────────────────────────────────────────────────────────────────────────────────

  logs
  ┌────────────────┐     ┌───────────────────┐              ┌────────────────┐
  │ filestream      │────▶│ add_fields        │─────────────▶│ default        │
  │ id: system-logs │     │ env: production   │              │ elasticsearch  │
  │ ✓ running       │     │ ✓ ok              │              │ ✓ connected    │
  │   1.2k eps      │     └───────────────────┘              │ es:9200        │
  │                 │                                        │   3.4k eps     │
  │  streams:       │                                        └────────────────┘
  │   syslog ✓      │
  │   messages ✓    │
  └────────────────┘

  ┌────────────────┐                                        ┌────────────────┐
  │ httpjson        │ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ▶│ default        │
  │ id: custom-http │                                       │ (not reached)  │
  │ ○ DISABLED      │                                        └────────────────┘
  │   enabled: false│
  └────────────────┘

  metrics
  ┌────────────────┐                                        ┌────────────────┐
  │ system/metrics  │──────────────────────────────────────▶│ default        │
  │ id: sys-metrics │        (no processors)                │ elasticsearch  │
  │ ✓ running       │                                       │ ✓ connected    │
  │   340 pts/s     │                                       │   3.4k eps     │
  │                 │                                       └────────────────┘
  │  streams:       │
  │   cpu     ✓ 10s │
  │   memory  ✓ 10s │
  │   fs      ✓ 60s │
  └────────────────┘

  traces
  ┌────────────────┐                                        ┌────────────────┐
  │ apm             │──────────────────────────────────────▶│ default        │
  │ id: apm-traces  │        (no processors)                │ elasticsearch  │
  │ ✓ running       │                                       │ ✓ connected    │
  │   :8200         │                                       │   3.4k eps     │
  │   820 traces/s  │                                       └────────────────┘
  │   rum: enabled  │
  └────────────────┘

 ─────────────────────────────────────────────────────────────────────────────────
  monitoring
  ┌────────────────┐                                        ┌────────────────┐
  │ self-monitoring │──────────────────────────────────────▶│ monitoring     │
  │ ✓ logs: on      │                                       │ elasticsearch  │
  │ ✓ metrics: on   │                                       │ ✓ es-mon:9200  │
  └────────────────┘                                        └────────────────┘

 ─────────────────────────────────────────────────────────────────────────────────
  ISSUES (1)
  ○ httpjson (id: custom-http) is disabled — enable with 'enabled: true'
```

### 15.3 Error State Example

What it looks like when things are broken:

```
 agent-cli status — edot-collector (PID 4521)        config: /etc/edot/config.yaml
 ═══════════════════════════════════════════════════════════════════════════════════

  RECEIVERS              PROCESSORS                        EXPORTERS
 ─────────────────────────────────────────────────────────────────────────────────

  traces
  ┌────────────────┐     ┌───────────────────┐              ┌────────────────┐
  │ otlp           │────▶│ batch             │─────────────▶│ elasticsearch  │
  │ ✓ grpc :4317   │     │ ✓ ok              │              │ ✗ ERROR        │
  │   0 spans/s    │     └───────────────────┘              │ connection     │
  └────────────────┘                                        │ refused :9200  │
                                                            └────────────────┘
  metrics
  ┌────────────────┐     ┌───────────────────┐              ┌────────────────┐
  │ hostmetrics    │────▶│ batch             │─────────────▶│ elasticsearch  │
  │ ✓ running      │     │ ⚠ queue 89% full  │              │ ✗ ERROR        │
  │   340 pts/s    │     └───────────────────┘              │ (same as above)│
  └────────────────┘                                        └────────────────┘

  logs
  ┌────────────────┐
  │ filelog        │     (no pipeline — orphaned receivers)
  │ ? ORPHANED     │
  │ defined but not│
  │ in any pipeline│
  └────────────────┘

 ─────────────────────────────────────────────────────────────────────────────────
  ISSUES (3)
  ✗ elasticsearch exporter: connection refused at localhost:9200
  ⚠ batch processor: queue at 89% capacity in metrics pipeline — backpressure risk
  ✗ filelog receiver: defined in config but not referenced in service.pipelines
```

### 15.4 TUI Navigation Keys

```
 ─────────────────────────────────────────────────────────────────────
  ↑↓    Navigate components    │  Enter   Drill into detail
  Tab   Switch signal lane     │  r       Refresh data
  L     Toggle live mode       │  /       Filter/search
  e     Show errors only       │  c       View raw config
  q     Quit                   │  ?       Help
 ─────────────────────────────────────────────────────────────────────
```

---

## 16. API Endpoints Research (Runtime Status Sources)

> This section documents the APIs available at runtime for querying agent status, metrics, and health. These are the data sources agent-cli will use *after* config file parsing to enrich the pipeline view with live state.

### 16.1 Elastic Agent 9.x Status APIs

#### HTTP Status Endpoint
- **URL:** `http://localhost:6791/api/status` (default, configurable via `agent.monitoring.http`)
- **Auth:** None by default on localhost
- **Method:** `GET`

**Typical response:**
```json
{
  "id": "a1b2c3d4-...",
  "name": "my-agent",
  "status": {
    "overall": "HEALTHY",            // HEALTHY | DEGRADED | FAILED | STARTING | STOPPING
    "message": "Running"
  },
  "version": {
    "build_hash": "abc123",
    "build_time": "2025-01-15T...",
    "build_version": "9.0.0",
    "build_snapshot": false
  },
  "components": [                     // ← key section: one entry per running beat/component
    {
      "id": "filestream-default",
      "name": "filestream",
      "status": {
        "overall": "HEALTHY",
        "message": "Running"
      },
      "units": [                      // sub-units within the component
        {
          "id": "filestream-default-filestream-system-logs",
          "type": "INPUT",            // INPUT or OUTPUT
          "status": "HEALTHY",
          "message": "Running",
          "payload": {                // component-specific metadata
            "streams": {
              "syslog": {"status": "HEALTHY"}
            }
          }
        },
        {
          "id": "filestream-default-elasticsearch-default",
          "type": "OUTPUT",
          "status": "HEALTHY",
          "message": "Connected"
        }
      ]
    },
    {
      "id": "system-metrics-default",
      "name": "beat/metrics",
      "status": {
        "overall": "HEALTHY",
        "message": "Running"
      },
      "units": [
        {
          "id": "system-metrics-default-system/metrics-sys-metrics",
          "type": "INPUT",
          "status": "HEALTHY",
          "message": "Running"
        },
        {
          "id": "system-metrics-default-elasticsearch-default",
          "type": "OUTPUT",
          "status": "HEALTHY",
          "message": "Connected"
        }
      ]
    }
  ],
  "fleet_state": {                    // only present if Fleet-managed
    "state": "ONLINE",
    "message": "Connected"
  }
}
```

**Key observations for agent-cli:**
- `components[]` maps directly to running beat processes — each has input and output units
- `units[].type` tells us if it's an INPUT or OUTPUT — we can wire these to our pipeline model
- Status values: `HEALTHY`, `DEGRADED`, `FAILED`, `STARTING`, `CONFIGURING`, `STOPPING`
- `fleet_state` presence indicates Fleet-managed — agent-cli should flag as read-only
- Component IDs encode the input type and output name, parseable for pipeline wiring

#### Diagnostics Endpoint
- **URL:** `http://localhost:6791/api/diagnostics`
- **Method:** `POST`
- **Returns:** ZIP file containing logs, config, pprof, and component diagnostics
- **Use case:** Deep debugging, not for regular status polling

#### Beat HTTP Metrics (per sub-process)
Each beat sub-process exposes its own metrics:
- **URL:** `http://localhost:5066/stats` (default for filebeat/metricbeat)
- **Key metrics in response:**
```json
{
  "beat": {
    "info": {"version": "9.0.0", "name": "filebeat"},
    "memstats": {"memory_alloc": 14882816, "gc_next": 23456789},
    "cpu": {"total": {"value": 1234, "pct": 0.023}}
  },
  "filebeat": {
    "events": {
      "active": 12,
      "added": 145230,
      "done": 145218
    },
    "harvester": {
      "open_files": 4,
      "running": 4
    }
  },
  "output": {
    "elasticsearch": {
      "events": {"acked": 145100, "failed": 3, "dropped": 0, "batches": 1420},
      "read": {"bytes": 890234},
      "write": {"bytes": 45023400}
    }
  },
  "libbeat": {
    "pipeline": {
      "events": {"active": 12, "published": 145230, "total": 145230},
      "queue": {"acked": 145218, "max_events": 4096}
    },
    "output": {
      "events": {"acked": 145100, "active": 118, "batches": 1420, "failed": 3}
    }
  }
}
```

**Metrics mapping for agent-cli:**

| agent-cli metric | Beat stats field | Calculation |
|-----------------|------------------|-------------|
| Events/sec (in) | `libbeat.pipeline.events.published` | Delta / time window |
| Events/sec (out) | `libbeat.output.events.acked` | Delta / time window |
| Error rate | `libbeat.output.events.failed` | Delta / time window |
| Drop rate | `output.*.events.dropped` | Delta / time window |
| Queue pressure | `libbeat.pipeline.events.active` / `queue.max_events` | Percentage |
| Memory | `beat.memstats.memory_alloc` | Direct |
| CPU | `beat.cpu.total.pct` | Direct |
| Open files | `filebeat.harvester.open_files` | Direct |

### 16.2 OTel Collector Status APIs

#### zpages Extension
- **URL:** `http://localhost:55679`
- **Requires:** `zpages` extension enabled in config
- **Pages:**
  - `/debug/servicez` — service status overview
  - `/debug/pipelinez` — pipeline topology and component status
  - `/debug/extensionz` — extension status
  - `/debug/featurez` — feature gate status
  - `/debug/tracez` — recent trace spans (for debugging)

**`/debug/pipelinez` response (HTML, needs scraping or use internal Go types):**
Shows each pipeline with its receivers, processors, and exporters, plus a status badge per component. The internal data structure (accessible when embedding the library) is:

```go
// From go.opentelemetry.io/collector/service
type PipelineStatus struct {
    PipelineID component.ID
    Receivers  []ComponentStatus
    Processors []ComponentStatus
    Exporters  []ComponentStatus
}

type ComponentStatus struct {
    ID     component.ID
    Status component.Status  // StatusOK, StatusError, StatusRecoverableError, etc.
    Error  string
    Since  time.Time
}
```

#### health_check Extension
- **URL:** `http://localhost:13133/`
- **Requires:** `health_check` extension enabled
- **Response:**
```json
{
  "status": "Server available",        // or "Server not available"
  "upSince": "2025-01-15T10:30:00Z",
  "uptime": "72h15m30s"
}
```
- **With `v2` config enabled (`use_v2: true`):**
```json
{
  "status": "StatusOK",
  "component_statuses": {
    "pipeline:traces": {
      "status": "StatusOK",
      "timestamp": "2025-01-15T10:30:00Z",
      "components": {
        "receiver:otlp": {"status": "StatusOK"},
        "processor:batch": {"status": "StatusOK"},
        "exporter:elasticsearch": {"status": "StatusRecoverableError", "error": "connection timeout"}
      }
    }
  }
}
```

#### Prometheus Self-Telemetry
- **URL:** `http://localhost:8888/metrics`
- **Requires:** `service.telemetry.metrics.address` configured (default: `localhost:8888`)
- **Format:** Prometheus exposition format

**Key metrics for agent-cli:**

```prometheus
# Receiver metrics
otelcol_receiver_accepted_spans{receiver="otlp", transport="grpc"} 145230
otelcol_receiver_refused_spans{receiver="otlp", transport="grpc"} 3
otelcol_receiver_accepted_metric_points{receiver="hostmetrics"} 89400
otelcol_receiver_accepted_log_records{receiver="filelog"} 230100

# Processor metrics
otelcol_processor_accepted_spans{processor="batch"} 145227
otelcol_processor_dropped_spans{processor="filter"} 0
otelcol_processor_accepted_metric_points{processor="batch"} 89400
otelcol_processor_dropped_metric_points{processor="filter"} 1204

# Exporter metrics
otelcol_exporter_sent_spans{exporter="elasticsearch"} 145100
otelcol_exporter_send_failed_spans{exporter="elasticsearch"} 127
otelcol_exporter_sent_metric_points{exporter="elasticsearch"} 88196
otelcol_exporter_queue_size{exporter="elasticsearch"} 340
otelcol_exporter_queue_capacity{exporter="elasticsearch"} 5000

# Process metrics
otelcol_process_memory_rss 152043520
otelcol_process_cpu_seconds_total 45.23
otelcol_process_uptime 260130
```

**Metrics mapping for agent-cli:**

| agent-cli metric | Prometheus metric | Calculation |
|-----------------|-------------------|-------------|
| Events/sec (receiver) | `otelcol_receiver_accepted_*` | `rate()` over time window |
| Events/sec (exporter) | `otelcol_exporter_sent_*` | `rate()` over time window |
| Error rate (receiver) | `otelcol_receiver_refused_*` | `rate()` over time window |
| Drop rate (processor) | `otelcol_processor_dropped_*` | `rate()` over time window |
| Export failures | `otelcol_exporter_send_failed_*` | `rate()` over time window |
| Queue pressure | `queue_size / queue_capacity` | Percentage |
| Memory | `otelcol_process_memory_rss` | Direct (bytes) |
| CPU | `otelcol_process_cpu_seconds_total` | `rate()` = CPU cores used |
| Uptime | `otelcol_process_uptime` | Direct (seconds) |

---

## 17. Go Interface Design

> Core abstractions that the entire agent-cli codebase will be built around. These must support both Elastic Agent and OTel Collector from day one.

### 17.1 Core Types

```go
package agentcli

import "time"

// ──── Signal Type ─────────────────────────────────

type Signal string

const (
    SignalLogs    Signal = "logs"
    SignalMetrics Signal = "metrics"
    SignalTraces  Signal = "traces"
)

// ──── Health ──────────────────────────────────────

type HealthStatus string

const (
    StatusHealthy  HealthStatus = "healthy"
    StatusDegraded HealthStatus = "degraded"
    StatusError    HealthStatus = "error"
    StatusDisabled HealthStatus = "disabled"
    StatusUnknown  HealthStatus = "unknown"
)

type Health struct {
    Status  HealthStatus
    Message string          // human-readable reason
    Since   time.Time       // when this status was last observed
}

// ──── Metrics ─────────────────────────────────────

type Metrics struct {
    EventsInPerSec  float64  // throughput into the component
    EventsOutPerSec float64  // throughput out of the component
    ErrorsPerSec    float64  // errors per second
    DroppedPerSec   float64  // events dropped per second
    MemoryBytes     int64    // memory usage (0 if unknown)
    CPUPercent      float64  // CPU usage (0 if unknown)
    QueueUsage      float64  // 0.0–1.0 queue fullness (0 if N/A)
    Custom          map[string]interface{} // agent-specific extras
}

// ──── Pipeline Components ─────────────────────────

type ComponentKind string

const (
    KindInput     ComponentKind = "input"      // receiver in OTel
    KindProcessor ComponentKind = "processor"
    KindOutput    ComponentKind = "output"      // exporter in OTel
    KindConnector ComponentKind = "connector"   // OTel only
    KindExtension ComponentKind = "extension"   // OTel only
)

type Component struct {
    ID       string            // unique ID (e.g., "otlp", "filestream-default")
    Name     string            // display name
    Kind     ComponentKind
    Type     string            // specific type (e.g., "filestream", "hostmetrics", "otlp")
    Health   Health
    Metrics  Metrics
    Config   map[string]interface{} // raw config for this component
    Children []Component       // sub-components (e.g., streams in EA, scrapers in hostmetrics)
}

// ──── Pipeline ────────────────────────────────────

type Pipeline struct {
    Signal     Signal
    Name       string          // pipeline name (e.g., "traces", "logs", "metrics/custom")
    Inputs     []Component     // receivers / inputs
    Processors []Component     // ordered list of processors
    Outputs    []Component     // exporters / outputs
    Connectors []Component     // connectors bridging to other pipelines (OTel only)
}

// ──── Agent Status (top-level) ────────────────────

type AgentType string

const (
    AgentElastic AgentType = "elastic-agent"
    AgentEDOT    AgentType = "edot"
    AgentOTel    AgentType = "otel"
)

type AgentInfo struct {
    Type      AgentType
    Version   string
    PID       int
    ConfigPath string
    Uptime    time.Duration
}

type AgentStatus struct {
    Agent      AgentInfo
    Health     Health            // overall agent health
    Pipelines  []Pipeline        // all data pipelines
    Extensions []Component       // OTel extensions / EA monitoring
    Issues     []Issue           // aggregated problems
    FetchedAt  time.Time         // when this snapshot was taken
}

type IssueSeverity string

const (
    SeverityError   IssueSeverity = "error"
    SeverityWarning IssueSeverity = "warning"
    SeverityInfo    IssueSeverity = "info"
)

type Issue struct {
    Severity    IssueSeverity
    Component   string         // which component this relates to
    Pipeline    string         // which pipeline (empty if agent-level)
    Message     string
}
```

### 17.2 Agent Interface

```go
package agentcli

import "context"

// Agent is the core interface that each agent adapter must implement.
// It abstracts the differences between Elastic Agent, EDOT, and OTel Collector.
type Agent interface {
    // Type returns the agent type (elastic-agent, edot, otel).
    Type() AgentType

    // Info returns basic agent metadata (version, PID, config path).
    Info(ctx context.Context) (AgentInfo, error)

    // Status returns a full pipeline status snapshot.
    // This is the main method — it combines config parsing, API queries,
    // and metrics collection into a single AgentStatus.
    Status(ctx context.Context) (AgentStatus, error)

    // ParseConfig reads and validates the on-disk configuration file.
    // Returns pipelines as configured (before runtime state enrichment).
    // Also returns any config validation issues.
    ParseConfig(ctx context.Context) ([]Pipeline, []Issue, error)

    // Health returns just the overall agent health (lightweight check).
    Health(ctx context.Context) (Health, error)
}
```

### 17.3 Discovery Interface

```go
package discovery

import "context"

// DiscoveredAgent represents an agent found on the host.
type DiscoveredAgent struct {
    Type       agentcli.AgentType
    PID        int
    ConfigPath string
    Endpoints  []string   // API endpoints found (e.g., "http://localhost:6791")
    Source     string     // how it was found: "process", "path", "port"
}

// Discoverer finds agents running on the local host.
type Discoverer interface {
    // Discover scans for all agents on the host using all strategies.
    Discover(ctx context.Context) ([]DiscoveredAgent, error)
}

// Strategy is a single discovery method.
type Strategy interface {
    Name() string
    Discover(ctx context.Context) ([]DiscoveredAgent, error)
}
```

### 17.4 Adapter Implementations (Sketch)

```go
// internal/agent/elasticagent/adapter.go
type ElasticAgentAdapter struct {
    configPath string
    statusURL  string   // e.g., "http://localhost:6791"
}

func (a *ElasticAgentAdapter) Type() agentcli.AgentType { return agentcli.AgentElastic }

func (a *ElasticAgentAdapter) ParseConfig(ctx context.Context) ([]agentcli.Pipeline, []agentcli.Issue, error) {
    // 1. Read elastic-agent.yml from disk
    // 2. Parse YAML into internal structure
    // 3. Map inputs → pipelines by inferring signal from input type
    // 4. Wire inputs to outputs via use_output field
    // 5. Attach per-stream processors
    // 6. Run validation checks (missing outputs, duplicate IDs, etc.)
    // 7. Return []Pipeline and []Issue
}

func (a *ElasticAgentAdapter) Status(ctx context.Context) (agentcli.AgentStatus, error) {
    // 1. ParseConfig() for baseline pipeline structure
    // 2. Query /api/status for runtime component health
    // 3. Query beat /stats endpoints for per-component metrics
    // 4. Merge config + runtime + metrics into AgentStatus
    // 5. Compute derived issues (queue pressure, error rates, etc.)
}

// internal/agent/otel/adapter.go
type OTelAdapter struct {
    configPath  string
    zpagesURL   string   // e.g., "http://localhost:55679"
    healthURL   string   // e.g., "http://localhost:13133"
    metricsURL  string   // e.g., "http://localhost:8888"
}

func (a *OTelAdapter) ParseConfig(ctx context.Context) ([]agentcli.Pipeline, []agentcli.Issue, error) {
    // 1. Read config.yaml from disk
    // 2. Parse YAML into OTel config structure
    // 3. Read service.pipelines for explicit wiring
    // 4. Map receivers/processors/exporters to Pipeline model
    // 5. Detect orphaned components (defined but not in any pipeline)
    // 6. Detect connector bridging between pipelines
    // 7. Run validation checks
}

func (a *OTelAdapter) Status(ctx context.Context) (agentcli.AgentStatus, error) {
    // 1. ParseConfig() for baseline pipeline structure
    // 2. Query health_check v2 for per-component status
    // 3. Scrape Prometheus /metrics for throughput, errors, queue stats
    // 4. Optionally query zpages for additional detail
    // 5. Merge into AgentStatus
}
```

### 17.5 Interface Design Rationale

- **`Status()` is the main method** — it returns everything the TUI and CLI need in one call. Adapters handle all the complexity internally.
- **`ParseConfig()` is separate** because Phase 1 can start with *just* config parsing (no live APIs needed). This is the natural first implementation step.
- **`Health()` is lightweight** — useful for quick checks and the `discover` command without pulling full metrics.
- **`Pipeline` is the universal model** — both Elastic Agent's implicit wiring and OTel's explicit `service.pipelines` map to the same structure. The adapters handle the translation.
- **`Component.Children`** handles nested structures (EA streams inside an input, OTel scrapers inside hostmetrics).
- **`Issue` is first-class** — validation problems, runtime errors, and warnings are all surfaced consistently regardless of agent type.

---

## 18. Competitive Landscape & Prior Art

### 18.1 Existing Tools

| Tool | What It Does | Relevance to agent-cli |
|------|-------------|----------------------|
| **`elastic-agent status`** | Built-in CLI command. Shows component health in a flat list. No pipeline view, no metrics, no TUI. | Our baseline — we need to do everything this does, better, plus pipeline visualization and metrics. |
| **`otelcol validate`** | Built-in OTel subcommand. Validates config syntax only — no runtime status. | We should incorporate equivalent validation in `ParseConfig()`. |
| **`otelcol components`** | Lists available components in the binary. No pipeline or status info. | Useful reference for component type detection. |
| **[otel-tui](https://github.com/ymtdzzz/otel-tui)** | Bubbletea-based TUI for *observing OTLP data flowing through a collector* in real time. Focuses on trace/metric/log payloads, not agent config or health. | Great UX reference for our TUI. Same tech stack (Bubbletea). Different purpose — they show data content, we show pipeline health. |
| **[otelbin.io](https://www.otelbin.io/)** | Web-based OTel config visualizer and validator. Paste YAML → see pipeline diagram. | Closest to our config visualization goal, but web-only, no runtime status, no Elastic Agent support. Strong UX reference for pipeline rendering. |
| **[otel-config-validator](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/configschema)** | JSON Schema generator for OTel config. | Could be used for our OTel config validation. |
| **Elastic Fleet UI** | Kibana-based UI for managing Fleet-managed agents. Shows agent status, policies, integrations. | Rich but web-only, requires Kibana + Fleet Server. No standalone/OTel support. |
| **[k9s](https://github.com/derailed/k9s)** | TUI for Kubernetes cluster management. Not directly related but excellent UX reference for navigable TUI dashboards. | UX inspiration for navigation, filtering, drill-down patterns. |
| **[lazydocker](https://github.com/jesseduffield/lazydocker)** | TUI for Docker management. | UX inspiration for the three-panel layout and status indicators. |
| **[grpcui](https://github.com/fullstorydev/grpcui)** | Web UI for gRPC services. | Potential reference for interacting with EA's gRPC control plane. |

### 18.2 Key Takeaways

**No existing tool does what agent-cli aims to do.** The gap is:
- No unified tool works across Elastic Agent, EDOT, and vanilla OTel
- Existing OTel tools (otel-tui, otelbin) focus on either data payloads or config syntax — not pipeline health with metrics
- `elastic-agent status` is flat and text-only — no pipeline visualization, no per-component metrics
- Fleet UI requires a full Kibana deployment — agent-cli works from any terminal

**UX references to study:**
- **otelbin.io** — pipeline diagram rendering (how they draw receivers → processors → exporters)
- **otel-tui** — Bubbletea patterns for real-time telemetry display
- **k9s** — navigation model (resource list → detail view → logs), keyboard shortcuts, filtering
- **lazydocker** — multi-panel layout, status colors, compact information density

### 18.3 Differentiation

agent-cli's unique value proposition:

1. **Multi-agent** — one tool for EA, EDOT, and OTel (no competitor does this)
2. **Pipeline-first** — visualizes the actual data flow, not just a list of components
3. **Config + runtime** — combines on-disk config analysis with live health and metrics
4. **Terminal-native** — works over SSH, in containers, on headless servers (no browser needed)
5. **Scriptable** — JSON/YAML output for automation alongside the TUI
6. **Phase 2 editing** — guided config modification (no competitor offers this for OTel)

---

## 19. Log-Based Diagnostics Research

> agent-cli should be able to read agent/collector logs from disk and surface common problems with actionable guidance. This turns agent-cli from a status viewer into a diagnostic tool.

### 19.1 Log File Locations

| Agent | Default Log Path | Format |
|-------|-----------------|--------|
| **Elastic Agent** | `/opt/Elastic/Agent/data/elastic-agent-*/logs/elastic-agent-*.ndjson` | NDJSON (structured) |
| **EA Beat sub-processes** | `/opt/Elastic/Agent/data/elastic-agent-*/logs/default/filebeat-*.ndjson`, `metricbeat-*.ndjson` | NDJSON (structured) |
| **OTel Collector** | stdout/stderr (typically captured by systemd journal or Docker logs) | Structured text (`zap` logger) |
| **OTel (file output)** | Configurable via `service.telemetry.logs.output_paths` | JSON or text |

### 19.2 Elastic Agent — Common Log Errors & Patterns

#### Connection & Output Errors

| Pattern | Log Signature | Severity | Diagnosis | Suggested Fix |
|---------|--------------|----------|-----------|---------------|
| **Elasticsearch unreachable** | `"message":"Error publishing events"`, `"error":{"message":"connection refused"}` | ✗ Error | Output cannot reach ES cluster | Check ES is running, verify `outputs.*.hosts` URL and port, check firewall rules |
| **Authentication failure** | `"message":"Failed to connect"`, `"error":{"message":"401 Unauthorized"}` | ✗ Error | API key or credentials invalid | Regenerate API key, check `outputs.*.api_key` or `username`/`password` |
| **TLS handshake failure** | `"error":{"message":"x509: certificate signed by unknown authority"}` | ✗ Error | TLS cert not trusted | Add CA cert via `ssl.certificate_authorities`, or set `ssl.verification_mode: none` for testing |
| **Index/data stream issue** | `"error":{"message":"index_not_found_exception"}` or `"message":"data stream not found"` | ✗ Error | Target index or data stream doesn't exist | Check index templates are installed, verify ILM/data stream setup |
| **Bulk indexing rejection** | `"error":{"message":"circuit_breaking_exception"}` or `429 Too Many Requests` | ⚠ Degraded | ES is under pressure, rejecting writes | Scale ES cluster, reduce ingestion rate, check ES JVM heap |

#### Input & Harvester Errors

| Pattern | Log Signature | Severity | Diagnosis | Suggested Fix |
|---------|--------------|----------|-----------|---------------|
| **File not found** | `"message":"File not found"`, `"path":"/var/log/missing.log"` | ⚠ Warning | Configured path doesn't exist | Verify `paths` in input config, check file permissions |
| **Permission denied** | `"message":"Failed opening file"`, `"error":{"message":"permission denied"}` | ✗ Error | Agent can't read the target file | Run agent as root or fix file permissions, check SELinux/AppArmor |
| **Too many open files** | `"message":"Too many open file handlers"` or `"error":{"message":"too many open files"}` | ✗ Error | File descriptor limit hit | Increase `ulimit -n`, reduce `harvester_limit`, close idle harvesters |
| **Multiline misconfiguration** | `"message":"Multiline pattern compile error"` | ✗ Error | Invalid regex in multiline config | Fix `multiline.pattern` regex syntax |
| **Registry file corruption** | `"message":"Error loading state"` | ⚠ Warning | Filebeat registry corrupted | Delete registry file and restart (will re-read files from beginning) |

#### Agent Lifecycle Errors

| Pattern | Log Signature | Severity | Diagnosis | Suggested Fix |
|---------|--------------|----------|-----------|---------------|
| **Component crash loop** | `"message":"Component stopped unexpectedly"` repeated within short interval | ✗ Error | Beat sub-process crashing and restarting | Check component-specific logs for root cause, look for OOM or panic |
| **Fleet enrollment failure** | `"message":"fail to enroll"`, `"error":{"message":"unauthorized"}` | ✗ Error | Can't connect to Fleet Server | Verify Fleet URL and enrollment token, check Fleet Server is running |
| **Policy update failure** | `"message":"fail to update"`, `"fleet.policy.id":"..."` | ⚠ Warning | Agent connected to Fleet but can't apply policy | Check policy exists in Kibana, verify agent version compatibility |
| **Configuration error** | `"message":"Cannot parse configuration"`, `"error.message":"yaml: ..."` | ✗ Error | Malformed YAML in config file | Fix YAML syntax at reported line number |
| **Port already in use** | `"message":"listen tcp :8200: bind: address already in use"` | ✗ Error | Another process using the same port | Find conflicting process with `lsof -i :PORT`, change port in config |

### 19.3 OTel Collector — Common Log Errors & Patterns

#### Exporter Errors

| Pattern | Log Signature | Severity | Diagnosis | Suggested Fix |
|---------|--------------|----------|-----------|---------------|
| **Backend unreachable** | `"msg":"Exporting failed. Will retry..."`, `"error":"connection refused"` | ✗ Error | Exporter can't reach backend | Verify `endpoint` URL, check network, confirm backend is running |
| **Auth failure** | `"msg":"Exporting failed"`, `"error":"Unauthenticated"` or `"401"` | ✗ Error | Credentials rejected by backend | Check API key, token, or certificate configuration |
| **Timeout** | `"msg":"Exporting failed"`, `"error":"context deadline exceeded"` | ⚠ Degraded | Export taking too long | Increase timeout, check backend latency, reduce batch size |
| **Retry exhaustion** | `"msg":"Exporting failed. No more retries left"` | ✗ Error | All retry attempts failed, data being dropped | Fix underlying connection issue, check `retry_on_failure` config |
| **Queue overflow** | `"msg":"Dropping data because sending_queue is full"` | ✗ Error | Export queue full, data loss occurring | Increase `sending_queue.queue_size`, fix slow exporter, scale backend |
| **TLS error** | `"msg":"Exporting failed"`, `"error":"tls: ..."` or `"x509: ..."` | ✗ Error | TLS certificate issue | Fix cert paths, add CA cert, check cert expiration |

#### Receiver Errors

| Pattern | Log Signature | Severity | Diagnosis | Suggested Fix |
|---------|--------------|----------|-----------|---------------|
| **Port conflict** | `"msg":"Failed to start receiver"`, `"error":"bind: address already in use"` | ✗ Error | Another process on same port | Change port or stop conflicting process |
| **Failed scrape** | `"msg":"Error scraping"`, `"scraper":"..."` | ⚠ Warning | hostmetrics or prometheus scraper failing | Check target availability, permissions, scraper config |
| **File permission** | `"msg":"Failed to open file"`, `"error":"permission denied"` | ✗ Error | filelog receiver can't read files | Fix file permissions, run as appropriate user |
| **Regex parse error** | `"msg":"Failed to process entry"`, `"operator":"regex_parser"` | ⚠ Warning | Log line doesn't match regex pattern | Fix `regex` pattern or add `on_error: send` to not drop unmatched |
| **OTLP decode error** | `"msg":"Failed to decode message"` | ⚠ Warning | Malformed OTLP data from client | Check client SDK version and configuration |

#### Processor Errors

| Pattern | Log Signature | Severity | Diagnosis | Suggested Fix |
|---------|--------------|----------|-----------|---------------|
| **OTTL parse error** | `"msg":"Failed to parse OTTL statement"` | ✗ Error | Invalid transform language syntax | Fix OTTL statement syntax in `transform` processor |
| **Filter match error** | `"msg":"Failed to match filter"`, `"error_mode":"propagate"` | ✗ Error | Filter expression evaluation failed | Fix filter expression, or set `error_mode: ignore` |
| **Memory limit hit** | `"msg":"Memory usage exceeded limit"`, `"processor":"memory_limiter"` | ⚠ Degraded | Memory limiter is throttling/dropping data | Increase `limit_mib` or reduce incoming data volume |
| **Batch timeout** | `"msg":"Batch timeout reached, sending incomplete batch"` (at high frequency) | ⚠ Info | Batches not filling before timeout | Normal at low volume; at high volume, increase `send_batch_size` |

#### Collector Lifecycle Errors

| Pattern | Log Signature | Severity | Diagnosis | Suggested Fix |
|---------|--------------|----------|-----------|---------------|
| **Config validation failure** | `"msg":"Cannot convert config"` or `"msg":"Invalid configuration"` | ✗ Error | Config YAML is valid but semantically wrong | Check component names, field types, required fields |
| **Unknown component** | `"msg":"Factory not found"`, `"component":"..."` | ✗ Error | Component referenced in config isn't compiled into this binary | Use a collector distribution that includes the component, or build custom |
| **Extension start failure** | `"msg":"Failed to start extension"` | ⚠ Warning | Extension (zpages, health_check) couldn't start | Check port conflicts, permissions |
| **OOM killed** | (no log — sudden process termination, check `dmesg` or journal) | ✗ Error | Collector ran out of memory | Add `memory_limiter` processor, increase container memory limit |
| **Panic/crash** | `"msg":"panic"` or Go stack trace in logs | ✗ Error | Bug in collector or component | Report to upstream, check if component version is known-buggy |

### 19.4 Log Parsing Strategy for agent-cli

#### Approach
```
agent-cli diagnose                         # scan logs and report issues
agent-cli diagnose --since 1h             # only last hour
agent-cli diagnose --severity error       # only errors
agent-cli status --with-logs              # include log-based issues in status view
```

#### Implementation
1. **Locate log files** — Use discovery to find agent, then check well-known log paths and process stdout/journal
2. **Tail recent entries** — Default to last 1000 lines or last 1 hour (configurable)
3. **Pattern matching** — Match against known error signatures (regex-based pattern library)
4. **Deduplication** — Group repeated occurrences of the same error (e.g., "connection refused" x 347 in last hour)
5. **Severity classification** — Map each matched pattern to error/warning/info
6. **Actionable output** — Each matched issue includes: what happened, why it matters, and what to do

#### Pattern Library Structure
```go
type LogPattern struct {
    ID          string            // e.g., "ea-output-connection-refused"
    Agent       AgentType         // which agent this applies to (or "all")
    Component   ComponentKind     // input, processor, output, lifecycle
    Severity    IssueSeverity
    Pattern     *regexp.Regexp    // what to match in log lines
    Fields      []string          // structured fields to extract (NDJSON)
    Title       string            // "Elasticsearch output unreachable"
    Description string            // "The agent cannot connect to..."
    Suggestion  string            // "Check ES is running, verify hosts..."
}
```

#### Integration with Pipeline View
Log-based issues merge into the existing `Issue` type and appear in the TUI's ISSUES section:

```
 ─────────────────────────────────────────────────────────────────────────────────
  ISSUES (4)                                                     [e] errors only
  ✗ elasticsearch exporter: connection refused (x347 in last 1h)
    → Check ES is running at localhost:9200, verify firewall rules
  ✗ filelog receiver: permission denied on /var/log/secure
    → Run collector as root or grant read access to the otel user
  ⚠ batch processor: queue at 89% — backpressure risk
    → Increase sending_queue.queue_size or fix slow exporter
  ⚠ hostmetrics scraper: intermittent scrape failures (x12 in last 1h)
    → Check if /proc is accessible, verify scraper permissions
```

### 19.5 Log Diagnostic Validation Checks Summary

| Category | # Patterns (EA) | # Patterns (OTel) | Notes |
|----------|-----------------|-------------------|-------|
| Output/Exporter errors | 5 | 6 | Connection, auth, TLS, timeouts, queue overflow |
| Input/Receiver errors | 5 | 5 | Files, permissions, ports, parsing |
| Processor errors | — | 4 | OTTL, filters, memory limiter, batching |
| Agent lifecycle | 5 | 5 | Crashes, config errors, port conflicts, OOM |
| **Total** | **15** | **20** | Extensible pattern library |

> This pattern library should be shipped as a data file or embedded Go map so it can be updated independently of the binary logic. Community contributions for new patterns should be straightforward.

---

## 20. References

- [Elastic Agent docs](https://www.elastic.co/guide/en/fleet/current/elastic-agent-configuration.html)
- [OpenTelemetry Collector docs](https://opentelemetry.io/docs/collector/)
- [EDOT Collector](https://github.com/elastic/elastic-agent)
- [Cobra CLI](https://github.com/spf13/cobra)
- [Bubbletea TUI](https://github.com/charmbracelet/bubbletea)
- [Lipgloss styling](https://github.com/charmbracelet/lipgloss)
- [Bubbles components](https://github.com/charmbracelet/bubbles)
- [Huh forms](https://github.com/charmbracelet/huh)
- [GoReleaser](https://goreleaser.com/)
