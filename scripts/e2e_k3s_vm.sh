#!/usr/bin/env bash

set -euo pipefail

SCENARIO="${1:-happy}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXAMPLE_DIR="$ROOT_DIR/examples/k3s-e2e"
BASE_RUN_ID="${MERIDIAN_RUN_ID:-meridian-${SCENARIO}-$(date -u +%Y%m%d-%H%M%S)}"
NAMESPACE="${MERIDIAN_E2E_NAMESPACE:-meridian-e2e}"
USER_ID="$(id -u)"
ROOTLESS_BIN_DIR="${MERIDIAN_ROOTLESS_BIN_DIR:-${HOME}/.local/nerdctl-full/bin}"
NERDCTL_BIN="${MERIDIAN_NERDCTL_BIN:-${HOME}/bin/nerdctl}"
KUBECTL_BIN="${MERIDIAN_KUBECTL_BIN:-kubectl}"
K3S_IMPORT_CMD="${MERIDIAN_K3S_IMPORT_CMD:-k3s ctr images import}"
ROOTLESS_SETUP_BIN="${MERIDIAN_ROOTLESS_SETUP_BIN:-${HOME}/.local/nerdctl-full/bin/containerd-rootless-setuptool.sh}"
BUILDKIT_HOST="${MERIDIAN_BUILDKIT_HOST:-unix:///run/user/${USER_ID}/buildkit/buildkitd.sock}"
SUDO_PASSWORD="${MERIDIAN_SUDO_PASSWORD:-}"
SCENARIO_MATRIX=(happy drop-traces misroute-logs auth-fail backend-unreachable)

copy_overlay() {
  local destination="$1"
  local tag="$2"
  cp -R "$EXAMPLE_DIR/." "$destination/"
  python3 - "$destination" "$tag" <<'PY'
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
tag = sys.argv[2]
for path in root.rglob("*.yaml"):
    text = path.read_text()
    path.write_text(text.replace("IMAGE_TAG", tag))
PY
}

run_sudo() {
  if sudo -n true >/dev/null 2>&1; then
    sudo "$@"
    return
  fi
  if [[ -n "$SUDO_PASSWORD" ]]; then
    printf '%s\n' "$SUDO_PASSWORD" | sudo -S "$@"
    return
  fi
  sudo "$@"
}

preflight() {
  if [[ -d "$ROOTLESS_BIN_DIR" ]]; then
    export PATH="$ROOTLESS_BIN_DIR:$PATH"
  fi

  if [[ ! -d "$EXAMPLE_DIR/overlays/happy" ]]; then
    echo "missing examples/k3s-e2e overlays" >&2
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

  if ! command -v python3 >/dev/null 2>&1; then
    echo "python3 not found" >&2
    exit 3
  fi
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

  echo "BuildKit is not available for nerdctl. Install it with containerd-rootless-setuptool.sh install-buildkit-containerd or set MERIDIAN_ROOTLESS_SETUP_BIN." >&2
  exit 3
}

build_image() {
  local image="$1"
  "$NERDCTL_BIN" build -f "$EXAMPLE_DIR/Dockerfile" -t "$image" "$ROOT_DIR"
}

import_image() {
  local image="$1"
  local artifact_dir="$2"
  local archive="$artifact_dir/$(echo "$image" | tr ':/' '__').tar"
  "$NERDCTL_BIN" save -o "$archive" "$image"
  run_sudo bash -lc "$K3S_IMPORT_CMD '$archive'"
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
    ((try += 1))
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
  timeout 10s curl -sfG --max-time 8 "http://$ip/loki/api/v1/query_range" \
    --data-urlencode "query={k8s_namespace_name=\"${namespace}\",service_name=\"${service_name}\"} |= \"\\\"run_id\\\":\\\"${run_id}\\\"\"" \
    --data-urlencode "start=$start_ns" \
    --data-urlencode "end=$end_ns" \
    --data-urlencode "limit=20" \
    --data-urlencode "direction=backward"
}

query_tempo() {
  local ip="$1"
  local service_name="$2"
  timeout 12s curl -sfG --max-time 10 "http://$ip:3200/api/search" \
    --data-urlencode 'limit=20' \
    --data-urlencode "q={resource.service.name=\"$service_name\"}"
}

