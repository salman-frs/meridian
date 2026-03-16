---
title: meridian diff
---

## meridian diff

Compare two collector configs and classify risky changes

```
meridian diff [flags]
```

### Options

```
      --base string                 git base ref used to materialize the old config
      --collector-image string      collector image used when --collector-binary is not provided (default "otel/opentelemetry-collector-contrib@sha256:e7c92c715f28ff142f3bcaccd4fc5603cf4c71276ef09954a38eb4038500a5a5")
      --engine string               container engine used for collector image semantic validation: auto|docker|containerd (default "auto")
      --head string                 git head ref used to materialize the new config
  -h, --help                        help for diff
      --new string                  new collector config file used for diff-aware review
      --old string                  old collector config file used for diff-aware review
      --severity-threshold string   minimum diff severity threshold: low|medium|high (default "low")
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

