# Assertions and Contracts

## Summary

Meridian supports two layers of output validation:

- `v1` assertions for runtime flow checks
- `v2` contracts for normalized, reviewer-facing behavior checks

## Why they exist

A passing runtime run only tells you that telemetry moved. It does not necessarily tell you that the output shape is still correct after filtering, routing, or mutation. Assertions and contracts raise the review bar.

## How assertions work

Meridian always applies default assertions for each active signal:

- at least one item was received
- telemetry arrived within the configured capture timeout
- the run ID is present
- the capture sink decoded the payload without errors

Custom `v1` assertions extend that with filters and expectations.

## How contracts work

Contracts operate on normalized telemetry and are intended for reviewer confidence after transformations. They support fixture-aware evaluation and richer expectations such as exact counts, string matching, regex, and numeric metric checks.

## What they prove

Assertions prove runtime-flow conditions over captured telemetry.

Contracts prove that the normalized output satisfies the declared expectations for the selected fixture and signal.

Neither should be relaxed casually. They are part of the review contract.

## Artifacts

Contract-related artifacts include:

- `contracts.json`
- `contracts.md`
- `capture.normalized.json`

## Related pages

- [Runtime Commands](runtime-commands.md)
- [Examples](../reference/examples.md)
