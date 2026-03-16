# Usage and Troubleshooting

This document keeps extended operational details out of the top-level `README.md`.

## Architecture Snapshot

`agent-cli` follows a layered model:

1. CLI/TUI presentation (`internal/cli`, `internal/tui`)
2. Shared pipeline model and health normalization (`internal/pipeline`)
3. Agent adapters (`internal/agent/elasticagent`, `internal/agent/edot`, `internal/agent/otel`)
4. Config + metrics collection (`internal/config`, `internal/metrics`)
5. Output rendering (`internal/output`)

See [`architecture.md`](../architecture.md) for full responsibilities and data flow.

## Agent-Specific Usage

### Elastic Agent

```bash
agent-cli status --agent elastic-agent
agent-cli status --agent elastic-agent --format json
agent-cli status --agent elastic-agent --elastic-config /opt/Elastic/Agent/elastic-agent.yml
agent-cli status --agent elastic-agent --elastic-url http://127.0.0.1:6791
agent-cli tui --agent elastic-agent
```

### EDOT

```bash
agent-cli status --agent edot \
  --edot-config /etc/edot/config.yaml \
  --edot-zpages-url http://127.0.0.1:55679 \
  --edot-health-url http://127.0.0.1:13133/ \
  --edot-metrics-url http://127.0.0.1:8888/metrics

agent-cli tui --agent edot --edot-config /etc/edot/config.yaml
```

### Generic OTel Collector

```bash
agent-cli status --agent otel \
  --otel-config /etc/otelcol/config.yaml \
  --otel-zpages-url http://127.0.0.1:55679 \
  --otel-health-url http://127.0.0.1:13133/ \
  --otel-metrics-url http://127.0.0.1:8888/metrics
```

## TUI Keys (Current)

- `q` / `ctrl+c`: quit
- `up` / `down`: move selection
- `r`: refresh trigger placeholder

Live mode:

```bash
agent-cli tui --live --refresh 5s
```

## Troubleshooting

### `status` shows only example pipeline

Cause: no `--agent` was passed.

Fix:

```bash
agent-cli status --agent elastic-agent
```

### `elastic agent config not found; pass --elastic-config`

Cause: default config path not found on this host.

Fix: pass the explicit config path.

```bash
agent-cli status --agent elastic-agent --elastic-config /path/to/elastic-agent.yml
```

### `edot config not found; pass --edot-config`

Cause: EDOT requires explicit config path.

Fix:

```bash
agent-cli status --agent edot --edot-config /etc/edot/config.yaml
```

### `otel config not found; pass --otel-config`

Cause: OTel requires explicit config path.

Fix:

```bash
agent-cli status --agent otel --otel-config /etc/otelcol/config.yaml
```

### zPages / health / metrics endpoint errors

Cause: collector endpoints are disabled, bound to a different host/port, or blocked.

Checks:

- zPages: `http://127.0.0.1:55679` (or custom)
- health_check: `http://127.0.0.1:13133/` (or custom)
- metrics: `http://127.0.0.1:8888/metrics` (or custom)

Fix: pass explicit URLs that match your collector config.

### Integration tests are skipped

Cause: integration tests require Docker and `RUN_INTEGRATION=1`.

Fix:

```bash
cd test/integration
docker compose up -d
cd ../..
RUN_INTEGRATION=1 go test ./test/integration/... -count=1
```

## More References

- Requirements: [`project-requirements.md`](../project-requirements.md)
- Decision history: [`decision-log.md`](../decision-log.md)
- Active/planned specs: [GitHub issues](https://github.com/alvarolobato/agent-cli/issues)
