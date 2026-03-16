# Validate

## Summary

`meridian validate` is the fast, static-confidence entry point. It combines repo-side validation with Collector-native semantic validation when a binary or image target is available.

## Why it exists

Collector review needs a fast first pass that is stronger than YAML parsing but cheaper than runtime execution. `validate` is that layer.

## How it works

`validate`:

1. expands config sources
2. attempts repo-side loading through `configio`
3. reports when local loading is skipped for URI-only inputs
4. loads environment values
5. runs Collector-native semantic validation through `collector.Analyze`
6. prints a human summary or JSON payload

## What it proves

It proves that Meridian can understand the config locally when materialization is possible and that the selected Collector target can validate the config at the semantic layer.

It does not prove runtime flow.

## Inputs and outputs

Primary inputs:

- `--config`
- `--config-dir`
- `--env-file`
- `--env`
- `--collector-binary`
- `--collector-image`
- `--engine`

Primary outputs:

- terminal or JSON stage summary
- findings grouped by severity

## Failure modes

Common failure cases:

- no config input was provided
- env interpolation failed
- local loading failed on malformed YAML
- semantic target could not be resolved
- Collector-native validation failed

## Related pages

- [Semantic Validation](../core-concepts/semantic-validation.md)
- [Config Inputs](../reference/config-inputs.md)
