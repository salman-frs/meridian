# Goals and Non-Goals

## Summary

Meridian is opinionated about what it is trying to make safer. The product is centered on pre-merge confidence for OpenTelemetry Collector configuration changes.

## Goals

Meridian is designed to:

- catch risky Collector changes before they merge
- make Collector review easier with readable graphs, diffs, summaries, and artifacts
- validate runtime flow without depending on live vendor backends by default
- keep local and CI behavior close enough that results are portable
- remain maintainable as a Go CLI with explicit orchestration stages

## Why these goals matter

Collector review usually fails in one of two ways:

- teams only do static syntax checks, which are too shallow
- teams rely on production-like environments too early, which are too noisy and expensive

Meridian tries to give teams a middle path: stronger than linting, cheaper and more deterministic than full environment testing.

## Non-goals

Meridian is not intended to:

- replace production monitoring or canary analysis
- fully validate vendor-specific auth and backend ingestion in default `safe` mode
- become a general Kubernetes test framework
- hide Collector behavior behind a proprietary control plane
- replace expert judgment in review with a single score

## Practical implication

When Meridian says a run passed, the correct interpretation is:

the selected config, Collector target, and runtime harness produced the expected evidence under the chosen mode.

That is intentionally narrower than “production is safe,” and the docs should preserve that distinction everywhere.
