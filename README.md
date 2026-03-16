# Meridian

Prove telemetry flows before you merge.

Meridian is a local-first CLI and GitHub Action for reviewing OpenTelemetry Collector configuration changes. It combines repo-side validation, Collector-native semantic checks, graphing, diff-aware review hints, deterministic runtime execution, assertions and contracts, and durable artifacts for local debugging and PR review.

Documentation now lives in the versioned MkDocs site:

- Published docs: [salman-frs.github.io/meridian](https://salman-frs.github.io/meridian/)
- Source docs: [docs/](docs/)
- CLI reference: [docs/reference/cli/](docs/reference/cli/)

## Quickstart

```bash
go build -o ./bin/meridian ./cmd/meridian
./bin/meridian validate -c examples/basic/collector.yaml
./bin/meridian check -c examples/basic/collector.yaml
```

Artifacts are written under `./meridian-artifacts/runs/<run_id>/` by default, with `./meridian-artifacts/runs/latest/` pointing at the most recent runtime run.

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

## Highlights

- `safe` mode is the default runtime path
- `auto` prefers Docker and falls back to containerd-backed flows when supported
- semantic validation can use `--collector-binary` or a Collector image
- runtime bundles preserve `report.json`, `summary.md`, `config.patched.yaml`, captures, and related evidence artifacts
- the repo ships a canonical k3s regression fixture under [`examples/k3s-e2e/`](examples/k3s-e2e/)

## GitHub Action

Use the composite action locally with `uses: ./action` or externally with `uses: salman-frs/meridian/action@v1`.

See the docs site for the full action contract and CI setup guides.

## Development

```bash
go test ./...
go run ./cmd/meridian-docs
```

For documentation work:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r docs/requirements.txt
mkdocs serve
```
