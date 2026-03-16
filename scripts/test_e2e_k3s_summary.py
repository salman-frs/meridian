import importlib.util
import pathlib
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).with_name("e2e_k3s_summary.py")
SPEC = importlib.util.spec_from_file_location("e2e_k3s_summary", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(MODULE)


class K3sSummaryTests(unittest.TestCase):
    def test_happy_scenario_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            artifact_dir = pathlib.Path(tmp)
            self._write_payload(artifact_dir / "prom-http-requests.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "prom-checkout.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "prom-heartbeat.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "loki-storefront.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "tempo-storefront.json", {"traces": [{"traceID": "abc"}]})
            self._write_logs(
                artifact_dir,
                storefront=["browse_request", "checkout_request"],
                checkout=["checkout_processed"],
                inventory=["inventory_reserved", "inventory_heartbeat"],
                trafficgen=["trafficgen_started", "trafficgen_completed"],
            )

            summary = MODULE.build_summary(
                artifact_dir,
                "happy",
                "run-123",
                {"spans": 2, "logs": 3, "metrics": 4},
            )

            self.assertEqual(summary["result"], "PASS")
            self.assertEqual(summary["failures"], [])

    def test_backend_unreachable_requires_otel_error_and_absent_backends(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            artifact_dir = pathlib.Path(tmp)
            self._write_logs(
                artifact_dir,
                storefront=["browse_request", "checkout_request", "otel_error"],
                checkout=["checkout_processed"],
                inventory=["inventory_reserved"],
                trafficgen=["trafficgen_completed"],
            )

            summary = MODULE.build_summary(
                artifact_dir,
                "backend-unreachable",
                "run-456",
                {"spans": 0, "logs": 0, "metrics": 0},
            )

            self.assertEqual(summary["result"], "PASS")

    def test_drop_traces_passes_when_spans_are_zero(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            artifact_dir = pathlib.Path(tmp)
            self._write_payload(artifact_dir / "prom-http-requests.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "prom-checkout.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "prom-heartbeat.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "loki-storefront.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "tempo-storefront.json", {"traceID": "unexpected"})
            self._write_logs(
                artifact_dir,
                storefront=["browse_request", "checkout_request"],
                checkout=["checkout_processed"],
                inventory=["inventory_reserved"],
                trafficgen=["trafficgen_completed"],
            )

            summary = MODULE.build_summary(
                artifact_dir,
                "drop-traces",
                "run-789",
                {"spans": 0, "logs": 1, "metrics": 1},
            )

            self.assertEqual(summary["result"], "PASS")

    def test_misroute_logs_passes_with_logs_absent(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            artifact_dir = pathlib.Path(tmp)
            self._write_payload(artifact_dir / "prom-http-requests.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "prom-checkout.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "prom-heartbeat.json", {"data": {"result": [1]}})
            self._write_payload(artifact_dir / "tempo-storefront.json", {"traces": [{"traceID": "abc"}]})
            (artifact_dir / "logs").mkdir(parents=True, exist_ok=True)

            summary = MODULE.build_summary(
                artifact_dir,
                "misroute-logs",
                "run-321",
                {"spans": 3, "logs": 0, "metrics": 2},
            )

            self.assertEqual(summary["result"], "PASS")

    def _write_payload(self, path: pathlib.Path, payload: dict) -> None:
        path.write_text(MODULE.json.dumps(payload))

    def _write_logs(self, artifact_dir: pathlib.Path, **services: list[str]) -> None:
        logs_dir = artifact_dir / "logs"
        logs_dir.mkdir(parents=True, exist_ok=True)
        for service, events in services.items():
            payloads = [
                MODULE.json.dumps({"event": event, "service": service, "run_id": "run-123"})
                for event in events
            ]
            (logs_dir / f"{service}.log").write_text("\n".join(payloads) + "\n")


if __name__ == "__main__":
    unittest.main()
