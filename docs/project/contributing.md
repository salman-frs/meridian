# Contributing

## Development loop

```bash
go test ./...
go build ./...
```

When command help text changes, regenerate the CLI docs:

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
