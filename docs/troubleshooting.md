# Troubleshooting

## Docker missing

Meridian runtime commands need Docker. If `meridian check` or `meridian test` fails before container startup:

- confirm `docker version` works
- confirm the daemon is running
- rerun with `--verbose`

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
