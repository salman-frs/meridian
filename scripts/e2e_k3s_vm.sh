#!/usr/bin/env bash

set -euo pipefail

SCENARIO="${1:-happy}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXAMPLE_DIR="$ROOT_DIR/examples/k3s-e2e"
RUN_ID="${MERIDIAN_RUN_ID:-meridian-${SCENARIO}-$(date -u +%Y%m%d-%H%M%S)}"
ARTIFACT_DIR="${MERIDIAN_E2E_ARTIFACT_DIR:-/tmp/meridian-e2e-custom/$RUN_ID}"
TMP_DIR="$ARTIFACT_DIR/rendered"
NAMESPACE="${MERIDIAN_E2E_NAMESPACE:-meridian-e2e}"
USER_ID="$(id -u)"
NERDCTL_BIN="${MERIDIAN_NERDCTL_BIN:-${HOME}/bin/nerdctl}"
KUBECTL_BIN="${MERIDIAN_KUBECTL_BIN:-kubectl}"
K3S_IMPORT_CMD="${MERIDIAN_K3S_IMPORT_CMD:-sudo k3s ctr images import}"
ROOTLESS_SETUP_BIN="${MERIDIAN_ROOTLESS_SETUP_BIN:-${HOME}/.local/nerdctl-full/bin/containerd-rootless-setuptool.sh}"
BUILDKIT_HOST="${MERIDIAN_BUILDKIT_HOST:-unix:///run/user/${USER_ID}/buildkit/buildkitd.sock}"

mkdir -p "$ARTIFACT_DIR"
rm -rf "$TMP_DIR"
mkdir -p "$TMP_DIR"

if [[ ! -d "$EXAMPLE_DIR/overlays/$SCENARIO" ]]; then
  echo "unknown scenario: $SCENARIO" >&2
  exit 2
fi

if ! command -v "$KUBECTL_BIN" >/dev/null 2>&1; then
  echo "kubectl not found: $KUBECTL_BIN" >&2
  exit 3
fi

if [[ ! -x "$NERDCTL_BIN" ]]; then
  if command -v nerdctl >/dev/null 2>&1; then
    NERDCTL_BIN="$(command -v nerdctl)"
  else
    echo "nerdctl not found; set MERIDIAN_NERDCTL_BIN" >&2
    exit 3
  fi
fi

TAG="$RUN_ID"
STORE_IMAGE="meridian-k3s-e2e-storefront:$TAG"
CHECKOUT_IMAGE="meridian-k3s-e2e-checkout:$TAG"
INVENTORY_IMAGE="meridian-k3s-e2e-inventory:$TAG"
TRAFFIC_IMAGE="meridian-k3s-e2e-trafficgen:$TAG"

copy_overlay() {
  cp -R "$EXAMPLE_DIR/." "$TMP_DIR/"
  python3 - "$TMP_DIR" "$TAG" <<'PY'
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
tag = sys.argv[2]
for path in root.rglob("*.yaml"):
    text = path.read_text()
    path.write_text(text.replace("IMAGE_TAG", tag))
PY
}

build_image() {
  local image="$1"
  "$NERDCTL_BIN" build \
    -f "$EXAMPLE_DIR/Dockerfile" \
    -t "$image" \
    "$ROOT_DIR"
}

ensure_buildkit() {
  export BUILDKIT_HOST

  if command -v buildctl >/dev/null 2>&1 && buildctl --addr "$BUILDKIT_HOST" debug workers >/dev/null 2>&1; then
    return 0
  fi

  if [[ -x "$ROOTLESS_SETUP_BIN" ]]; then
    "$ROOTLESS_SETUP_BIN" install-buildkit-containerd >/dev/null
    sleep 2
  fi

  if command -v buildctl >/dev/null 2>&1 && buildctl --addr "$BUILDKIT_HOST" debug workers >/dev/null 2>&1; then
    return 0
  fi

  {
    echo "BuildKit is not available for nerdctl. Install it with containerd-rootless-setuptool.sh install-buildkit-containerd or set MERIDIAN_ROOTLESS_SETUP_BIN." >&2
    exit 3
  }
}

