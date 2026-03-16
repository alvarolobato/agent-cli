# agent-cli

`agent-cli` is a Go CLI/TUI for inspecting observability agent pipelines and health.

Supported targets today:

- Elastic Agent
- EDOT Collector
- Generic OpenTelemetry Collector

## Project Description

`agent-cli` is a terminal-first operational tool for understanding how telemetry is flowing through local agent pipelines and where it is failing.

It combines static configuration parsing with live runtime signals so operators can:

- inspect pipeline topology (`inputs/receivers -> processors -> outputs/exporters`)
- assess normalized component health (`healthy`, `degraded`, `error`, `disabled`, `unknown`)
- view key throughput and failure indicators (events, drops, send failures)
- spot misconfigurations such as missing wiring or disabled/orphaned components
- use both human-oriented terminal views (table/TUI) and machine-friendly output (`--format json`)

The codebase uses a layered adapter architecture so each agent type maps into a shared pipeline model:

- Elastic Agent adapter
- EDOT adapter
- Generic OTel adapter

## Project Status

- **Merged:** Phase 0, Phase 1A, Phase 1B, Phase 1C
- **Planned:** Phase 1D, Phase 1E, Phase 2

Roadmap issue: [#2](https://github.com/alvarolobato/agent-cli/issues/2)

## Install

Prerequisites: Go `1.24+`

```bash
git clone https://github.com/alvarolobato/agent-cli.git
cd agent-cli
make build
./bin/agent-cli --help
```

Alternative without Make:

```bash
go build -o bin/agent-cli ./cmd/agent-cli
./bin/agent-cli --help
```

## Quick Start

```bash
agent-cli discover
agent-cli status
agent-cli status --format json
agent-cli tui
```

## Common Usage

Elastic Agent:

```bash
agent-cli status --agent elastic-agent
agent-cli status --agent elastic-agent --format json
```

EDOT:

```bash
agent-cli status --agent edot --edot-config /etc/edot/config.yaml
```

OTel:

```bash
agent-cli status --agent otel --otel-config /etc/otelcol/config.yaml
```

For full command options:

```bash
agent-cli --help
agent-cli status --help
agent-cli discover --help
agent-cli tui --help
```

## Contributing

1. Pick or create a scoped issue.
2. Create a branch from `main`.
3. Implement with tests.
4. Run checks:

```bash
go build ./...
go test -race ./...
go vet ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run
```

5. Open a PR referencing the issue (`Closes #<issue-number>`).

## Additional Documentation

Deep-dive usage, architecture overview, troubleshooting, and implementation links:

- [`docs/usage-and-troubleshooting.md`](docs/usage-and-troubleshooting.md)
- [`architecture.md`](architecture.md)
- [`decision-log.md`](decision-log.md)
- [`project-requirements.md`](project-requirements.md)
- [`AGENTS.md`](AGENTS.md)

## License

No license file is currently present in the repository.
