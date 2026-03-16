# Runtime Modes and Patching

## Summary

Runtime mode defines how Meridian rewrites the target config for execution. This is the heart of the product's deterministic review model.

## Why it exists

A normal Collector config may send data to live backends or external services. That is often the wrong first feedback loop for pre-merge review. Meridian instead patches the config so it can inject and capture telemetry in a controlled way.

## How patching works

The `patch` package:

- injects a Meridian-managed OTLP receiver
- injects a Meridian-managed OTLP capture exporter
- selects pipelines by name or signal when requested
- rewrites pipeline receivers and exporters according to the chosen mode
- preserves connector exporters in `safe` mode
- records the resulting plan and writes `config.patched.yaml`

## Modes

### `safe`

`safe` replaces normal exporters with the Meridian capture exporter while preserving connector exporters.

Use this when you want deterministic, low-risk evidence.

### `tee`

`tee` preserves original exporters and appends Meridian capture.

Use it when you want to observe runtime output while keeping the original exporter path active.

### `live`

`live` also preserves real exporters and appends Meridian capture.

Use it only when synthetic telemetry reaching real destinations is acceptable.

## What it proves and does not prove

`safe` proves pipeline flow under the patched harness. It does not prove live vendor connectivity.

`tee` and `live` increase realism, but they also increase operational risk and reduce the purity of the deterministic test harness.

## Read next

- [Runtime Commands](../features/runtime-commands.md)
- [Artifact Model](artifact-model.md)
