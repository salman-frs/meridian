# Install

## Build from source

Meridian is a single Go binary.

```bash
go build -o ./bin/meridian ./cmd/meridian
```

Requirements:

- Go 1.25+ to build from source
- Docker for the default runtime path
- `nerdctl` for Linux `--engine containerd`
- Lima for macOS `--engine containerd` via `lima nerdctl`
- Graphviz `dot` only when rendering SVG graphs

## Install a release binary

```bash
curl -fsSL -o meridian.tar.gz https://github.com/salman-frs/meridian/releases/latest/download/meridian-linux-amd64.tar.gz
tar -xzf meridian.tar.gz
./meridian version
```

Release artifacts are published by the repository release workflow and include checksums.

## Install docs tooling

For local docs work:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r docs/requirements.txt
go run ./cmd/meridian-docs
mkdocs serve
```

## Install matrix

| Use case | Minimum requirement |
| --- | --- |
| `validate` with local YAML only | Go build plus local files |
| `validate` with semantic image checks | Docker or containerd plus Collector image pull access |
| `graph --render=svg` | Graphviz `dot` |
| `test`, `check`, `ci` | Supported container engine |
| `--engine containerd` on macOS | Lima with `lima nerdctl` |

## Output layout

Runtime and debug-oriented commands use `./meridian-artifacts` by default and accept `--output` to choose a different artifact root.
