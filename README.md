# Meridian

Prove telemetry flows before you merge.

Meridian is a local-first CLI and GitHub Action for reviewing OpenTelemetry Collector configuration changes. It combines repo-side validation, Collector-native semantic validation, graph generation, diff-aware review hints, deterministic runtime execution, assertions and contracts, and durable artifacts for local debugging and pull request review.

## Why Meridian exists

Collector configuration is operationally important and difficult to review with confidence.

A config can be:

- syntactically valid but unsupported by the selected Collector distribution
- structurally plausible but semantically wrong
- accepted by static checks while still dropping or misrouting telemetry at runtime
- impossible for a reviewer to reason about from YAML alone

Meridian is designed to give engineers a stronger pre-merge loop by producing evidence, not just terminal output.

## What Meridian proves

Meridian is designed to prove:

- the config can be loaded and analyzed repo-side when sources are locally materializable
- the selected Collector binary or image can validate the config semantically
- topology changes can be reviewed through graph and diff artifacts
- synthetic telemetry can traverse the configured path in a deterministic runtime harness
- assertions and contracts can be evaluated against captured output
- reviewers can inspect durable artifacts after the command finishes

## What Meridian does not prove

Meridian does not replace production validation.

In the default `safe` mode, it does not prove:

- live vendor backend authentication
- production ingestion correctness
- generic Kubernetes compatibility outside the repo-owned fixture

## Quickstart

Build the CLI:

```bash
go build -o ./bin/meridian ./cmd/meridian
```

Run static validation:

```bash
./bin/meridian validate -c examples/basic/collector.yaml
```

Run the recommended local confidence workflow:

```bash
./bin/meridian check -c examples/basic/collector.yaml
```

Inspect the latest bundle:

```bash
./bin/meridian debug summary
./bin/meridian debug bundle --format json
```

By default, runtime artifacts are written under `./meridian-artifacts/runs/<run_id>/`, with `./meridian-artifacts/runs/latest/` pointing to the most recent runtime run.

## Commands

- `meridian validate`
- `meridian graph`
- `meridian diff`
- `meridian test`
- `meridian check`
- `meridian ci`
- `meridian debug logs|capture|summary|bundle`
- `meridian version`
- `meridian completion`

## Runtime model

- `safe` is the default mode and the recommended review path
- `tee` preserves original exporters and appends Meridian capture
- `live` may send synthetic telemetry to real destinations and should be used deliberately
- `auto` prefers Docker and falls back to containerd-backed flows when supported

## GitHub Action

Use the composite action locally with `uses: ./action` or externally with `uses: salman-frs/meridian/action@v1`.

Minimal example:

```yaml
jobs:
  meridian:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
    steps:
      - uses: actions/checkout@v4
      - uses: salman-frs/meridian/action@v1
        with:
          config: examples/basic/collector.yaml
          engine: auto
```

## Documentation

Documentation lives in the versioned MkDocs site:

- Published docs: [salman-frs.github.io/meridian](https://salman-frs.github.io/meridian/)
- Source docs: [docs/](docs/)

Recommended entry points:

- product overview: [docs/overview/index.md](docs/overview/index.md)
- getting started: [docs/getting-started/index.md](docs/getting-started/index.md)
- core concepts: [docs/core-concepts/index.md](docs/core-concepts/index.md)
- feature deep dives: [docs/features/index.md](docs/features/index.md)
- architecture: [docs/architecture/index.md](docs/architecture/index.md)
- generated CLI reference: [docs/reference/cli/index.md](docs/reference/cli/index.md)

## Repository highlights

- semantic validation can use `--collector-binary` or a Collector image
- runtime bundles preserve `report.json`, `summary.md`, `config.patched.yaml`, captures, and related evidence artifacts
- the repository ships a canonical k3s regression fixture under [`examples/k3s-e2e/`](examples/k3s-e2e/)

## Development

Run the Go test suite:

```bash
go test ./...
```

Regenerate CLI docs:

```bash
go run ./cmd/meridian-docs
```

Work on the docs site locally:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r docs/requirements.txt
mkdocs serve
```
