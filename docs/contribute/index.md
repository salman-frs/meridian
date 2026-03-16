# Contributor Guide

## Development loop

```bash
go test ./...
go build ./...
```

When changing user-facing CLI help or commands, regenerate the docs:

```bash
go run ./cmd/meridian-docs
```

## Code ownership guidelines

- keep validation rules in `internal/validate`
- keep capture behavior in `internal/capture`
- keep telemetry generation in `internal/generator`
- keep runtime orchestration in `internal/runtime`
- keep reporting renderers pure where possible

## Reproducing CI failures

- download the artifact bundle
- inspect `report.json` and `collector.log`
- rerun locally with `--keep-containers`

## Docs work

See [Docs Authoring](docs.md) for the local MkDocs workflow and [Release Docs Workflow](releases.md) for publishing behavior.
