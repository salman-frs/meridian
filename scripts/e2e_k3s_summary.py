#!/usr/bin/env python3

from __future__ import annotations

import json
import pathlib
import sys
from typing import Any


SCENARIO_CONTRACTS: dict[str, dict[str, Any]] = {
    "happy": {
        "gateway_positive": {"spans": True, "logs": True, "metrics": True},
        "prometheus": {"http_requests": True, "checkout": True, "heartbeat": True},
        "tempo": {"storefront": True},
        "app_events": {
            "storefront": ["browse_request", "checkout_request"],
            "checkout": ["checkout_processed"],
            "inventory": ["inventory_reserved", "inventory_heartbeat"],
            "trafficgen": ["trafficgen_started", "trafficgen_completed"],
        },
        "error_events_absent": ["auth_error", "otel_error"],
    },
    "drop-traces": {
        "gateway_positive": {"logs": True, "metrics": True},
        "gateway_zero": {"spans": True},
        "prometheus": {"http_requests": True, "checkout": True, "heartbeat": True},
        "app_events": {
            "storefront": ["browse_request", "checkout_request"],
            "checkout": ["checkout_processed"],
            "inventory": ["inventory_reserved"],
            "trafficgen": ["trafficgen_completed"],
        },
        "error_events_absent": ["auth_error", "otel_error"],
    },
    "misroute-logs": {
        "gateway_positive": {"spans": True, "metrics": True},
        "prometheus": {"http_requests": True, "checkout": True, "heartbeat": True},
        "loki": {"storefront_run_logs": False},
        "tempo": {"storefront": True},
        "app_logs_present": {
            "storefront": False,
            "checkout": False,
            "inventory": False,
            "trafficgen": False,
        },
        "error_events_absent": ["auth_error", "otel_error"],
    },
    "auth-fail": {
        "gateway_positive": {"metrics": True},
        "prometheus": {"http_requests": True, "heartbeat": True},
        "app_events": {
            "storefront": ["checkout_dependency_error"],
            "inventory": ["auth_error"],
            "trafficgen": ["trafficgen_completed"],
        },
        "app_events_absent": {
            "checkout": ["checkout_processed"],
        },
        "error_events_present": ["auth_error"],
    },
    "backend-unreachable": {
        "prometheus": {"http_requests": False, "checkout": False, "heartbeat": False},
        "loki": {"storefront_run_logs": False},
        "gateway_zero": {"spans": True},
        "app_events": {
            "storefront": ["browse_request", "checkout_request"],
            "checkout": ["checkout_processed"],
            "inventory": ["inventory_reserved"],
            "trafficgen": ["trafficgen_completed"],
        },
        "error_events_present": ["otel_error"],
    },
}


def load_json(path: pathlib.Path) -> Any | None:
    if not path.exists():
        return None
    text = path.read_text().strip()
    if not text:
        return None
    return json.loads(text)


def has_prom_result(payload: Any | None) -> bool:
    if not payload:
        return False
    return bool(payload.get("data", {}).get("result"))


def has_loki_result(payload: Any | None) -> bool:
    if not payload:
        return False
    return bool(payload.get("data", {}).get("result"))


def has_tempo_result(payload: Any | None) -> bool:
    if not payload:
        return False
    traces = payload.get("traces")
    if isinstance(traces, list):
        return len(traces) > 0
    blob = json.dumps(payload)
    return "traceID" in blob or '"service.name"' in blob


def service_logs(artifact_dir: pathlib.Path) -> dict[str, str]:
    logs_dir = artifact_dir / "logs"
    return {
        "storefront": read_text(logs_dir / "storefront.log"),
        "checkout": read_text(logs_dir / "checkout.log"),
        "inventory": read_text(logs_dir / "inventory.log"),
        "trafficgen": read_text(logs_dir / "trafficgen.log"),
    }


def read_text(path: pathlib.Path) -> str:
    if not path.exists():
        return ""
    return path.read_text()


def build_observed(artifact_dir: pathlib.Path, deltas: dict[str, float]) -> dict[str, Any]:
    prom_http = load_json(artifact_dir / "prom-http-requests.json")
    prom_checkout = load_json(artifact_dir / "prom-checkout.json")
    prom_heartbeat = load_json(artifact_dir / "prom-heartbeat.json")
    loki = load_json(artifact_dir / "loki-storefront.json")
    tempo = load_json(artifact_dir / "tempo-storefront.json")
    logs = service_logs(artifact_dir)
    combined_logs = "\n".join(logs.values())

    return {
        "gateway_deltas": deltas,
        "prometheus": {
            "http_requests": has_prom_result(prom_http),
            "checkout": has_prom_result(prom_checkout),
            "heartbeat": has_prom_result(prom_heartbeat),
        },
        "loki": {
            "storefront_run_logs": has_loki_result(loki),
        },
        "tempo": {
            "storefront": has_tempo_result(tempo),
        },
        "app_logs": {
            service: {
                "text_present": bool(text.strip()),
                "events": collect_events(text),
            }
            for service, text in logs.items()
        },
        "combined_log_text": combined_logs,
    }


def collect_events(text: str) -> list[str]:
    events: list[str] = []
    for line in text.splitlines():
        try:
            payload = json.loads(line)
        except json.JSONDecodeError:
            continue
        event = payload.get("event")
        if isinstance(event, str):
            events.append(event)
    return events


