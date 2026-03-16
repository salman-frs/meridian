# CLI Overview

## Summary

Meridian's CLI is small on purpose. The commands map to a few distinct user intents rather than a large set of modes.

## Command groups by intent

### Validate and understand a config

- `validate`
- `graph`
- `diff`

### Prove runtime flow

- `test`
- `check`
- `ci`

### Inspect previous runs

- `debug logs`
- `debug capture`
- `debug summary`
- `debug bundle`

### Shell and metadata helpers

- `version`
- `completion`

## Recommended entry points

- first local run: `validate`, then `check`
- PR review workflow: `diff`, then `ci`
- post-failure inspection: `debug summary`, `debug bundle`, `debug logs`

## Generated reference

The generated pages under [Generated CLI Reference](cli/index.md) are the source of truth for exact command help, inherited flags, and usage strings.
