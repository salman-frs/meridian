# Release Docs Workflow

Documentation publishing is separate from the product-facing composite action.

## Branch and tag behavior

- pull requests build the site and fail on docs drift
- pushes to `main` publish the `dev` docs version
- `v*` tags publish `<major>.<minor>` and update `latest`

## Version policy

The site keeps:

- `dev`
- `latest`
- the current release plus the previous four release-doc versions

## Why versioned docs matter

Meridian's CLI behavior, artifact contract, and docs structure are part of its user interface. Versioned docs reduce drift between published binaries and the reference site.
