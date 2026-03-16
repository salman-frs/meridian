# CI Maintainer Workflow

## Summary

This workflow is for the engineer who owns the pull request automation rather than the Collector config itself.

## Workflow

1. Add the Meridian action to the workflow.
2. select the config path, runtime engine, and optional assertions file
3. ensure the job has `pull-requests: write` when PR comments are desired
4. verify artifacts, step summary, and PR comment behavior in a test pull request
5. teach reviewers where Meridian evidence lives

## What matters operationally

CI maintainers should care most about:

- deterministic action behavior
- machine-readable `ci` output
- artifact retention
- permissions for PR comment updates
- runner compatibility with the selected engine

## Common mistakes

- using `containerd` on a runner that does not have `nerdctl`
- treating the PR comment as the only evidence surface
- ignoring uploaded artifacts when the summary is insufficient

## Read next

- [GitHub Action](../features/github-action.md)
- [GitHub Action Reference](../reference/action.md)
