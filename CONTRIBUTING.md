# Contributing

## Development loop

Build the binary:

```bash
go build ./cmd/meridian
```

## Adding validation rules

- keep rules in `internal/validate`
- return structured `Finding` values with remediation and next-step text
- prefer high-signal checks over exhaustive linting

## Adding runtime adapters

- keep capture behavior in `internal/capture`
- keep telemetry emission behavior in `internal/generator`
- keep Docker/container orchestration in `internal/runtime`

## Reproducing CI failures

- download the artifact bundle
- inspect `report.json` and `collector.log`
- rerun locally with `--keep-containers`
