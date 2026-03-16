# Config Inputs

Meridian supports multiple ways to provide Collector config.

## Repeatable `--config`

`--config` is repeatable and accepts:

- local file paths
- Collector-native config URIs

Example:

```bash
./bin/meridian validate \
  -c collector.yaml \
  -c yaml:exporters::otlp::endpoint=localhost:4317
```

## `--config-dir`

Use `--config-dir` when your Collector config is already rendered into a directory layout.

```bash
./bin/meridian validate --config-dir ./rendered-config
```

## Env interpolation

Meridian resolves env values from:

1. `--env-file`
2. repeatable `--env KEY=value`
3. exported shell env vars

Repo-side parsing tracks referenced env names and reports missing values clearly.

## Collector-native validation target

For semantic validation:

- use `--collector-binary` to point at a specific local Collector build
- otherwise Meridian can use `--collector-image` through the selected engine

## URI-only runtime constraints

Runtime commands can work with URI-only inputs only when Meridian can materialize a local runtime config:

- directly from locally materializable sources
- or from Collector `print-config`

If neither path exists, runtime preparation fails before execution.
