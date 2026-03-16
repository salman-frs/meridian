# K3s Regression Workflow

Use the repo-owned fixture when validating Meridian itself against the canonical VM environment.

## Run one scenario

```bash
scripts/e2e_k3s_vm.sh happy
```

## Run the full matrix

```bash
scripts/e2e_k3s_vm.sh all
```

## Review scenario output

Artifacts are written under:

```text
/tmp/meridian-e2e-custom/<run-id>/
```

Key files:

- `summary.md`
- `summary.json`
- `logs/*.log`
- `prom-http-requests.json`
- `prom-checkout.json`
- `prom-heartbeat.json`
- `loki-storefront.json`
- `tempo-storefront.json`

Remember that negative scenarios are supposed to pass when the expected failure mode is detected correctly.
