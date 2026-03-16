# Package Map

## Summary

Meridian's package boundaries reflect operational responsibilities rather than theoretical layers.

## Package responsibilities

### `internal/app`

Builds the Cobra command tree, validates options, and orchestrates high-level command behavior.

### `internal/configio`

Expands config inputs, materializes local sources, interpolates environment values, and produces normalized config models.

### `internal/collector`

Executes Collector-native semantic validation and effective-config retrieval using either a local binary or an image through the selected engine.

### `internal/graph`

Builds the graph model that powers Mermaid, DOT, SVG, and table output.

### `internal/diffing`

Compares old and new configurations and classifies review-relevant changes.

### `internal/patch`

Patches the config for runtime execution and emits the `TestPlan`.

### `internal/runtime`

Owns engine resolution, container startup, capture sink wiring, telemetry injection, capture waiting, and container cleanup.

### `internal/assert`

Loads assertion and contract suites and evaluates them against captured or normalized telemetry.

### `internal/report`

Renders human-facing summaries, diff markdown, contract markdown, terminal output, PR comments, and persists bundle artifacts.

### `internal/model`

Defines shared types such as config models, run results, semantic reports, artifact manifests, and contract structures.

## Architectural implication

The docs should mirror these boundaries. If a documentation page spans too many of these responsibilities without naming them, it will usually feel vague.