import_image() {
  local image="$1"
  local archive="$ARTIFACT_DIR/$(echo "$image" | tr ':/' '__').tar"
  "$NERDCTL_BIN" save -o "$archive" "$image"
  bash -lc "$K3S_IMPORT_CMD '$archive'"
}

svc_ip() {
  "$KUBECTL_BIN" get svc -n observability "$1" -o jsonpath='{.spec.clusterIP}'
}

query_gateway_counter() {
  local metric="$1"
  local ip="$2"
  timeout 25s curl -sf --max-time 20 "http://$ip:8888/metrics" | awk -v name="$metric" '$1 ~ name"{" {print $2}' | tail -n 1
}

retry() {
  local attempts="$1"
  local sleep_seconds="$2"
  shift 2
  local try=1
  until "$@"; do
    if (( try >= attempts )); then
      return 1
    fi
    sleep "$sleep_seconds"
    ((try+=1))
  done
}

query_prometheus() {
  local expr="$1"
  local ip="$2"
  timeout 25s curl -sfG --max-time 20 "http://$ip/api/v1/query" --data-urlencode "query=$expr"
}

query_loki() {
  local namespace="$1"
  local service_name="$2"
  local run_id="$3"
  local ip="$4"
  local end_ns start_ns
  end_ns="$(date +%s%N)"
  start_ns="$(( end_ns - 900000000000 ))"
  timeout 25s curl -sfG --max-time 20 "http://$ip/loki/api/v1/query_range" \
    --data-urlencode "query={k8s_namespace_name=\"${namespace}\",service_name=\"${service_name}\"} |= \"\\\"run_id\\\":\\\"${run_id}\\\"\"" \
    --data-urlencode "start=$start_ns" \
    --data-urlencode "end=$end_ns" \
    --data-urlencode "limit=20" \
    --data-urlencode "direction=backward"
}

query_tempo() {
  local ip="$1"
  local service_name="$2"
  timeout 25s curl -sfG --max-time 20 "http://$ip:3200/api/search" \
    --data-urlencode 'limit=20' \
    --data-urlencode "q={resource.service.name=\"$service_name\"}"
}

wait_for_deployments() {
  "$KUBECTL_BIN" rollout status deployment/storefront -n "$NAMESPACE" --timeout=180s
  "$KUBECTL_BIN" rollout status deployment/checkout -n "$NAMESPACE" --timeout=180s
  "$KUBECTL_BIN" rollout status deployment/inventory -n "$NAMESPACE" --timeout=180s
}

collect_logs() {
  mkdir -p "$ARTIFACT_DIR/logs"
  for item in storefront checkout inventory trafficgen; do
    "$KUBECTL_BIN" logs -n "$NAMESPACE" $( [[ "$item" == "trafficgen" ]] && echo "job/$item" || echo "deployment/$item" ) >"$ARTIFACT_DIR/logs/$item.log" 2>&1 || true
  done
  "$KUBECTL_BIN" get all -n "$NAMESPACE" >"$ARTIFACT_DIR/kubectl-get-all.txt" 2>&1 || true
  "$KUBECTL_BIN" get events -n "$NAMESPACE" --sort-by=.lastTimestamp >"$ARTIFACT_DIR/events.txt" 2>&1 || true
}

cleanup_namespace() {
  "$KUBECTL_BIN" delete namespace "$NAMESPACE" --ignore-not-found=true --wait=true --timeout=180s >/dev/null 2>&1 || true
}

copy_overlay
cleanup_namespace
ensure_buildkit

build_image "$STORE_IMAGE"
build_image "$CHECKOUT_IMAGE"
build_image "$INVENTORY_IMAGE"
build_image "$TRAFFIC_IMAGE"

import_image "$STORE_IMAGE"
import_image "$CHECKOUT_IMAGE"
import_image "$INVENTORY_IMAGE"
import_image "$TRAFFIC_IMAGE"

"$KUBECTL_BIN" apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: $NAMESPACE
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: meridian-e2e-run
  namespace: $NAMESPACE
data:
  MERIDIAN_RUN_ID: $RUN_ID
EOF

GATEWAY_IP="$(svc_ip otel-gateway)"
LOKI_IP="$(svc_ip loki)"
PROM_IP="$(svc_ip prometheus-server)"
TEMPO_IP="$(svc_ip tempo)"

