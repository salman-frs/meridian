# GitHub Action

## Summary

The composite GitHub Action makes Meridian practical in pull requests by standardizing how `meridian ci` is executed and how its evidence is surfaced.

## Why it exists

Teams need more than a job that returns exit code `0` or `1`. They need CI feedback that is:

- reviewable
- durable
- consistent across pull requests

The action provides that wrapper without hiding the underlying CLI behavior.

## How it works

The action:

1. builds the Meridian binary from the checked-out repository
2. runs `meridian ci`
3. uploads artifacts
4. writes the GitHub step summary
5. updates a single PR comment when comment mode is enabled

It accepts the same runtime selector as the CLI: `auto`, `docker`, or `containerd`.

## What it proves

The action proves whatever `meridian ci` proves for the selected inputs. It does not add extra validation semantics beyond packaging those results for GitHub review.

## Limits

- `containerd` is intended for Linux runners with `nerdctl`
- the current implementation depends on `actions/upload-artifact@v4`
- GHES compatibility is not part of the current contract

## Related pages

- [GitHub Action Reference](../reference/action.md)
- [CI Maintainer Workflow](../workflows/ci-maintainer.md)
