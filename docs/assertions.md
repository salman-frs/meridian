# Assertions And Contracts

Meridian always applies default flow assertions for detected signal types:

- at least one item arrives
- telemetry arrives within the configured capture timeout
- the current `meridian.run_id` is present
- the capture sink decodes the payload without errors

## v1 Assertions

```yaml
version: 1
defaults:
  timeout: 10s
  min_count: 1

assertions:
  - id: traces-flow
    severity: fail
    signal: traces
    where:
      attributes:
        meridian.run_id: "{{RUN_ID}}"
    expect:
      min_count: 1
```

Supported `where` filters in v1:

- `attributes`
- `span_name`
- `metric_name`
- `body`

Supported `expect` keys in v1:

- `min_count`
- `exists`
- `attributes_present`
- `attributes_absent`

Meridian replaces `{{RUN_ID}}` placeholders with the current run ID and applies `defaults.min_count` when a custom assertion does not define `exists` or `min_count`.

## v2 Contracts And Fixtures

The same `--assertions` flag now accepts a richer v2 file format for fixture-driven contract testing. Contracts are evaluated against normalized captured telemetry and are meant for reviewer confidence after routing, filtering, redaction, or transform changes.

```yaml
version: 2
defaults:
  min_count: 1

fixtures:
  - redaction

contracts:
  - id: no-authorization-header
    severity: fail
    signal: traces
    fixture: redaction
    expect:
      attributes_absent:
        - http.request.header.authorization
```

Built-in fixtures:

- `pass-through`
- `redaction`
- `filter-drop`
- `routing-copy`
- `routing-move`
- `metric-transform`

Supported contract expectations:

- `min_count`
- `exact_count`
- `max_count`
- `exists`
- `attributes_present`
- `attributes_absent`
- `equals`
- `contains`
- `regex`
- `metric_value.eq|gt|gte|lt|lte`

Supported field paths for `equals`, `contains`, and `regex`:

- `span_name`
- `metric_name`
- `body`
- `fixture`
- `run_id`
- `attributes.<key>`
- `resource.<key>`
- `metric_value`

Runtime bundles now also include:

- `contracts.json`
- `contracts.md`
- `capture.normalized.json`
