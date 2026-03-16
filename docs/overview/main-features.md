# Main Features

## Summary

Meridian's command surface is compact, but each command sits in a broader review model. The right way to understand the product is by capability, not by memorizing verbs.

## Static validation

`validate` loads the config, performs repo-side validation where possible, and runs Collector-native semantic validation against a real target. It is the first fast-confidence step.

See [Validate](../features/validate.md).

## Graph generation

`graph` turns Collector pipeline structure into a graph model and artifacts. It makes topology changes reviewable instead of buried in YAML.

See [Graph](../features/graph.md).

## Diff-aware review

`diff` classifies changes and highlights the parts of a config edit that deserve careful review. It prefers effective config when that evidence exists.

See [Diff](../features/diff.md).

## Runtime proof

`test`, `check`, and `ci` run the deterministic harness. These commands patch the config, boot a Collector, inject synthetic telemetry, wait for capture, evaluate assertions and contracts, and write an artifact bundle.

See [Runtime Commands](../features/runtime-commands.md).

## Assertions and contracts

Assertions provide runtime flow checks. Contracts extend that into fixture-driven behavior validation against normalized telemetry.

See [Assertions and Contracts](../features/assertions-and-contracts.md).

## Artifact inspection

`debug` subcommands let engineers inspect the bundle after the fact, which matters because reviewers often consume output asynchronously.

See [Debug Commands](../features/debug-commands.md).

## CI integration

The composite GitHub Action turns Meridian into a PR review primitive by uploading artifacts, writing a step summary, and optionally updating a persistent PR comment.

See [GitHub Action](../features/github-action.md).
