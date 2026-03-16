# K3s Fixture

The repository-owned fixture under [`examples/k3s-e2e/`](https://github.com/salman-frs/meridian/tree/main/examples/k3s-e2e) is Meridian's authoritative real-cluster regression target.

## Purpose

The fixture exists to validate Meridian itself against a small, editable, deterministic Kubernetes stack.

It is not intended to become a generic Kubernetes testing framework.

## Fixture roles

- `storefront`: entrypoint service
- `checkout`: downstream request handler
- `inventory`: reserve handler plus heartbeat metrics
- `trafficgen`: repeatable workload driver

## Scenarios

- `happy`
- `drop-traces`
- `misroute-logs`
- `auth-fail`
- `backend-unreachable`

Negative scenarios are successful when the expected failure mode is detected correctly.

## Primary acceptance signals

On the project VM, the blocking signals are:

- pod logs
- Prometheus evidence
- gateway counters

Tempo and Loki artifacts remain supporting evidence when available, but they are not always run-scoped enough to be the only acceptance gate.
