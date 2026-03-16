# Semantic Validation

## Summary

Semantic validation is Meridian's bridge between static YAML analysis and actual Collector behavior. It validates the selected config against a real Collector target before runtime execution.

## Why it exists

Repo-side parsing can tell you whether the config is structurally understandable. It cannot tell you whether the Collector distribution you intend to run:

- supports the referenced components
- accepts the actual config schema
- can render a usable effective config

Semantic validation closes that gap.

## How it works

Meridian resolves the semantic target in this order:

1. `--collector-binary`
2. `--collector-image` through the selected engine
3. skip semantic validation where the command allows it

It then executes these stages:

- `components`
- `validate`
- `print-config`

`print-config` is best effort and requires the `otelcol.printInitialConfig` feature gate.

## What it proves

Semantic validation proves that the selected Collector target can execute the relevant validation stages for the provided config sources.

It does not prove runtime flow by itself. That remains the job of the runtime harness.

## Inputs and outputs

Key inputs:

- config sources
- env values
- Collector binary or image
- engine selection

Key outputs:

- stage results in `summary.md` and `report.json`
- `collector-components.json` when inventory is available
- `semantic-findings.json` when findings exist
- `config.final.yaml` when `print-config` succeeds

## Failure modes and limits

Expect degraded behavior when:

- no semantic target can be resolved
- the selected Collector does not implement `components`
- `print-config` is unsupported or unavailable
- the target Collector rejects the config

Meridian reports those cases explicitly so effective-config evidence is never implied when it does not exist.

## Read next

- [Validate](../features/validate.md)
- [Artifact Model](artifact-model.md)
