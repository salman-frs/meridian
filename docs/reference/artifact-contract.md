# Artifact Contract

This page documents the runtime bundle and k3s fixture contracts that Meridian currently preserves.

## Runtime bundle

Standard files:

- `report.json`
- `summary.md`
- `config.patched.yaml`
- `graph.mmd`
- `collector.log`
- `captures/`

Conditional files:

- `config.final.yaml` when Collector `print-config` succeeded
- `collector-components.json` when component inventory was collected
- `semantic-findings.json` when semantic findings exist
- `graph.svg` when SVG rendering is requested and Graphviz is available
- `diff.md` when diff data exists
- `contracts.json`
- `contracts.md`
- `capture.normalized.json`

## Runtime provenance fields

`report.json` records:

- original config source
- runtime config source
- engine and runtime backend
- timings
- findings, diff, graph, semantic, assertions, contracts, and captures
- bundle paths

## `latest` symlink

Meridian maintains `runs/latest` as a symlink to the most recent runtime run under the selected output root.

## K3s fixture artifacts

Per scenario, the VM runner preserves:

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