SPAN_BEFORE="$(query_gateway_counter otelcol_exporter_sent_spans_total "$GATEWAY_IP" || echo 0)"
LOG_BEFORE="$(query_gateway_counter otelcol_exporter_sent_log_records_total "$GATEWAY_IP" || echo 0)"
METRIC_BEFORE="$(query_gateway_counter otelcol_exporter_sent_metric_points_total "$GATEWAY_IP" || echo 0)"

"$KUBECTL_BIN" apply -k "$TMP_DIR/overlays/$SCENARIO"

wait_for_deployments
retry 12 5 "$KUBECTL_BIN" get endpoints storefront -n "$NAMESPACE" -o jsonpath='{.subsets[0].addresses[0].ip}' >/dev/null
"$KUBECTL_BIN" delete job trafficgen -n "$NAMESPACE" --ignore-not-found=true --wait=true >/dev/null 2>&1 || true
if ! "$KUBECTL_BIN" create -n "$NAMESPACE" -f "$TMP_DIR/base/trafficgen-job.yaml"; then
  "$KUBECTL_BIN" replace --force -n "$NAMESPACE" -f "$TMP_DIR/base/trafficgen-job.yaml"
fi
retry 12 2 "$KUBECTL_BIN" get job trafficgen -n "$NAMESPACE" >/dev/null
"$KUBECTL_BIN" wait --for=condition=complete job/trafficgen -n "$NAMESPACE" --timeout=180s
sleep 20

SPAN_AFTER="$(query_gateway_counter otelcol_exporter_sent_spans_total "$GATEWAY_IP" || echo 0)"
LOG_AFTER="$(query_gateway_counter otelcol_exporter_sent_log_records_total "$GATEWAY_IP" || echo 0)"
METRIC_AFTER="$(query_gateway_counter otelcol_exporter_sent_metric_points_total "$GATEWAY_IP" || echo 0)"

retry 6 5 query_prometheus "meridian_http_requests_total{run_id=\"$RUN_ID\"}" "$PROM_IP" >"$ARTIFACT_DIR/prom-http-requests.json"
retry 6 5 query_prometheus "meridian_checkout_total{run_id=\"$RUN_ID\"}" "$PROM_IP" >"$ARTIFACT_DIR/prom-checkout.json" || true
retry 6 5 query_prometheus "meridian_inventory_heartbeat_total{run_id=\"$RUN_ID\"}" "$PROM_IP" >"$ARTIFACT_DIR/prom-heartbeat.json"
retry 18 5 query_loki "$NAMESPACE" storefront "$RUN_ID" "$LOKI_IP" >"$ARTIFACT_DIR/loki-storefront.json" || true
retry 6 5 query_tempo "$TEMPO_IP" storefront >"$ARTIFACT_DIR/tempo-storefront.json" || true

collect_logs

python3 - "$ARTIFACT_DIR" "$SCENARIO" "$RUN_ID" "$SPAN_BEFORE" "$SPAN_AFTER" "$LOG_BEFORE" "$LOG_AFTER" "$METRIC_BEFORE" "$METRIC_AFTER" <<'PY'
import json
import pathlib
import sys

artifact_dir = pathlib.Path(sys.argv[1])
scenario = sys.argv[2]
run_id = sys.argv[3]
span_before, span_after = map(float, sys.argv[4:6])
log_before, log_after = map(float, sys.argv[6:8])
metric_before, metric_after = map(float, sys.argv[8:10])

def load(path):
    file_path = artifact_dir / path
    if not file_path.exists():
        return None
    text = file_path.read_text().strip()
    if not text:
        return None
    return json.loads(text)

prom_http = load("prom-http-requests.json")
prom_checkout = load("prom-checkout.json")
prom_heartbeat = load("prom-heartbeat.json")
loki = load("loki-storefront.json")
tempo = load("tempo-storefront.json")

def has_prom_result(payload):
    if not payload:
        return False
    return bool(payload.get("data", {}).get("result"))

def has_loki_result(payload):
    if not payload:
        return False
    return bool(payload.get("data", {}).get("result"))

