# Release Docs Workflow

Documentation publishing is separate from the binary release workflow.

## Branch and tag behavior

- pull requests build the site and fail on docs drift
- pushes to `main` publish `dev`
- `v*` tags publish `<major>.<minor>` and update `latest`

## Maintainer responsibilities

- keep generated CLI reference checked in
- make docs changes in the same PR as behavior changes
- ensure release notes and docs agree on user-facing contract changes

## Why versioned docs

Meridian's CLI and artifact contract are part of its user interface. Versioned docs reduce drift between release binaries and the published reference.
