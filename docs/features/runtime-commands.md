# Runtime Commands: `test`, `check`, and `ci`

## Summary

The runtime commands are Meridian's core confidence path. They run a deterministic harness against a patched Collector config and write a bundle of evidence.

## Why they exist

Static validation can tell you whether a config is understandable. It cannot prove that telemetry still traverses the intended pipelines. The runtime commands exist to answer that question in a controlled environment.

## Command roles

### `test`

Runs the runtime harness directly. Use it when you want focused local execution without the opinionated review framing of `check`.

### `check`

Runs Meridian's opinionated end-to-end confidence workflow. This is the recommended local command for most engineers.

### `ci`

Wraps the same workflow in a CI-oriented interface, with machine-readable output, step summary writing, PR comment rendering, and annotation behavior.

## How they work

At a high level, Meridian:

1. loads and validates inputs
2. runs semantic validation
3. selects the runtime config source
4. writes static artifacts such as graph output
5. patches the config
6. optionally attaches diff analysis
7. starts the Collector through the selected engine
8. injects synthetic telemetry
9. waits for capture
10. evaluates assertions and contracts
11. writes the bundle and summary

## What they prove and do not prove

In `safe` mode, these commands prove deterministic pipeline flow through the patched harness. They do not prove real backend behavior.

`tee` and `live` can provide more realistic destination behavior, but they change the risk profile and should not be treated as equally deterministic.

## Inputs and outputs

Important flags include:

- `--mode`
- `--engine`
- `--collector-image`
- `--assertions`
- `--pipelines`
- `--render-graph`
- `--output`
- `--keep-containers`

`ci` adds:

- `--summary-file`
- `--json-file`
- `--pr-comment-file`
- `--pr`

## Failure modes

Common failure sources:

- runtime engine preflight failure
- Collector startup failure
- telemetry injection failure
- no capture for the expected signal
- failing assertions or contracts

## Related pages

- [Runtime Modes and Patching](../core-concepts/runtime-modes-and-patching.md)
- [Execution Pipeline](../architecture/execution-pipeline.md)
