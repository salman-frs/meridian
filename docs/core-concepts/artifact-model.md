# Artifact Model

## Summary

Artifacts are a first-class output of Meridian. They are not secondary debug files; they are the product surface reviewers use to understand what happened.

## Layout

Runtime-oriented runs use:

```text
meridian-artifacts/
  runs/
    latest -> <run_id>
    <run_id>/
```

The `latest` symlink is resolved relative to the selected `--output` root and powers the default behavior of the debug commands.

## Core artifacts

The baseline runtime bundle includes:

- `report.json`
- `summary.md`
- `config.patched.yaml`
- `graph.mmd`
- `collector.log`
- `captures/`

Additional artifacts appear when that evidence exists:

- `config.final.yaml`
- `collector-components.json`
- `semantic-findings.json`
- `graph.svg`
- `diff.md`
- `contracts.json`
- `contracts.md`
- `capture.normalized.json`

## Provenance and interpretation

The bundle is more useful when engineers distinguish:

- original config sources
- runtime config source
- patched execution config

That distinction matters when effective config is derived through Collector `print-config` rather than repo-local YAML materialization.

## Read next

- [Debug Commands](../features/debug-commands.md)
- [Artifact Contract](../reference/artifact-contract.md)