def evaluate_contract(scenario: str, observed: dict[str, Any]) -> list[str]:
    contract = SCENARIO_CONTRACTS[scenario]
    failures: list[str] = []

    for key, expected in contract.get("gateway_positive", {}).items():
        delta = observed["gateway_deltas"].get(key, 0)
        if expected and delta <= 0:
            failures.append(f"expected gateway delta for {key} > 0")
        if expected is False and delta > 0:
            failures.append(f"expected gateway delta for {key} to stay <= 0")

    for key, expected in contract.get("gateway_zero", {}).items():
        delta = observed["gateway_deltas"].get(key, 0)
        if expected and delta != 0:
            failures.append(f"expected gateway delta for {key} == 0, got {delta}")

    for section in ("prometheus", "loki", "tempo"):
        for key, expected in contract.get(section, {}).items():
            actual = observed[section].get(key)
            if actual != expected:
                failures.append(f"expected {section}.{key} == {expected}, got {actual}")

    for service, events in contract.get("app_events", {}).items():
        actual_events = set(observed["app_logs"].get(service, {}).get("events", []))
        for event in events:
            if event not in actual_events:
                failures.append(f"expected {service} log event {event}")

    for service, events in contract.get("app_events_absent", {}).items():
        actual_events = set(observed["app_logs"].get(service, {}).get("events", []))
        for event in events:
            if event in actual_events:
                failures.append(f"expected {service} log event {event} to be absent")

    for service, expected in contract.get("app_logs_present", {}).items():
        actual = observed["app_logs"].get(service, {}).get("text_present")
        if actual != expected:
            failures.append(f"expected {service} log presence == {expected}, got {actual}")

    combined = observed["combined_log_text"]
    for event in contract.get("error_events_present", []):
        if event not in combined:
            failures.append(f"expected combined logs to contain {event}")
    for event in contract.get("error_events_absent", []):
        if event in combined:
            failures.append(f"expected combined logs to exclude {event}")

    return failures


def build_summary(artifact_dir: pathlib.Path, scenario: str, run_id: str, deltas: dict[str, float]) -> dict[str, Any]:
    observed = build_observed(artifact_dir, deltas)
    failures = evaluate_contract(scenario, observed)
    return {
        "run_id": run_id,
        "scenario": scenario,
        "result": "PASS" if not failures else "FAIL",
        "expected": SCENARIO_CONTRACTS[scenario],
        "observed": {
            "gateway_deltas": observed["gateway_deltas"],
            "prometheus": observed["prometheus"],
            "loki": observed["loki"],
            "tempo": observed["tempo"],
            "app_logs": {
                service: {
                    "text_present": payload["text_present"],
                    "events": payload["events"],
                }
                for service, payload in observed["app_logs"].items()
            },
        },
        "failures": failures,
    }


def render_markdown(summary: dict[str, Any]) -> str:
    observed = summary["observed"]
    lines = [
        "# Meridian K3s E2E Summary",
        "",
        f"- run_id: `{summary['run_id']}`",
        f"- scenario: `{summary['scenario']}`",
        f"- result: `{summary['result']}`",
        f"- spans delta: `{observed['gateway_deltas'].get('spans', 0)}`",
        f"- logs delta: `{observed['gateway_deltas'].get('logs', 0)}`",
        f"- metrics delta: `{observed['gateway_deltas'].get('metrics', 0)}`",
        f"- prometheus http_requests: `{observed['prometheus']['http_requests']}`",
        f"- prometheus checkout: `{observed['prometheus']['checkout']}`",
        f"- prometheus heartbeat: `{observed['prometheus']['heartbeat']}`",
        f"- loki storefront logs: `{observed['loki']['storefront_run_logs']}`",
        f"- tempo storefront traces: `{observed['tempo']['storefront']}`",
    ]

    for service, payload in observed["app_logs"].items():
        events = ", ".join(payload["events"]) if payload["events"] else "none"
        lines.append(f"- {service} events: `{events}`")

    if summary["failures"]:
        lines.extend(["", "## Failures"])
        lines.extend([f"- {item}" for item in summary["failures"]])
    else:
        lines.extend(["", "## Result", "- PASS"])
    return "\n".join(lines) + "\n"


def main(argv: list[str]) -> int:
    if len(argv) != 10:
        raise SystemExit("usage: e2e_k3s_summary.py <artifact_dir> <scenario> <run_id> <span_before> <span_after> <log_before> <log_after> <metric_before> <metric_after>")

    artifact_dir = pathlib.Path(argv[1])
    scenario = argv[2]
    run_id = argv[3]
    if scenario not in SCENARIO_CONTRACTS:
        raise SystemExit(f"unsupported scenario: {scenario}")

    span_before, span_after = map(float, argv[4:6])
    log_before, log_after = map(float, argv[6:8])
    metric_before, metric_after = map(float, argv[8:10])
    deltas = {
        "spans": span_after - span_before,
        "logs": log_after - log_before,
        "metrics": metric_after - metric_before,
    }

    summary = build_summary(artifact_dir, scenario, run_id, deltas)
    (artifact_dir / "summary.json").write_text(json.dumps(summary, indent=2))
    (artifact_dir / "summary.md").write_text(render_markdown(summary))
    return 0 if not summary["failures"] else 2


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
