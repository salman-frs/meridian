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

To run the full regression matrix on the VM:

```bash
scripts/e2e_k3s_vm.sh all
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

The summary artifacts are the authoritative acceptance record for the fixture. On this VM, pod logs plus Prometheus and gateway-counter evidence are the primary gates. Tempo and Loki artifacts are still collected when available, but they are not the blocking success signal for every negative scenario because the backend queries are not fully run-scoped.

## Scenarios

- `happy`
- `drop-traces`
- `misroute-logs`
- `auth-fail`
- `backend-unreachable`

Each scenario exits `0` when its expected behavior is observed, including negative scenarios. A fault-injection scenario is considered a successful test when the expected failure mode is detected correctly.

## Images

The runner builds local images with `nerdctl`, saves them, and imports them into k3s containerd before deployment:

- `meridian-k3s-e2e-storefront`
- `meridian-k3s-e2e-checkout`
- `meridian-k3s-e2e-inventory`
- `meridian-k3s-e2e-trafficgen`

## Official OTel Demo

The official OpenTelemetry Demo is no longer a blocking acceptance gate for Meridian on this VM. It remains optional for manual compatibility checks only.
