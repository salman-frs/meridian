# Execution Pipeline

## Summary

The runtime path is centered on `RunService` in `internal/app/run.go`. This is the best place to understand the real operational sequence for `test`, `check`, and `ci`.

## Runtime sequence

### 1. Load inputs

Meridian validates global and runtime options, expands config sources, loads env values, and attempts repo-side config loading.

### 2. Prepare artifacts and engine

It allocates a run directory, reserves runtime ports, and resolves the selected runtime engine.

### 3. Run semantic validation

It validates the config against the selected Collector target and records stage results and findings.

### 4. Select the runtime config

It chooses the runtime config source:

- repo-side merged config when available
- Collector-rendered config when required and available

### 5. Write static artifacts

It runs repo-side validation and graph generation on the runtime config model and stores those results in the pending run result.

### 6. Patch the runtime config

It injects the Meridian receiver/exporter pair, adjusts selected pipelines, and builds a `TestPlan`.

### 7. Attach diff analysis

When diff inputs are present or the command includes diff-aware behavior, it attaches diff results to the run.

### 8. Execute the runtime harness

The runtime runner starts the Collector, injects synthetic telemetry, waits for capture, persists normalized capture output, evaluates assertions and contracts, and collects logs.

### 9. Write the bundle

Reporting writes `report.json`, `summary.md`, diff and contract artifacts, semantic artifacts, and the `latest` symlink.

## Why this matters

This explicit pipeline is the backbone of both the docs and the codebase. When describing a feature, the docs should make it clear where that feature sits in the sequence.
