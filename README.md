# Meridian

Prove telemetry flows before you merge.

Meridian is a local-first CLI and GitHub Action for OpenTelemetry Collector configs. It makes config changes reviewable and safer by combining:

- static validation
- pipeline graph generation
- diff-aware risk hints
- deterministic runtime checks against an ephemeral Collector
- fixture-driven assertions and contracts
- artifacted reports for local debugging and PR review

Runtime commands support `--engine auto|docker|containerd`. `auto` prefers Docker when both are available and falls back to `nerdctl`-backed containerd on Linux or `lima nerdctl` on macOS.

For Kubernetes end-to-end validation, Meridian now ships a repo-owned k3s fixture under `examples/k3s-e2e/`. The official OpenTelemetry Demo is optional/manual only and is not the blocking acceptance gate for this repo anymore.

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

Run against containerd:

```bash
./bin/meridian check -c examples/basic/collector.yaml --engine containerd
```

On macOS, `--engine containerd` uses `lima nerdctl`. Direct `nerdctl` reuse of an OrbStack Docker context is not supported.

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

For Docker, Meridian routes capture traffic through a host alias. For Linux containerd, Meridian uses host networking via `nerdctl`. For macOS containerd, Meridian uses `lima nerdctl`, published injection ports, and `host.lima.internal` for the capture sink.

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
          engine: auto
```

Published usage:

```yaml
      - uses: salman-frs/meridian/action@v1
        with:
          config: examples/basic/collector.yaml
          engine: auto
```

More setup details live in [docs/ci-github-actions.md](docs/ci-github-actions.md).

## Docs

- [install.md](docs/install.md)
- [ci-github-actions.md](docs/ci-github-actions.md)
- [k3s-e2e.md](docs/k3s-e2e.md)
- [assertions.md](docs/assertions.md)
- [troubleshooting.md](docs/troubleshooting.md)
- [config-patching.md](docs/config-patching.md)

## Examples

- [basic](examples/basic/collector.yaml)
- [routing](examples/routing/collector.yaml)
- [redaction](examples/redaction/collector.yaml)
- [multipipeline](examples/multipipeline/collector.yaml)
- [k3s-e2e](examples/k3s-e2e/)

## K3s E2E

Run the repo-owned stack on the VM:

```bash
scripts/e2e_k3s_vm.sh happy
```

Available scenarios:

- `happy`
- `drop-traces`
- `misroute-logs`
- `auth-fail`
- `backend-unreachable`
