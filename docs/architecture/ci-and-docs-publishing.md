# CI and Docs Publishing

## Summary

Meridian has two distinct but related automation surfaces:

- the product-facing GitHub Action for validating Collector configs
- the repository's own docs publishing workflow

## Product CI path

The composite action:

- builds the Meridian binary
- runs `meridian ci`
- uploads artifacts
- writes the step summary
- optionally updates a PR comment

This is the workflow external repositories consume.

## Docs publishing path

The docs workflow:

- regenerates CLI reference pages
- fails on generated-doc drift
- builds the MkDocs site in strict mode
- deploys `dev` from `main`
- deploys release docs on `v*` tags using `mike`

## Why it is separate

Docs publishing and product CI solve different problems:

- product CI validates Collector configs
- docs CI validates and publishes Meridian documentation

Keeping them separate prevents documentation tooling from leaking into the normal product validation surface.
