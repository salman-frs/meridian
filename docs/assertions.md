# Assertions

Meridian always applies default flow assertions for detected signal types:

- at least one item arrives
- telemetry arrives within the configured capture timeout
- the current `meridian.run_id` is present
- the capture sink decodes the payload without errors

## Custom assertions

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
