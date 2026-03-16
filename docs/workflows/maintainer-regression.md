# Meridian Maintainer Regression Workflow

## Summary

This workflow is for maintainers validating Meridian itself, especially changes to runtime orchestration, patching, reporting, and fixture evaluation.

## Workflow

1. run `go test ./...`
2. exercise the normal local CLI flows where relevant
3. run the repo-owned k3s fixture on the project VM
4. inspect `summary.md`, `summary.json`, and supporting scenario artifacts
5. confirm that negative scenarios pass for the right reasons

## Why this workflow exists

The k3s fixture is a regression target for Meridian, not a generic user tutorial. It exists to provide one controlled cluster-level acceptance surface the project owns.

## What to verify

Maintainers should verify:

- scenario exit semantics
- artifact completeness
- fidelity of summary reporting
- run-scoped evidence behavior for gateway counters, Prometheus, and logs

## Read next

- [K3s Fixture](../features/k3s-fixture.md)
- [CI and Docs Publishing](../architecture/ci-and-docs-publishing.md)
