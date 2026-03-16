# Runtime Engine Model

## Summary

Meridian treats the container engine as a first-class runtime input because it affects both semantic validation and runtime execution.

## Supported engines

- `auto`
- `docker`
- `containerd`

## Resolution rules

`auto` prefers Docker when both Docker and containerd paths are available. This preserves a stable happy path for common local and CI environments.

## Engine behavior by environment

### Docker

Docker is the default expectation for local runs and GitHub Actions. For capture routing, Meridian uses a host alias strategy.

### Linux containerd

Linux containerd expects `nerdctl` and uses host networking for the capture path.

### macOS containerd

macOS containerd uses `lima nerdctl`, published injection ports, and `host.lima.internal` for the capture sink endpoint.

## Why this matters

Engine selection affects:

- how Collector images are run for semantic validation
- how runtime ports are wired
- what repro steps are realistic
- what troubleshooting commands make sense

## Read next

- [Runtime Commands](../features/runtime-commands.md)
- [Recommended Workflows](../workflows/index.md)
