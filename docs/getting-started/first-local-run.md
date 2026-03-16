# First Local Run

## Summary

A useful first local run is not just executing a command. It is understanding what happened and where the evidence lives.

## Recommended sequence

### Build the binary

```bash
go build -o ./bin/meridian ./cmd/meridian
```

### Validate a known-good config

```bash
./bin/meridian validate -c examples/basic/collector.yaml
```

### Run the default confidence path

```bash
./bin/meridian check -c examples/basic/collector.yaml
```

### Inspect the results

```bash
./bin/meridian debug summary
./bin/meridian debug bundle
```

## What to verify

After your first run, confirm that you can identify:

- which config source Meridian used
- which semantic target it selected
- whether runtime patching used repo-local config or Collector-rendered config
- where `summary.md`, `report.json`, `collector.log`, and `config.patched.yaml` live
- how `runs/latest` resolves

## If env values are involved

Use:

```bash
./bin/meridian validate -c collector.yaml --env-file .env --env OTLP_ENDPOINT=localhost:4317
```

## Read next

- [Config Source Model](../core-concepts/config-source-model.md)
- [Artifact Model](../core-concepts/artifact-model.md)
