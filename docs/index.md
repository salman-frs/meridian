# Meridian Docs

Meridian is a local-first CLI and GitHub Action for reviewing OpenTelemetry Collector configuration changes before merge.

It combines:

- repo-side validation
- Collector-native semantic validation
- graph generation
- diff-aware review hints
- deterministic runtime checks
- assertion and contract evaluation
- durable artifacts for local debugging and pull-request review

## Start here

- New user: [Quickstart](overview/quickstart.md)
- Need install details: [Install](overview/install.md)
- Want the exact command surface: [CLI Reference](reference/cli/index.md)
- Integrating CI: [GitHub Action task guide](tasks/github-action.md)
- Maintaining the project: [Contributor Guide](contribute/index.md)

## Product model

Meridian is built around three evidence layers:

1. Repo-side parsing and structure checks on locally materializable config sources.
2. Collector-native semantic validation against the selected binary or image.
3. Deterministic runtime execution against a patched config with capture-backed assertions.

The default path is `safe` mode. It proves flow and reviewability without sending synthetic telemetry to real backends.

## Documentation structure

This site follows an OSS documentation split similar to large infrastructure projects:

- `Overview`: what Meridian is and how to get started
- `Concepts`: product behavior and mental models
- `Tasks`: focused how-to guides
- `Tutorials`: end-to-end guided flows
- `Reference`: exact contracts, flags, and artifacts
- `Contribute`: maintainer and documentation workflows
