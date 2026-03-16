# First CI Setup

## Summary

The goal of a first CI setup is parity with the local `ci` workflow, not a custom wrapper around Meridian.

## Minimal workflow

```yaml
name: meridian

on:
  pull_request:
    paths:
      - "otel/**"
      - "**/collector*.yaml"
      - "**/otelcol*.yaml"

jobs:
  meridian:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
    steps:
      - uses: actions/checkout@v4
      - uses: salman-frs/meridian/action@v1
        with:
          config: examples/basic/collector.yaml
          engine: auto
```

## What this produces

The composite action:

- builds the Meridian binary
- runs `meridian ci`
- uploads artifacts
- writes the GitHub step summary
- optionally updates a single PR comment marked with `<!-- meridian-comment -->`

## Recommended review behavior

Teach reviewers to inspect:

- the step summary for a quick read
- the PR comment for persistent status
- the artifact bundle for full evidence

## Read next

- [GitHub Action](../features/github-action.md)
- [Reviewer Pull Request Workflow](../workflows/reviewer-pull-request.md)
