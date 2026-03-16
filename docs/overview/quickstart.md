# Quickstart

This is the shortest path to a meaningful Meridian run.

## Build the CLI

```bash
go build -o ./bin/meridian ./cmd/meridian
```

## Run static validation

```bash
./bin/meridian validate -c examples/basic/collector.yaml
```

What this does:

- loads local config sources when available
- interpolates env values from `--env-file`, `--env`, and the shell
- runs repo-side validation
- runs Collector-native semantic validation when a binary or image is available

## Run the opinionated confidence workflow

```bash
./bin/meridian check -c examples/basic/collector.yaml
```

This performs:

- semantic validation
- graph artifact generation
- runtime patching in `safe` mode
- synthetic telemetry injection
- capture-backed assertions
- markdown and JSON artifact generation

## Inspect the latest run

By default, runtime artifacts land under `./meridian-artifacts/runs/<run_id>/`, with `./meridian-artifacts/runs/latest/` pointing at the newest run.

Useful follow-ups:

```bash
./bin/meridian debug summary
./bin/meridian debug bundle --format json
./bin/meridian debug logs
./bin/meridian debug capture
```

## Next steps

- Learn the install and runtime prerequisites in [Install](install.md)
- Understand safe vs tee vs live in [Runtime Modes](../concepts/runtime-modes.md)
- See the full artifact contract in [Artifact Contract](../reference/artifact-contract.md)
