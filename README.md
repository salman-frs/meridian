# Meridian

Prove telemetry flows before you merge.

Meridian is a local-first CLI and GitHub Action for OpenTelemetry Collector configs. It makes config changes reviewable and safer by combining:

- static validation
- pipeline graph generation
- diff-aware risk hints
- deterministic runtime checks against an ephemeral Collector
- artifacted reports for local debugging and PR review

## Quickstart

Build the CLI:

```bash
go build -o ./bin/meridian ./cmd/meridian
```

Run validation:

```bash
./bin/meridian validate -c examples/basic/collector.yaml
```

Run the full safe-mode check:

```bash
./bin/meridian check -c examples/basic/collector.yaml
```

Artifacts are written under `./meridian-artifacts/runs/<run_id>/`.

Install a release binary:

```bash
curl -fsSL -o meridian.tar.gz https://github.com/salman-frs/meridian/releases/latest/download/meridian-linux-amd64.tar.gz
tar -xzf meridian.tar.gz
./meridian version
```

## Commands

- `meridian validate`
- `meridian graph`
- `meridian diff`
- `meridian test`
- `meridian check`
- `meridian ci`
- `meridian debug logs|capture|bundle`
- `meridian version`
- `meridian completion`

## Safe Mode

`safe` mode is the default. Meridian patches the target config so synthetic telemetry is injected through an OTLP receiver and exported only to Meridian-managed capture infrastructure instead of real destinations.

What safe mode validates:

- config parses and wires correctly
- processors still allow telemetry to pass through
- telemetry for enabled signal types reaches the capture sink
- custom output assertions still match captured telemetry

What safe mode does not validate:

- real exporter connectivity
- vendor backend authentication
- live production routing outside the patched test harness

Meridian always injects a Meridian-managed OTLP receiver for deterministic runtime input and persists bounded sample captures by default instead of full payload dumps.

Use `tee` or `live` only when you intentionally want more realistic destination behavior.

## GitHub Action

Minimal usage:

```yaml
name: meridian

on:
  pull_request:
    paths:
      - "otel/**"
      - "**/collector*.yaml"
      - "**/otelcol*.yaml"

jobs:
  meridian:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
    steps:
      - uses: actions/checkout@v4
      - uses: ./action
        with:
          config: examples/basic/collector.yaml
```

Published usage:

```yaml
      - uses: salman-frs/meridian/action@v1
        with:
          config: examples/basic/collector.yaml
```

More setup details live in [docs/ci-github-actions.md](docs/ci-github-actions.md).

## Docs

- [install.md](docs/install.md)
- [ci-github-actions.md](docs/ci-github-actions.md)
- [assertions.md](docs/assertions.md)
- [troubleshooting.md](docs/troubleshooting.md)
- [config-patching.md](docs/config-patching.md)

## Examples

- [basic](examples/basic/collector.yaml)
- [routing](examples/routing/collector.yaml)
- [redaction](examples/redaction/collector.yaml)
- [multipipeline](examples/multipipeline/collector.yaml)
