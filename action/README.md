# Meridian Action

This composite action builds the Meridian CLI from the checked-out repository, runs `meridian ci`, uploads the generated artifacts, and updates a single PR comment marked with `<!-- meridian-comment -->` when comment mode is enabled.

Use `uses: ./action` for local development in this repository. The published `meridian/action@v1` tag is intended for external repos once release binaries and checksums are published.
