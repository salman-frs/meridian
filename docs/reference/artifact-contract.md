# Artifact Contract

## Runtime bundle contract

### Core files

- `report.json`
- `summary.md`
- `config.patched.yaml`
- `graph.mmd`
- `collector.log`
- `captures/`

### Conditional files

- `config.final.yaml`
- `collector-components.json`
- `semantic-findings.json`
- `graph.svg`
- `diff.md`
- `contracts.json`
- `contracts.md`
- `capture.normalized.json`

## Runtime metadata carried in `report.json`

`report.json` includes:

- config path and runtime config source
- engine and runtime backend
- timings
- findings
- diff results
- graph model
- semantic report
- test plan
- assertions, contracts, and captures
- artifact manifest paths

## `latest` symlink

Meridian maintains `runs/latest` as a symlink to the most recent runtime run under the selected output root.

## K3s fixture contract

Per scenario, the fixture runner preserves:

- `summary.md`
- `summary.json`
- `logs/*.log`
- `kubectl-get-all.txt`
- `events.txt`
- `prom-http-requests.json`
- `prom-checkout.json`
- `prom-heartbeat.json`
- `loki-storefront.json`
- `tempo-storefront.json`
