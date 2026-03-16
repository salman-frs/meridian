# Release Policy

Meridian publishes two documentation tracks:

- `dev`: built from the current `main` branch
- release docs: published from `v*` tags as `<major>.<minor>` with a `latest` alias

The docs site uses `mike` for versioned publishing.

## Supported docs versions

The target policy is:

- keep `dev`
- keep `latest`
- keep the current release plus the previous four release doc versions

If a documentation change only affects unreleased behavior, it belongs in `dev` first.

## Release binary policy

The binary release workflow currently publishes:

- `darwin` and `linux`
- `amd64` and `arm64`
- tarballs plus `checksums.txt`

## Backward-compatibility expectations

Docs should preserve stable top-level CLI verbs and describe behavior that matches the checked-in code for that version. When behavior differs by version, release docs should document the released contract, not `main`.
