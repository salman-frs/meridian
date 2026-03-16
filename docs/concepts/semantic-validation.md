# Semantic Validation

Semantic validation is Meridian's Collector-native check layer. It validates the config against an actual Collector distribution before runtime execution.

## Target resolution

Meridian resolves the semantic target in this order:

1. `--collector-binary`
2. `--collector-image` using the selected engine
3. skip semantic validation when the command allows it and no target is available

## Stages

Meridian surfaces explicit semantic stages:

- `components`
- `validate`
- `print-config`

`print-config` is best effort and uses the `otelcol.printInitialConfig` feature gate. Meridian reports when that stage is skipped, unsupported, or fails.

## Why this matters

Repo-side parsing alone cannot prove that the selected Collector build:

- supports the referenced components
- accepts the actual config schema
- emits usable effective config for diffing and runtime provenance

## Evidence artifacts

When available, semantic validation writes:

- `collector-components.json`
- `semantic-findings.json`
- `config.final.yaml`

Those files complement `report.json` and `summary.md`.

## URI-only configs

If every config source is a Collector-native URI such as `yaml:` or `file:`, repo-side parsing may be skipped. In that case, semantic validation becomes the primary static evidence path.
