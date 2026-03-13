# Troubleshooting

## Container engine missing

Meridian runtime commands need a supported container engine. If `meridian check` or `meridian test` fails before container startup:

- confirm `docker version` works when using Docker
- confirm `nerdctl version` works on Linux when using `--engine containerd`
- confirm `lima nerdctl version` works on macOS when using `--engine containerd`
- confirm the selected daemon/runtime is running
- try `--engine docker` or `--engine containerd` explicitly instead of `auto`
- rerun with `--verbose`

On macOS, `--engine containerd` uses Lima rather than trying to talk to an OrbStack Docker socket directly.

## Missing env vars

Validation failures for missing env vars include the variable name and a remediation hint. Provide the value with:

- `--env-file path/to/.env`
- `--env KEY=value`
- exported shell environment variables

## Runtime failures

Open the artifact directory and inspect:

- `collector.log`
- `config.patched.yaml`
- `graph.mmd`
- `diff.md` when diff-aware review is enabled
- `captures/*.json`

For deeper debugging:

```bash
meridian test -c collector.yaml --keep-containers --verbose
```
