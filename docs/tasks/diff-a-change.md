# Review A Config Diff

## Compare two files directly

```bash
./bin/meridian diff --old old.yaml --new new.yaml
```

## Compare git refs

```bash
./bin/meridian diff --base origin/main --head HEAD --new otel/collector.yaml
```

## Gate runtime review to changed inputs

```bash
./bin/meridian check -c otel/collector.yaml --old old.yaml --new new.yaml --changed-only
```

`--changed-only` requires explicit diff inputs.

## Filter by severity

```bash
./bin/meridian diff --old old.yaml --new new.yaml --severity-threshold medium
```

Use this when you want reviewer attention focused on medium and high-risk changes only.
