# Config Source Model

## Summary

Meridian accepts more than one kind of Collector input, but not all inputs are equally inspectable. The config source model explains when Meridian can reason about config locally and when it must rely on Collector-native behavior.

## Supported input shapes

Meridian accepts:

- repeatable `--config` values
- `--config-dir`
- local file paths
- supported Collector-native config URIs

Examples include local YAML files and URI-style inputs such as `yaml:` and `file:`.

## How local loading works

The `configio` package expands the source list, materializes any locally materializable sources, interpolates env values, merges YAML documents, and produces a normalized config model.

Repo-side parsing depends on at least one materializable YAML source. If all inputs are URI-only and cannot be materialized locally, Meridian reports that repo-side parsing was skipped.

## Why this matters at runtime

Runtime patching needs a concrete config model. Meridian can get that in two ways:

- from source-merged config when all inputs are locally materializable
- from Collector `print-config` when URI sources require Collector-native rendering

If neither path is available, runtime preparation fails before the Collector starts.

## Environment interpolation

Meridian resolves env values from:

1. `--env-file`
2. repeatable `--env KEY=value`
3. exported shell environment variables

Missing env values are tracked as part of the config model and should be treated as review-relevant evidence, not just a parsing nuisance.

## Read next

- [Config Inputs reference](../reference/config-inputs.md)
- [Semantic Validation](semantic-validation.md)
