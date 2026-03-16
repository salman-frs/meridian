# K3s Fixture

## Summary

The repository-owned fixture under `examples/k3s-e2e/` is Meridian's real-cluster regression target for Meridian itself.

## Why it exists

Meridian needs one authoritative cluster-level validation environment that the project owns end to end. Using a repo-owned fixture keeps the acceptance surface small, editable, and understandable.

## What it includes

The fixture uses four application roles:

- `storefront`
- `checkout`
- `inventory`
- `trafficgen`

These services are intentionally limited so regressions can be diagnosed from artifacts instead of from a large demo stack.

## Scenario model

Scenarios include:

- `happy`
- `drop-traces`
- `misroute-logs`
- `auth-fail`
- `backend-unreachable`

Negative scenarios are expected to pass when the expected failure mode is detected correctly.

## Acceptance signals

Primary signals on the project VM are:

- pod logs
- Prometheus evidence
- gateway counters

Tempo and Loki remain supporting evidence, but are not the sole blocking signal in every scenario because they are not always fully run-scoped.

## Related pages

- [Meridian Maintainer Regression Workflow](../workflows/maintainer-regression.md)
- [Artifact Contract](../reference/artifact-contract.md)
