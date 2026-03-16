# CLI Reference

These pages are generated from the Cobra command tree with `go run ./cmd/meridian-docs`.
Do not hand-edit the generated command pages. Update command help text in Go code, then regenerate.

## Commands

- [`meridian`](meridian.md): Review and runtime-test OpenTelemetry Collector configs
- [`meridian check`](meridian_check.md): Run Meridian's opinionated end-to-end confidence workflow
- [`meridian ci`](meridian_ci.md): CI-friendly compatibility wrapper around check
- [`meridian completion`](meridian_completion.md): Generate shell completion scripts
- [`meridian debug`](meridian_debug.md): Inspect artifacts from a previous Meridian run
- [`meridian debug bundle`](meridian_debug_bundle.md): Print the run bundle manifest
- [`meridian debug capture`](meridian_debug_capture.md): Print capture samples from a run directory
- [`meridian debug logs`](meridian_debug_logs.md): Print collector logs from a run directory
- [`meridian debug summary`](meridian_debug_summary.md): Print the stored markdown summary
- [`meridian diff`](meridian_diff.md): Compare two collector configs and classify risky changes
- [`meridian graph`](meridian_graph.md): Build a pipeline graph for the collector config
- [`meridian test`](meridian_test.md): Run the runtime harness against the collector config
- [`meridian validate`](meridian_validate.md): Run static validation against a collector config
- [`meridian version`](meridian_version.md): Print the Meridian version
