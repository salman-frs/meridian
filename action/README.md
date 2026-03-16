# Meridian Action

This composite action builds the Meridian CLI from the checked-out repository, runs `meridian ci`, uploads the generated artifacts, and updates a single PR comment marked with `<!-- meridian-comment -->` when comment mode is enabled.

The action accepts the same runtime selector as the CLI through the `engine` input: `auto`, `docker`, or `containerd`. For GitHub Actions, `containerd` is intended for Linux runners with `nerdctl` installed.

Use the `assertions` input for either a legacy `v1` assertions file or a `v2` contracts/fixtures file.

Use `uses: ./action` for local development in this repository. The published `meridian/action@v1` tag is intended for external repos once release binaries and checksums are published.

This action currently depends on `actions/upload-artifact@v4`, so it is not GHES-compatible as written.
