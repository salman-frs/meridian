# Reporting and Artifacts

## Summary

Meridian's reporting subsystem turns runtime state into durable evidence. This matters because the human product surface is often the artifact bundle, not the original terminal session.

## How bundle writing works

`report.WriteBundle` is responsible for:

- ensuring the run directory exists
- writing semantic findings and component inventory when available
- writing `config.final.yaml` when effective config exists
- writing `report.json`
- writing `summary.md`
- writing `diff.md` when diff data exists
- writing `contracts.json` and `contracts.md`
- updating the `runs/latest` symlink

## Summary rendering

`RenderSummaryMarkdown` builds the reviewer-facing markdown summary. It focuses on:

- overall run state
- semantic stage status
- risk highlights from diff analysis
- contract and assertion results
- top failure context
- artifact names

## CI-facing reporting

The same report subsystem also provides:

- PR comment markdown
- terminal summaries
- GitHub workflow command annotations for failures

## Why this matters

When the docs describe “what a run produced,” they should align with the actual bundle writer and renderer behavior rather than an idealized artifact list.
