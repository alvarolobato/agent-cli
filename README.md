# agent-cli

`agent-cli` is a cross-agent CLI and TUI for Elastic Agent, EDOT Collector, and OpenTelemetry Collector.

## Installation

### Download from GitHub Releases

Use binaries from the project's [GitHub Releases](https://github.com/alvarolobato/agent-cli/releases).

### Build from source

```bash
go build ./cmd/agent-cli
./agent-cli --help
```

### Docker

```bash
docker build -t agent-cli .
docker run --rm --net=host agent-cli --help
docker run --rm --net=host agent-cli status
```

## Quick Start

```bash
# Auto-detect or use defaults
agent-cli status

# Point directly to an install/config directory
agent-cli status --path /opt/Elastic/Agent
agent-cli status --path /etc/otelcol

# Launch interactive TUI
agent-cli tui --path /opt/Elastic/Agent
agent-cli tui --refresh 5s
```

## Commands

- `agent-cli status`: Pipeline status report (ASCII diagram + table, or JSON).
- `agent-cli tui`: Interactive dashboard with drill-down screens.
- `agent-cli discover`: Local discovery strategies summary.
- `agent-cli completion <shell>`: Shell completion scripts.

### `status` examples

```bash
agent-cli status --agent elastic-agent --elastic-config /opt/Elastic/Agent/elastic-agent.yml
agent-cli status --agent edot --edot-config /etc/edot/config.yaml
agent-cli status --agent otel --otel-config /etc/otelcol/config.yaml
agent-cli status --format json
```

### `tui` examples

```bash
agent-cli tui --agent elastic-agent
agent-cli tui --live
agent-cli tui --refresh 5s
```

## Supported Agents

- Elastic Agent (`elastic-agent`)
- EDOT Collector (`edot`)
- OpenTelemetry Collector (`otel`)

## Output Example (ASCII Pipeline)

```text
INPUTS                             PROCESSORS                         OUTPUTS
--------------------------------------------------------------------------------------------------------
✓ system-logs (in 120.0/s out 118.0/s err 0) ✓ batch (in 118.0/s out 118.0/s err 0) ✓ default (in 118.0/s out 118.0/s err 0)
```

## TUI Keyboard Shortcuts

- `up/down` or `left/right`: Move between dashboard columns
- `Enter`: Open detail screen for selected lane
- `e`: Open errors and warnings screen
- `c`: Open raw config screen
- `r`: Manual refresh timestamp update
- `l`: Toggle live mode
- `Esc` / `b`: Back to dashboard
- `q`: Quit

## Shell Completion

```bash
# bash
agent-cli completion bash > /etc/bash_completion.d/agent-cli

# zsh
source <(agent-cli completion zsh)

# fish
agent-cli completion fish | source

# powershell
agent-cli completion powershell > agent-cli.ps1
```

## Development

```bash
go build ./...
go test ./...
go vet ./...
golangci-lint run
```

## Contributing

1. Create a feature branch from `main`.
2. Run all checks locally.
3. Open a pull request that references the issue (`Closes #<issue>`).
4. Wait for CI and code review before merge.
