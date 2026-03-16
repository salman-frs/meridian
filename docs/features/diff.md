# Diff

## Summary

`meridian diff` compares two Collector configurations and classifies review-relevant changes. It is designed to highlight risk, not to replace a generic text diff.

## Why it exists

Not every config edit deserves the same scrutiny. Reviewers care most about changes that affect routing, processors, connectors, extensions, auth, TLS, and telemetry service behavior.

## How it works

Meridian can diff:

- explicit old and new file paths
- git base and head refs

When effective config from Collector `print-config` is available, Meridian prefers that evidence. Otherwise it falls back to source-level comparison.

Runtime commands can also incorporate diff-aware review hints and optionally restrict them with `--changed-only`.

## What it proves

Diff output proves how Meridian classified the structural change between two configurations. It does not prove that the changed config is safe by itself; use it together with validation and runtime evidence.

## Inputs and outputs

Important flags:

- `--old`
- `--new`
- `--base`
- `--head`
- `--severity-threshold`
- `--changed-only` in runtime flows

Artifacts:

- diff section in `report.json`
- highlights in `summary.md`
- `diff.md` when diff data exists

## Related pages

- [Graph](graph.md)
- [Reviewer Pull Request Workflow](../workflows/reviewer-pull-request.md)
