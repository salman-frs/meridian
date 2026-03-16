# Runtime Engines

Meridian supports `auto`, `docker`, and `containerd`.

## Auto resolution

`auto` prefers Docker when both engines are available. This keeps behavior stable for common local and CI environments.

## Docker

Docker is the default expectation for:

- local `test` and `check`
- GitHub Actions examples
- semantic validation via image when no local Collector binary is provided

For Docker, Meridian routes capture traffic through a host alias.

## Linux containerd

On Linux, `--engine containerd` expects `nerdctl` and uses host networking for the capture path.

## macOS containerd

On macOS, `--engine containerd` uses `lima nerdctl`, published injection ports, and `host.lima.internal` for capture sink routing.

Direct `nerdctl` reuse of an OrbStack Docker context is not part of Meridian's supported contract.
