# K3s E2E Stack

Meridian's canonical Kubernetes end-to-end fixture is the repo-owned stack under [`examples/k3s-e2e/`](../examples/k3s-e2e/).

This stack is intentionally small and editable:

- `storefront`: HTTP entrypoint with `/healthz`, `/browse`, and `/checkout`
- `checkout`: downstream checkout handler
- `inventory`: downstream reserve handler plus heartbeat metrics
- `trafficgen`: repeatable job that drives `/browse` and `/checkout`

The stack sends traces and metrics to the existing `observability/otel-gateway` path and writes structured JSON logs to stdout so the cluster log pipeline can forward them to Loki.

## Run On The VM

```bash
scripts/e2e_k3s_vm.sh happy
```

Artifacts are written to:

```bash
/tmp/meridian-e2e-custom/<run-id>/
```

Important files:

- `summary.md`
- `summary.json`
- `logs/*.log`
- `prom-http-requests.json`
- `prom-heartbeat.json`
- `loki-storefront.json`
- `tempo-storefront.json`

## Scenarios

- `happy`
- `drop-traces`
- `misroute-logs`
- `auth-fail`
- `backend-unreachable`

`backend-unreachable` is expected to exit non-zero after it captures exporter failures and missing backend evidence.

## Images

The runner builds local images with `nerdctl`, saves them, and imports them into k3s containerd before deployment:

- `meridian-k3s-e2e-storefront`
- `meridian-k3s-e2e-checkout`
- `meridian-k3s-e2e-inventory`
- `meridian-k3s-e2e-trafficgen`

## Official OTel Demo

The official OpenTelemetry Demo is no longer a blocking acceptance gate for Meridian on this VM. It remains optional for manual compatibility checks only.
