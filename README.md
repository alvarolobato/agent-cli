# agent-cli

`agent-cli` is a Go CLI/TUI for inspecting observability agent pipelines and health.

Supported targets today:

- Elastic Agent
- EDOT Collector
- Generic OpenTelemetry Collector

## Project Status

- **Merged:** Phase 0, Phase 1A, Phase 1B, Phase 1C
- **Planned:** Phase 1D, Phase 1E, Phase 2

Roadmap issue: [#2](https://github.com/alvarolobato/agent-cli/issues/2)

## Install

Prerequisites: Go `1.24+`

```bash
git clone https://github.com/alvarolobato/agent-cli.git
cd agent-cli
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
