# Reviewer Pull Request Workflow

## Summary

This workflow is for engineers reviewing a Collector change in a pull request, whether or not they ran Meridian locally themselves.

## Workflow

1. Read the PR diff and identify the targeted config.
2. Read the Meridian step summary.
3. Read the PR comment if comment mode is enabled.
4. Inspect the uploaded artifact bundle when the summary alone is not enough.
5. Focus on diff highlights, semantic stage status, runtime signal status, and contract failures.

## What to prioritize

Reviewers should pay particular attention to:

- risk highlights in the diff section
- whether effective-config evidence was available
- runtime failures by signal
- contract failures that indicate changed output shape
- `collector.log` when startup or patching looks suspicious

## Failure interpretation

A failed run is not always a product regression. It may represent:

- a genuine config problem
- a missing environment dependency
- an unsupported Collector behavior
- a changed contract that needs explicit review

The artifact bundle should let you distinguish those cases.

## Read next

- [Diff](../features/diff.md)
- [Debug Commands](../features/debug-commands.md)
