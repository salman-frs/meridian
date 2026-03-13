# Install

## Local build

```bash
go build -o ./bin/meridian ./cmd/meridian
```

## Release binary

```bash
curl -fsSL -o meridian.tar.gz https://github.com/salman-frs/meridian/releases/latest/download/meridian-linux-amd64.tar.gz
tar -xzf meridian.tar.gz
./meridian version
```

## Binary layout

The CLI is a single binary. On each run it writes a deterministic artifact bundle containing:

- `report.json`
- `summary.md`
- `graph.mmd`
- `collector.log`
- `config.patched.yaml`
- `captures/*.json`
- optional `graph.svg`
- optional `diff.md`

## Requirements

- Go 1.24+ to build from source
- Docker for `check` and `test`
- Graphviz optional for `meridian graph --render=svg`
