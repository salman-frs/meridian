# Contributing

## Development loop

```bash
go test ./...
go build ./...
```

## Docs workflow

Regenerate CLI reference pages whenever command help text or flags change:

```bash
go run ./cmd/meridian-docs
```

To work on the docs site locally:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r docs/requirements.txt
mkdocs serve
```

## Code ownership guidelines

- keep validation rules in `internal/validate`
- keep capture behavior in `internal/capture`
- keep telemetry emission behavior in `internal/generator`
- keep runtime orchestration in `internal/runtime`
- keep reporting renderers pure where possible

## Reproducing CI failures

- download the artifact bundle
- inspect `report.json` and `collector.log`
- rerun locally with `--keep-containers`

## Release docs publishing

- PRs build the docs site and fail on drift
- `main` publishes the `dev` docs version
- `v*` tags publish `<major>.<minor>` plus `latest`
