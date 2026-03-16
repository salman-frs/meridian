# Overview

Meridian is an engineer-first tool for proving OpenTelemetry Collector changes before merge. It is not a linter with a nicer UI and it is not a hosted validation service. It is a pragmatic CLI and GitHub Action that produces evidence a reviewer can trust.

## What Meridian is

At a high level, Meridian does four things:

- reads Collector configuration from local files, rendered directories, and supported Collector-native config URIs
- validates that configuration at both the repo level and the Collector-distribution level
- runs a deterministic runtime harness against a patched config
- writes durable artifacts for authors, reviewers, and CI maintainers

## Design goals

Meridian is built around a few explicit product priorities:

- local-first usage
- deterministic default behavior
- reviewable outputs over opaque automation
- artifact-first execution
- concrete, Go-first implementation boundaries

These priorities shape both the user experience and the code structure.

## Intended users

Meridian is primarily for:

- platform engineers who own shared Collector configurations
- observability engineers who review telemetry routing and processing changes
- application teams that want pre-merge confidence for Collector edits
- CI maintainers who need a repeatable PR validation loop

## Adoption path

The recommended sequence is:

1. start with `validate`
2. move to `check` in `safe` mode
3. inspect the generated artifacts
4. add assertions or contracts where your review standard needs them
5. wire `ci` into pull requests with the composite action

Meridian becomes significantly more valuable once teams rely on artifact review rather than terminal output alone.

## Read next

- [Goals and Non-Goals](goals-and-non-goals.md)
- [Users and Use Cases](users-and-use-cases.md)
- [Main Features](main-features.md)