wait_for_deployments() {
  "$KUBECTL_BIN" rollout status deployment/storefront -n "$NAMESPACE" --timeout=180s
  "$KUBECTL_BIN" rollout status deployment/checkout -n "$NAMESPACE" --timeout=180s
  "$KUBECTL_BIN" rollout status deployment/inventory -n "$NAMESPACE" --timeout=180s
}

collect_logs() {
  local artifact_dir="$1"
  mkdir -p "$artifact_dir/logs"
  for item in storefront checkout inventory trafficgen; do
    "$KUBECTL_BIN" logs -n "$NAMESPACE" $( [[ "$item" == "trafficgen" ]] && echo "job/$item" || echo "deployment/$item" ) >"$artifact_dir/logs/$item.log" 2>&1 || true
  done
  "$KUBECTL_BIN" get all -n "$NAMESPACE" >"$artifact_dir/kubectl-get-all.txt" 2>&1 || true
  "$KUBECTL_BIN" get events -n "$NAMESPACE" --sort-by=.lastTimestamp >"$artifact_dir/events.txt" 2>&1 || true
}

cleanup_namespace() {
  "$KUBECTL_BIN" delete namespace "$NAMESPACE" --ignore-not-found=true --wait=true --timeout=180s >/dev/null 2>&1 || true
}

prepare_images() {
  local tag="$1"
  local artifact_dir="$2"
  local store_image="meridian-k3s-e2e-storefront:$tag"
  local checkout_image="meridian-k3s-e2e-checkout:$tag"
  local inventory_image="meridian-k3s-e2e-inventory:$tag"
  local traffic_image="meridian-k3s-e2e-trafficgen:$tag"

  mkdir -p "$artifact_dir"
  ensure_buildkit
  build_image "$store_image"
  build_image "$checkout_image"
  build_image "$inventory_image"
  build_image "$traffic_image"

  import_image "$store_image" "$artifact_dir"
  import_image "$checkout_image" "$artifact_dir"
  import_image "$inventory_image" "$artifact_dir"
  import_image "$traffic_image" "$artifact_dir"
}

create_run_namespace() {
  local run_id="$1"
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
  MERIDIAN_RUN_ID: $run_id
EOF
}

collect_backend_evidence() {
  local artifact_dir="$1"
  local run_id="$2"
  local loki_ip="$3"
  local prom_ip="$4"
  local tempo_ip="$5"

  retry 6 5 query_prometheus "meridian_http_requests_total{run_id=\"$run_id\"}" "$prom_ip" >"$artifact_dir/prom-http-requests.json" || true
  retry 6 5 query_prometheus "meridian_checkout_total{run_id=\"$run_id\"}" "$prom_ip" >"$artifact_dir/prom-checkout.json" || true
  retry 6 5 query_prometheus "meridian_inventory_heartbeat_total{run_id=\"$run_id\"}" "$prom_ip" >"$artifact_dir/prom-heartbeat.json" || true
  retry 2 2 query_loki "$NAMESPACE" storefront "$run_id" "$loki_ip" >"$artifact_dir/loki-storefront.json" || true
  retry 3 3 query_tempo "$tempo_ip" storefront >"$artifact_dir/tempo-storefront.json" || true
}

evaluate_scenario() {
  local artifact_dir="$1"
  local scenario="$2"
  local run_id="$3"
  local span_before="$4"
  local span_after="$5"
  local log_before="$6"
  local log_after="$7"
  local metric_before="$8"
  local metric_after="$9"

  python3 "$ROOT_DIR/scripts/e2e_k3s_summary.py" \
    "$artifact_dir" "$scenario" "$run_id" \
    "$span_before" "$span_after" \
    "$log_before" "$log_after" \
    "$metric_before" "$metric_after"
}

