# Artifact Model

Artifacts are first-class Meridian output. They are the durable evidence for what happened during a run.

## Layout

Runtime runs use:

```text
meridian-artifacts/
  runs/
    latest -> <run_id>
    <run_id>/
```

`latest` resolves relative to the selected `--output` directory and is used by the debug commands when `--run` is omitted.

## Core runtime artifacts

Every runtime-oriented run aims to preserve:

- `report.json`
- `summary.md`
- `config.patched.yaml`
- `graph.mmd`
- `collector.log`
- `captures/`

Additional artifacts appear when the run produces that evidence:

- `config.final.yaml`
- `collector-components.json`
- `semantic-findings.json`
- `graph.svg`
- `diff.md`
- `contracts.json`
- `contracts.md`
- `capture.normalized.json`

## Provenance model

Meridian reports three related config views:

- original config sources
- runtime config provenance, which explains whether runtime patching used source-merged or Collector-rendered config
- final patched config used for execution

That distinction is important when config URIs or `print-config` are involved.
