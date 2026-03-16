# System Architecture

## Summary

Meridian is organized as a Go CLI with explicit subsystems for config loading, semantic validation, graphing, diffing, patching, runtime execution, assertion evaluation, and reporting.

## End-to-end view

```mermaid
flowchart TD
    A["Config Inputs (--config / --config-dir / URIs)"] --> B["configio: expand, materialize, interpolate"]
    B --> C["validate: repo-side findings"]
    B --> D["collector: semantic validation and print-config"]
    C --> E["graph / diff preparation"]
    D --> F["runtime config selection"]
    F --> G["patch: inject receiver/exporter and build test plan"]
    G --> H["runtime: start collector, inject telemetry, wait for capture"]
    H --> I["assert: default assertions and contracts"]
    I --> J["report: report.json, summary.md, diff.md, contracts.md"]
    J --> K["debug commands / CI summary / PR comment / uploaded artifacts"]
```

## Major subsystems

- `internal/app`: command definitions, option parsing, and orchestration entry points
- `internal/configio`: source expansion, materialization, YAML loading, env interpolation
- `internal/collector`: Collector-native semantic validation and effective-config resolution
- `internal/graph`: pipeline graph model generation
- `internal/diffing`: structural diff classification
- `internal/patch`: runtime config patching and test-plan generation
- `internal/runtime`: container execution, injection, capture wait, runtime adapters
- `internal/assert`: assertions, contract loading, and evaluation
- `internal/report`: markdown, terminal, diff, contract, and bundle rendering
- `internal/model`: shared types and artifact manifest definitions

## Architectural intent

Meridian is intentionally concrete. The implementation favors explicit stages over framework-heavy abstractions so that a failing run can usually be traced to one subsystem and one stage.
