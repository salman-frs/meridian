# Config Inputs

## Repeatable `--config`

`--config` accepts repeatable config sources, including local files and supported Collector-native URIs.

Example:

```bash
./bin/meridian validate \
  -c collector.yaml \
  -c yaml:exporters::otlp::endpoint=localhost:4317
```

## `--config-dir`

Use `--config-dir` when the configuration already exists as a rendered directory of YAML files.

```bash
./bin/meridian validate --config-dir ./rendered-config
```

## Materializable vs non-materializable sources

Meridian can reason repo-side about:

- local YAML files
- `yaml:` sources

Sources that cannot be materialized locally may still be valid for Collector-native semantic validation, but they reduce what Meridian can inspect without `print-config`.

## Env resolution

Env values are resolved from:

1. `--env-file`
2. repeatable `--env KEY=value`
3. exported shell environment variables

## Collector target selection

For semantic validation, Meridian prefers:

1. `--collector-binary`
2. otherwise `--collector-image` through the selected engine
