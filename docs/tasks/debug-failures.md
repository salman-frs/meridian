# Debug Failures

## Runtime startup failures

If the Collector does not become ready:

- inspect `collector.log`
- inspect `config.patched.yaml`
- rerun with `--keep-containers --verbose`

```bash
./bin/meridian test -c collector.yaml --keep-containers --verbose
```

## Missing env values

Provide values through:

- `--env-file`
- repeatable `--env KEY=value`
- exported shell environment variables

## Semantic validation gaps

If `print-config` is unavailable:

- confirm the Collector build supports `print-config`
- confirm it accepts `otelcol.printInitialConfig`
- expect effective-config artifacts to be absent

## Engine issues

Validate the runtime first:

- `docker version`
- `nerdctl version`
- `lima nerdctl version`

If `auto` picks the wrong engine for your environment, select `--engine docker` or `--engine containerd` explicitly.
