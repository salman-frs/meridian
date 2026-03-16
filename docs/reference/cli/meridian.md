---
title: meridian
---

## meridian

Review and runtime-test OpenTelemetry Collector configs

### Options

```
      --collector-binary string   path to a collector binary used for semantic validation
  -c, --config stringArray        collector config source; repeatable and may be a file path or collector config URI
      --config-dir string         path to a rendered collector config directory
      --env stringArray           inline KEY=VALUE env vars
      --env-file string           dotenv file used for config interpolation
      --format string             output format: human|json (default "human")
  -h, --help                      help for meridian
      --quiet                     suppress human progress output
      --verbose                   enable verbose output
```

### SEE ALSO

* [meridian check](meridian_check.md)	 - Run Meridian's opinionated end-to-end confidence workflow
* [meridian ci](meridian_ci.md)	 - CI-friendly compatibility wrapper around check
* [meridian completion](meridian_completion.md)	 - Generate shell completion scripts
* [meridian debug](meridian_debug.md)	 - Inspect artifacts from a previous Meridian run
* [meridian diff](meridian_diff.md)	 - Compare two collector configs and classify risky changes
* [meridian graph](meridian_graph.md)	 - Build a pipeline graph for the collector config
* [meridian test](meridian_test.md)	 - Run the runtime harness against the collector config
* [meridian validate](meridian_validate.md)	 - Run static validation against a collector config
* [meridian version](meridian_version.md)	 - Print the Meridian version

