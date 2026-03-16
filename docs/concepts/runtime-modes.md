# Runtime Modes

Meridian supports three runtime modes: `safe`, `tee`, and `live`.

## Safe

`safe` is the default and recommended mode.

Behavior:

- injects a Meridian-managed OTLP receiver
- injects a Meridian-managed OTLP capture exporter
- replaces pipeline receivers with the injected receiver
- replaces pipeline exporters with the capture exporter
- preserves connector exporters

What it proves:

- the config wires correctly
- processors still let telemetry pass
- selected signals arrive at the capture sink
- assertions and contracts match normalized captured output

What it does not prove:

- real exporter connectivity
- vendor auth correctness
- production routing to live backends

## Tee

`tee` preserves original exporters and appends the Meridian capture exporter.

Use it when you want more realistic exporter behavior while still keeping Meridian evidence.

## Live

`live` also preserves real exporters and appends Meridian capture.

!!! warning
    `live` may send synthetic telemetry to real destinations. Use it deliberately and only when that is acceptable for the target environment.

## Runtime patching

Runtime patching depends on a locally materialized config:

- source-merged config when every source is locally materializable
- Collector-rendered `print-config` output when the config includes non-materializable URIs

If neither path is available, runtime preparation fails clearly before container execution starts.
