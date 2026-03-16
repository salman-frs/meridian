# Meridian

Meridian is a local-first CLI and GitHub Action for reviewing OpenTelemetry Collector configuration changes before merge.

It exists because Collector configuration is operationally important but hard to review with confidence. A config can be syntactically valid and still be wrong in ways that matter: unsupported components, broken routing, silent signal loss, or runtime behavior that reviewers cannot reconstruct after the fact.

Meridian addresses that gap by combining three evidence layers:

1. repo-side structure and configuration analysis
2. Collector-native semantic validation against a real distribution
3. deterministic runtime execution against a patched test harness with durable artifacts

## Why teams use Meridian

Teams use Meridian when they need a review loop that answers practical engineering questions:

- Does the config parse and wire correctly?
- Does the selected Collector distribution actually accept it?
- What changed, and how risky is that change?
- Do traces, metrics, and logs still move through the intended pipelines?
- Can a reviewer inspect evidence without rerunning the command?

## What Meridian proves

Meridian is designed to prove:

- local configuration structure is materially sane when sources can be loaded repo-side
- the selected Collector binary or image can validate the configuration
- important pipeline changes are visible in graph and diff artifacts
- synthetic telemetry can traverse the configured processing path in a deterministic runtime harness
- assertions and contracts can be evaluated against captured output
- every important run produces artifacts that remain useful in CI and local debugging

## What Meridian does not prove

Meridian does not try to replace production validation.

It does not prove:

- real vendor backend authentication in the default `safe` path
- live production ingestion health
- generic Kubernetes compatibility outside the repo-owned fixture
- any hosted service contract, because Meridian is intentionally local-first

## Start here

**New to Meridian**

- Read [What Meridian Is](overview/index.md)
- Follow [Quickstart](getting-started/quickstart.md)
- Complete [First Local Run](getting-started/first-local-run.md)

**Reviewer or CI integrator**

- Read [Evidence Model](core-concepts/evidence-model.md)
- Read [GitHub Action](features/github-action.md)
- Follow [Reviewer Pull Request Workflow](workflows/reviewer-pull-request.md)

**Maintainer or contributor**

- Read [System Architecture](architecture/system-architecture.md)
- Read [Package Map](architecture/package-map.md)
- Read [Contributing](project/contributing.md)

## Recommended learning path

1. Understand the product intent in [Overview](overview/index.md).
2. Get a successful run in [Getting Started](getting-started/index.md).
3. Build the right mental model in [Core Concepts](core-concepts/index.md).
4. Deepen feature knowledge in [Feature Deep Dives](features/index.md).
5. Adopt a real operating pattern in [Recommended Workflows](workflows/index.md).
6. Study implementation details in [Architecture](architecture/index.md).

## Feature map

| Need | Meridian feature |
| --- | --- |
| Validate static config correctness | [`validate`](features/validate.md) |
| Visualize pipeline wiring | [`graph`](features/graph.md) |
| Review risk between revisions | [`diff`](features/diff.md) |
| Prove runtime flow in a safe harness | [`test`, `check`, and `ci`](features/runtime-commands.md) |
| Lock behavior with output expectations | [Assertions and contracts](features/assertions-and-contracts.md) |
| Inspect run evidence after the fact | [Debug commands](features/debug-commands.md) |
| Put the same checks in pull requests | [GitHub Action](features/github-action.md) |
| Regression test Meridian itself in Kubernetes | [K3s fixture](features/k3s-fixture.md) |

## Workflow at a glance

The recommended path for a config author is:

```text
author config change
  -> meridian validate
  -> meridian check
  -> inspect summary.md / report.json / artifacts
  -> open PR
  -> CI runs meridian ci through the GitHub Action
  -> reviewer inspects PR comment, step summary, and artifact bundle
```

Meridian is most useful when its artifacts are treated as review evidence, not only as a pass or fail bit.
