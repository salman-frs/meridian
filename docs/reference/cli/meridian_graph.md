---
title: meridian graph
---

## meridian graph

Build a pipeline graph for the collector config

```
meridian graph [flags]
```

### Options

```
  -h, --help            help for graph
      --out string      write graph output to a file
      --render string   render mode: mermaid|dot|svg|none (default "mermaid")
      --view string     terminal view: table
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

