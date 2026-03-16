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

GATEWAY_SIGNAL = {
    "spans": "traces",
    "logs": "logs",
    "metrics": "metrics",
}

SECTION_SIGNAL = {
    "prometheus": "metrics",
    "loki": "logs",
    "tempo": "traces",
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


def build_contracts(scenario: str, observed: dict[str, Any]) -> list[dict[str, Any]]:
    expected = SCENARIO_CONTRACTS[scenario]
    contracts: list[dict[str, Any]] = []
    fixture = f"k3s/{scenario}"

    for key, enabled in expected.get("gateway_positive", {}).items():
        actual = observed["gateway_deltas"].get(key, 0)
        contracts.append(
            contract_result(
                contract_id=f"gateway-{key}-positive",
                signal=GATEWAY_SIGNAL[key],
                fixture=fixture,
                passed=(actual > 0) if enabled else (actual <= 0),
                message=f"gateway {key} delta should be positive",
                observed=f"delta={actual}",
                expected_text="delta > 0" if enabled else "delta <= 0",
                diff=[
                    f"expected gateway delta for {key} > 0"
                    if enabled
                    else f"expected gateway delta for {key} to stay <= 0"
                ],
                likely_causes=[
                    "the scenario did not route the expected signal through the gateway",
                    "collector routing or exporter behavior changed",
                ],
                next_steps=[
                    "inspect summary.json gateway_deltas and the scenario overlay",
                    "inspect gateway metrics and scenario pod logs",
                ],
            )
        )

    for key, enabled in expected.get("gateway_zero", {}).items():
        actual = observed["gateway_deltas"].get(key, 0)
        contracts.append(
            contract_result(
                contract_id=f"gateway-{key}-zero",
                signal=GATEWAY_SIGNAL[key],
                fixture=fixture,
                passed=(actual == 0) if enabled else (actual != 0),
                message=f"gateway {key} delta should remain zero",
                observed=f"delta={actual}",
                expected_text="delta == 0" if enabled else "delta != 0",
                diff=[f"expected gateway delta for {key} == 0, got {actual}"],
                likely_causes=[
                    "the negative scenario no longer suppresses that signal",
                    "gateway counters include unexpected traffic for this run",
                ],
                next_steps=[
                    "inspect the scenario overlay for the intended fault injection",
                    "check service logs and gateway counters for the affected signal",
                ],
            )
        )

    for section in ("prometheus", "loki", "tempo"):
        for key, value in expected.get(section, {}).items():
            actual = observed[section].get(key)
            contracts.append(
                contract_result(
                    contract_id=f"{section}-{key.replace('_', '-')}",
                    signal=SECTION_SIGNAL[section],
                    fixture=fixture,
                    passed=(actual == value),
                    message=f"{section} evidence should match the scenario contract",
                    observed=f"{section}.{key}={actual}",
                    expected_text=f"{section}.{key}={value}",
                    diff=[f"expected {section}.{key} == {value}, got {actual}"],
                    likely_causes=[
                        "backend evidence no longer matches the intended scenario behavior",
                        "run-scoped evidence is missing or contaminated by unrelated telemetry",
                    ],
                    next_steps=[
                        f"inspect {section} artifacts under the scenario bundle",
                        "check whether the scenario still isolates the expected backend behavior",
                    ],
                )
            )

    for service, events in expected.get("app_events", {}).items():
        actual_events = set(observed["app_logs"].get(service, {}).get("events", []))
        for event in events:
            contracts.append(
                contract_result(
                    contract_id=f"app-event-{service}-{event.replace('_', '-')}",
                    signal="logs",
                    fixture=fixture,
                    passed=(event in actual_events),
                    message=f"{service} should emit event {event}",
                    observed=f"{service} events={sorted(actual_events)}",
                    expected_text=f"contains {event}",
                    diff=[f"expected {service} log event {event}"],
                    likely_causes=[
                        "application behavior changed for the scenario",
                        "logs are missing or the event name changed",
                    ],
                    next_steps=[
                        f"inspect logs/{service}.log in the scenario bundle",
                        "confirm the scenario still drives the expected application path",
                    ],
                )
            )

    for service, events in expected.get("app_events_absent", {}).items():
        actual_events = set(observed["app_logs"].get(service, {}).get("events", []))
        for event in events:
            contracts.append(
                contract_result(
                    contract_id=f"app-event-absent-{service}-{event.replace('_', '-')}",
                    signal="logs",
                    fixture=fixture,
                    passed=(event not in actual_events),
                    message=f"{service} should not emit event {event}",
                    observed=f"{service} events={sorted(actual_events)}",
                    expected_text=f"excludes {event}",
                    diff=[f"expected {service} log event {event} to be absent"],
                    likely_causes=[
                        "the negative scenario no longer blocks the downstream success path",
                        "logs are coming from an unexpected code path",
                    ],
                    next_steps=[
                        f"inspect logs/{service}.log in the scenario bundle",
                        "check auth and downstream dependency settings for the scenario",
                    ],
                )
            )

    for service, value in expected.get("app_logs_present", {}).items():
        actual = observed["app_logs"].get(service, {}).get("text_present")
        contracts.append(
            contract_result(
                contract_id=f"app-log-presence-{service}",
                signal="logs",
                fixture=fixture,
                passed=(actual == value),
                message=f"{service} log presence should match the scenario contract",
                observed=f"{service}.text_present={actual}",
                expected_text=f"{service}.text_present={value}",
                diff=[f"expected {service} log presence == {value}, got {actual}"],
                likely_causes=[
                    "the scenario no longer suppresses or emits pod logs as expected",
                    "log collection on the VM changed",
                ],
                next_steps=[
                    f"inspect logs/{service}.log in the scenario bundle",
                    "verify the scenario's log routing or suppression settings",
                ],
            )
        )

    combined = observed["combined_log_text"]
    for event in expected.get("error_events_present", []):
        contracts.append(
            contract_result(
                contract_id=f"combined-error-present-{event.replace('_', '-')}",
                signal="logs",
                fixture=fixture,
                passed=(event in combined),
                message=f"combined logs should contain {event}",
                observed=f"contains={event in combined}",
                expected_text="contains=True",
                diff=[f"expected combined logs to contain {event}"],
                likely_causes=[
                    "the scenario no longer triggers the intended error condition",
                    "error logs were not collected into the scenario bundle",
                ],
                next_steps=[
                    "inspect combined service logs in the scenario bundle",
                    "verify the failing backend or auth condition still exists",
                ],
            )
        )

    for event in expected.get("error_events_absent", []):
        contracts.append(
            contract_result(
                contract_id=f"combined-error-absent-{event.replace('_', '-')}",
                signal="logs",
                fixture=fixture,
                passed=(event not in combined),
                message=f"combined logs should exclude {event}",
                observed=f"contains={event in combined}",
                expected_text="contains=False",
                diff=[f"expected combined logs to exclude {event}"],
                likely_causes=[
                    "an unexpected error path is now being exercised",
                    "the scenario is leaking unrelated failures into the run",
                ],
                next_steps=[
                    "inspect combined service logs in the scenario bundle",
                    "check whether the scenario overlay changed collector or app behavior",
                ],
            )
        )

    return contracts


def contract_result(
    contract_id: str,
    signal: str,
    fixture: str,
    passed: bool,
    message: str,
    observed: str,
    expected_text: str,
    diff: list[str],
    likely_causes: list[str],
    next_steps: list[str],
) -> dict[str, Any]:
    return {
        "id": contract_id,
        "severity": "fail",
        "signal": signal,
        "fixture": fixture,
        "status": "PASS" if passed else "FAIL",
        "message": message if passed else "contract failed",
        "observed": observed,
        "expected": expected_text,
        "diff": [] if passed else diff,
        "likely_causes": [] if passed else likely_causes,
        "next_steps": [] if passed else next_steps,
    }


def build_summary(artifact_dir: pathlib.Path, scenario: str, run_id: str, deltas: dict[str, float]) -> dict[str, Any]:
    observed = build_observed(artifact_dir, deltas)
    contracts = build_contracts(scenario, observed)
    failed_contracts = [item for item in contracts if item["status"] == "FAIL"]
    failures = [item["diff"][0] for item in failed_contracts if item["diff"]]
    return {
        "run_id": run_id,
        "scenario": scenario,
        "result": "PASS" if not failed_contracts else "FAIL",
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
        "contracts": contracts,
        "contract_summary": {
            "total": len(contracts),
            "pass": len(contracts) - len(failed_contracts),
            "fail": len(failed_contracts),
        },
    }


def render_markdown(summary: dict[str, Any]) -> str:
    observed = summary["observed"]
    lines = [
        "# Meridian K3s E2E Summary",
        "",
        f"- run_id: `{summary['run_id']}`",
        f"- scenario: `{summary['scenario']}`",
        f"- result: `{summary['result']}`",
        f"- contract summary: `{summary['contract_summary']['pass']}/{summary['contract_summary']['total']} passed`",
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

    lines.extend(["", "## Contract checks"])
    for contract in summary["contracts"]:
        lines.append(f"- `{contract['id']}`: {contract['status']} ({contract['signal']})")
        for item in contract["diff"]:
            lines.append(f"  - {item}")

    failing = next((item for item in summary["contracts"] if item["status"] == "FAIL"), None)
    if failing is not None:
        lines.extend(["", "## Top contract failure"])
        lines.append(f"- `{failing['id']}`: {failing['message']}")
        for item in failing["diff"]:
            lines.append(f"- Diff: {item}")
        for item in failing["likely_causes"]:
            lines.append(f"- Likely cause: {item}")
        for item in failing["next_steps"]:
            lines.append(f"- Next step: {item}")
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
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