run_single_scenario() {
  local scenario="$1"
  local tag="$2"
  local artifact_dir="$3"
  local rendered_dir="$artifact_dir/rendered"
  local run_id="${tag}-${scenario}"

  mkdir -p "$artifact_dir"
  rm -rf "$rendered_dir"
  mkdir -p "$rendered_dir"

  if [[ ! -d "$EXAMPLE_DIR/overlays/$scenario" ]]; then
    echo "unknown scenario: $scenario" >&2
    return 2
  fi

  copy_overlay "$rendered_dir" "$tag"
  cleanup_namespace
  create_run_namespace "$run_id"

  local gateway_ip loki_ip prom_ip tempo_ip
  gateway_ip="$(svc_ip otel-gateway)"
  loki_ip="$(svc_ip loki)"
  prom_ip="$(svc_ip prometheus-server)"
  tempo_ip="$(svc_ip tempo)"

  local span_before log_before metric_before
  span_before="$(query_gateway_counter otelcol_exporter_sent_spans_total "$gateway_ip" || echo 0)"
  log_before="$(query_gateway_counter otelcol_exporter_sent_log_records_total "$gateway_ip" || echo 0)"
  metric_before="$(query_gateway_counter otelcol_exporter_sent_metric_points_total "$gateway_ip" || echo 0)"

  "$KUBECTL_BIN" apply -k "$rendered_dir/overlays/$scenario"
  wait_for_deployments
  retry 12 5 "$KUBECTL_BIN" get endpoints storefront -n "$NAMESPACE" -o jsonpath='{.subsets[0].addresses[0].ip}' >/dev/null
  "$KUBECTL_BIN" delete job trafficgen -n "$NAMESPACE" --ignore-not-found=true --wait=true >/dev/null 2>&1 || true
  if ! "$KUBECTL_BIN" create -n "$NAMESPACE" -f "$rendered_dir/base/trafficgen-job.yaml"; then
    "$KUBECTL_BIN" replace --force -n "$NAMESPACE" -f "$rendered_dir/base/trafficgen-job.yaml"
  fi
  retry 12 2 "$KUBECTL_BIN" get job trafficgen -n "$NAMESPACE" >/dev/null
  "$KUBECTL_BIN" wait --for=condition=complete job/trafficgen -n "$NAMESPACE" --timeout=180s
  sleep 20

  local span_after log_after metric_after
  span_after="$(query_gateway_counter otelcol_exporter_sent_spans_total "$gateway_ip" || echo 0)"
  log_after="$(query_gateway_counter otelcol_exporter_sent_log_records_total "$gateway_ip" || echo 0)"
  metric_after="$(query_gateway_counter otelcol_exporter_sent_metric_points_total "$gateway_ip" || echo 0)"

  collect_backend_evidence "$artifact_dir" "$run_id" "$loki_ip" "$prom_ip" "$tempo_ip"
  collect_logs "$artifact_dir"

  evaluate_scenario "$artifact_dir" "$scenario" "$run_id" \
    "$span_before" "$span_after" \
    "$log_before" "$log_after" \
    "$metric_before" "$metric_after"
  python3 - "$artifact_dir/summary.json" <<'PY'
import json
import pathlib
import sys

summary = json.loads(pathlib.Path(sys.argv[1]).read_text())
counts = summary.get("contract_summary", {})
print(
    "Contract summary: "
    f"{counts.get('pass', 0)}/{counts.get('total', 0)} passed "
    f"({counts.get('fail', 0)} failed)"
)
PY
  cat "$artifact_dir/summary.md"
  python3 - "$artifact_dir/summary.json" <<'PY'
import json
import pathlib
import sys

summary = json.loads(pathlib.Path(sys.argv[1]).read_text())
raise SystemExit(0 if summary.get("result") == "PASS" else 2)
PY
}

run_matrix() {
  local matrix_dir="$1"
  local tag="$2"
  local failed=0

  mkdir -p "$matrix_dir"
  for scenario in "${SCENARIO_MATRIX[@]}"; do
    echo "=== Running scenario: $scenario ==="
    if ! run_single_scenario "$scenario" "$tag" "$matrix_dir/$scenario"; then
      failed=1
    fi
  done
  return "$failed"
}

main() {
  preflight

  local artifact_root
  if [[ "$SCENARIO" == "all" ]]; then
    artifact_root="${MERIDIAN_E2E_ARTIFACT_DIR:-/tmp/meridian-e2e-custom/$BASE_RUN_ID}"
    prepare_images "$BASE_RUN_ID" "$artifact_root"
    run_matrix "$artifact_root" "$BASE_RUN_ID"
    return
  fi

  artifact_root="${MERIDIAN_E2E_ARTIFACT_DIR:-/tmp/meridian-e2e-custom/$BASE_RUN_ID}"
  prepare_images "$BASE_RUN_ID" "$artifact_root"
  run_single_scenario "$SCENARIO" "$BASE_RUN_ID" "$artifact_root"
}

main "$@"
