---
title: meridian check
---

## meridian check

Run Meridian's opinionated end-to-end confidence workflow

```
meridian check [flags]
```

### Options

```
      --assertions string           assertions or contracts YAML file
      --base string                 git base ref used to materialize the old config
      --capture-samples int         maximum captured telemetry samples to persist per signal (default 5)
      --capture-timeout duration    capture wait timeout (default 10s)
      --changed-only                require explicit diff inputs and include only diff-aware review hints
      --collector-image string      collector image to run (default "otel/opentelemetry-collector-contrib@sha256:e7c92c715f28ff142f3bcaccd4fc5603cf4c71276ef09954a38eb4038500a5a5")
      --engine string               container engine: auto|docker|containerd (default "auto")
      --head string                 git head ref used to materialize the new config
  -h, --help                        help for check
      --inject-timeout duration     telemetry injection timeout (default 5s)
      --keep-containers             keep the collector container running after the test
      --mode string                 runtime mode: safe|tee|live (default "safe")
      --new string                  new collector config file used for diff-aware review
      --old string                  old collector config file used for diff-aware review
      --output string               artifact output directory (default "./meridian-artifacts")
      --pipelines strings           limit runtime checks to specific signals or pipelines
      --render-graph string         additional graph artifact for runtime commands: mermaid|svg|none (default "mermaid")
      --seed int                    deterministic generation seed (default 42)
      --severity-threshold string   minimum diff severity threshold: low|medium|high (default "low")
      --startup-timeout duration    collector startup timeout (default 10s)
      --timeout duration            overall runtime timeout (default 30s)
```

### Options inherited from parent commands

```
      --collector-binary string   path to a collector binary used for semantic validation
  -c, --config stringArray        collector config source; repeatable and may be a file path or collector config URI
      --config-dir string         path to a rendered collector config directory
      --env stringArray           inline KEY=VALUE env vars
      --env-file string           dotenv file used for config interpolation
      --format string             output format: human|json (default "human")
      --quiet                     suppress human progress output
      --verbose                   enable verbose output
```

### SEE ALSO

* [meridian](meridian.md)	 - Review and runtime-test OpenTelemetry Collector configs