tempo_blob = json.dumps(tempo or {})
tempo_hit = "traceID" in tempo_blob or '"traces"' in tempo_blob or '"service.name"' in tempo_blob

logs_text = ""
for name in ("storefront.log", "checkout.log", "inventory.log", "trafficgen.log"):
    path = artifact_dir / "logs" / name
    if path.exists():
        logs_text += path.read_text()

summary = {
    "run_id": run_id,
    "scenario": scenario,
    "gateway_deltas": {
        "spans": span_after - span_before,
        "logs": log_after - log_before,
        "metrics": metric_after - metric_before,
    },
    "prometheus": {
        "http_requests": has_prom_result(prom_http),
        "checkout": has_prom_result(prom_checkout),
        "heartbeat": has_prom_result(prom_heartbeat),
    },
    "loki": {
        "storefront_run_logs": has_loki_result(loki),
    },
    "tempo": {
        "storefront": tempo_hit,
    },
}

failures = []

if scenario == "happy":
    if summary["gateway_deltas"]["spans"] <= 0:
        failures.append("expected spans delta > 0")
    if summary["gateway_deltas"]["logs"] <= 0:
        failures.append("expected logs delta > 0")
    if summary["gateway_deltas"]["metrics"] <= 0:
        failures.append("expected metrics delta > 0")
    if not summary["prometheus"]["http_requests"]:
        failures.append("expected Prometheus meridian_http_requests_total result")
    if not summary["prometheus"]["heartbeat"]:
        failures.append("expected Prometheus meridian_inventory_heartbeat_total result")
    if not summary["loki"]["storefront_run_logs"]:
        failures.append("expected Loki run logs")
    if not summary["tempo"]["storefront"]:
        failures.append("expected Tempo storefront traces")
elif scenario == "drop-traces":
    if summary["prometheus"]["http_requests"] is False:
        failures.append("expected Prometheus http_requests result")
    if summary["loki"]["storefront_run_logs"] is False:
        failures.append("expected Loki run logs")
    if summary["tempo"]["storefront"]:
        failures.append("expected Tempo storefront traces to be absent")
elif scenario == "misroute-logs":
    if not summary["prometheus"]["http_requests"]:
        failures.append("expected Prometheus http_requests result")
    if not summary["tempo"]["storefront"]:
        failures.append("expected Tempo storefront traces")
    if summary["loki"]["storefront_run_logs"]:
        failures.append("expected Loki run logs to be absent")
elif scenario == "auth-fail":
    if "auth_error" not in logs_text:
        failures.append("expected auth_error in pod logs")
    if not summary["prometheus"]["http_requests"]:
        failures.append("expected storefront Prometheus traffic result")
elif scenario == "backend-unreachable":
    if "otel_error" not in logs_text:
        failures.append("expected otel_error in pod logs")
    if summary["prometheus"]["http_requests"]:
        failures.append("expected Prometheus http_requests result to be absent")
else:
    failures.append(f"unsupported scenario validation: {scenario}")

summary["failures"] = failures
(artifact_dir / "summary.json").write_text(json.dumps(summary, indent=2))

lines = [
    f"# Meridian K3s E2E Summary",
    "",
    f"- run_id: `{run_id}`",
    f"- scenario: `{scenario}`",
    f"- spans delta: `{summary['gateway_deltas']['spans']}`",
    f"- logs delta: `{summary['gateway_deltas']['logs']}`",
    f"- metrics delta: `{summary['gateway_deltas']['metrics']}`",
    f"- prometheus http_requests: `{summary['prometheus']['http_requests']}`",
    f"- prometheus heartbeat: `{summary['prometheus']['heartbeat']}`",
    f"- loki storefront logs: `{summary['loki']['storefront_run_logs']}`",
    f"- tempo storefront traces: `{summary['tempo']['storefront']}`",
]
if failures:
    lines.extend(["", "## Failures"])
    lines.extend([f"- {item}" for item in failures])
else:
    lines.extend(["", "## Result", "- PASS"])
(artifact_dir / "summary.md").write_text("\n".join(lines) + "\n")

if scenario == "backend-unreachable" and failures:
    sys.exit(1)
if failures:
    sys.exit(2)
PY

cat "$ARTIFACT_DIR/summary.md"
