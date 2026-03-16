# Debug Commands

## Summary

The `debug` command group is Meridian's post-run inspection surface. It exists because engineers and reviewers often consume output after the original command already finished.

## Why it exists

Meridian is artifact-first. The debug commands make those artifacts easier to consume without asking a user to remember bundle paths manually.

## Commands

- `debug summary`: print the stored markdown summary
- `debug bundle`: print the bundle manifest in human or JSON form
- `debug logs`: print stored Collector logs
- `debug capture`: print persisted capture samples

## How it works

When `--run` is omitted, debug commands resolve `runs/latest` relative to the selected output root. This keeps the common local inspection path simple.

## What it proves

Debug commands do not create new evidence. They expose previously generated evidence in a useful form.

## Failure modes

Failures usually mean:

- the expected run directory does not exist
- `latest` points to a missing path
- the artifact was never generated because the run failed earlier

## Related pages

- [Artifact Model](../core-concepts/artifact-model.md)
- [Artifact Contract](../reference/artifact-contract.md)
