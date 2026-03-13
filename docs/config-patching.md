# Config Patching

Meridian preserves the original config and writes a patched config for runtime execution.

## Safe mode

- injects an OTLP receiver named `otlp/meridian_in`
- injects an OTLP exporter named `otlp/meridian_capture`
- replaces pipeline receivers with the injected receiver
- replaces pipeline exporters with the capture exporter, while preserving connector exporters
- resolves the capture exporter endpoint from the selected runtime engine

## Tee mode

- keeps original exporters
- appends the Meridian capture exporter

## Live mode

- keeps real destination exporters
- appends the Meridian capture exporter so Meridian can still evaluate runtime assertions
- still injects the Meridian receiver for deterministic input
- may ship synthetic telemetry to real destinations, so use deliberately

Always inspect `config.patched.yaml` in the run bundle when debugging unexpected runtime behavior.
