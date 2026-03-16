---
title: meridian debug logs
---

## meridian debug logs

Print collector logs from a run directory

```
meridian debug logs [flags]
```

### Options

```
  -h, --help         help for logs
      --run string   run directory
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

* [meridian debug](meridian_debug.md)	 - Inspect artifacts from a previous Meridian run

