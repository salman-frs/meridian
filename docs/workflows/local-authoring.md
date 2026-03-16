# Local Authoring Workflow

## Summary

This is the recommended workflow for an engineer changing a Collector config locally before opening a pull request.

## Workflow

1. Edit the config.
2. Run `meridian validate`.
3. Run `meridian check`.
4. Inspect `summary.md`, `report.json`, `config.patched.yaml`, graph artifacts, and any contract output.
5. If behavior changed intentionally, update assertions or contracts with care.
6. Open the pull request with the local evidence already understood.

## Why this workflow is recommended

It front-loads the cheapest and most deterministic checks:

- static structure
- Collector-native semantics
- runtime flow in `safe` mode

That usually catches the most important regressions before CI runs.

## When to deviate

Use `tee` or `live` only when you explicitly need destination behavior beyond the default deterministic path.

## Read next

- [Runtime Commands](../features/runtime-commands.md)
- [Assertions and Contracts](../features/assertions-and-contracts.md)
