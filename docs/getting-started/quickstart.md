# Quickstart

## Summary

This quickstart is intentionally narrow. The goal is one successful local validation and one successful runtime-backed check.

## 1. Build Meridian

```bash
go build -o ./bin/meridian ./cmd/meridian
```

## 2. Run static validation

```bash
./bin/meridian validate -c examples/basic/collector.yaml
```

This proves that Meridian can:

- resolve the config input
- load local YAML when applicable
- run repo-side validation
- attempt Collector-native semantic validation

## 3. Run the opinionated confidence workflow

```bash
./bin/meridian check -c examples/basic/collector.yaml
```

This is the shortest path to a complete Meridian run. It will produce:

- `summary.md`
- `report.json`
- `config.patched.yaml`
- `graph.mmd`
- collector logs
- capture artifacts

Additional semantic, diff, and contract artifacts appear when the run produces them.

## 4. Inspect the latest bundle

```bash
./bin/meridian debug summary
./bin/meridian debug bundle --format json
./bin/meridian debug logs
./bin/meridian debug capture
```

## What to pay attention to

Do not treat a passing run as a single green light. Look at:

- semantic stage results
- graph and diff evidence when reviewing change
- runtime status by signal
- any contract failures or warnings

## Read next

- [First Local Run](first-local-run.md)
- [Runtime Commands](../features/runtime-commands.md)
