# Install

## Summary

Meridian is a single Go binary. Most installations are either:

- a local source build
- a release binary download

The runtime and graph features have additional dependencies depending on what commands you plan to use.

## Build from source

```bash
go build -o ./bin/meridian ./cmd/meridian
```

Requirements:

- Go 1.25 or newer
- a supported container engine for runtime commands

## Install a release binary

```bash
curl -fsSL -o meridian.tar.gz https://github.com/salman-frs/meridian/releases/latest/download/meridian-linux-amd64.tar.gz
tar -xzf meridian.tar.gz
./meridian version
```

## Runtime prerequisites

| Capability | Requirement |
| --- | --- |
| Repo-side validation only | local files and the Meridian binary |
| Semantic validation via image | Docker or containerd |
| Runtime commands | Docker, or `nerdctl`, or `lima nerdctl` on macOS |
| SVG graph rendering | Graphviz `dot` |

## Engine-specific notes

- `auto` prefers Docker when both Docker and containerd are available
- Linux containerd expects `nerdctl`
- macOS containerd uses `lima nerdctl`

## Docs development prerequisites

If you are contributing to the docs site:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r docs/requirements.txt
go run ./cmd/meridian-docs
mkdocs serve
```

## Read next

- [Quickstart](quickstart.md)
- [Runtime Engine Model](../core-concepts/runtime-engine-model.md)
