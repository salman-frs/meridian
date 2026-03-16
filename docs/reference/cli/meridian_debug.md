---
title: meridian debug
---

## meridian debug

Inspect artifacts from a previous Meridian run

### Options

```
  -h, --help            help for debug
      --output string   artifact output directory (default "./meridian-artifacts")
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
* [meridian debug bundle](meridian_debug_bundle.md)	 - Print the run bundle manifest
* [meridian debug capture](meridian_debug_capture.md)	 - Print capture samples from a run directory
* [meridian debug logs](meridian_debug_logs.md)	 - Print collector logs from a run directory
* [meridian debug summary](meridian_debug_summary.md)	 - Print the stored markdown summary

