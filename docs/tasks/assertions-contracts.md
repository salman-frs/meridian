# Write Assertions And Contracts

Meridian supports legacy `v1` assertions and richer `v2` contracts and fixtures through the same `--assertions` flag.

## Use `v1` assertions

`v1` is a lightweight flow-check format.

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

Supported filters:

- `attributes`
- `span_name`
- `metric_name`
- `body`

Supported expectations:

- `min_count`
- `exists`
- `attributes_present`
- `attributes_absent`

## Use `v2` contracts

`v2` is for reviewer-facing contract testing against normalized captured telemetry.

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

Generated contract artifacts:

- `contracts.json`
- `contracts.md`
- `capture.normalized.json`

## Run with an assertions file

```bash
./bin/meridian check -c collector.yaml --assertions assertions.yaml
```
