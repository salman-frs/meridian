# Diff-Aware Review

Meridian can compare two Collector configs and highlight risky changes for reviewers.

## Inputs

Diff-aware review supports:

- explicit file paths via `--old` and `--new`
- git refs via `--base` and `--head`
- severity filtering with `--severity-threshold`

Runtime workflows can also enable `--changed-only` to restrict review hints to explicit diff inputs.

## Classification focus

Meridian classifies changes in:

- pipeline wiring
- processors
- connectors
- extensions
- auth and TLS-related config
- `service.telemetry`

## Effective config preference

When Collector `print-config` is available, Meridian prefers effective-config diffs over raw source diffs. This improves review quality for configs that rely on includes, URI sources, or Collector rendering behavior.

## Outputs

Diff data appears in:

- `report.json`
- `summary.md`
- `diff.md` when changes exist

The goal is reviewer guidance, not a generic YAML diff replacement.
