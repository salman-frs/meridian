# Choose A Runtime Engine

## Use Docker

Use Docker when:

- you want the default, best-supported path
- you are on GitHub-hosted Linux runners
- you do not need explicit containerd coverage

```bash
./bin/meridian check -c collector.yaml --engine docker
```

## Use Linux containerd

Use Linux containerd when you specifically need `nerdctl` parity or a containerd-backed environment.

```bash
./bin/meridian check -c collector.yaml --engine containerd
```

Verify first:

```bash
nerdctl version
```

## Use macOS containerd

On macOS, Meridian expects Lima:

```bash
lima nerdctl version
./bin/meridian check -c collector.yaml --engine containerd
```

If you have both Docker and Lima installed, remember that `--engine auto` still prefers Docker.
