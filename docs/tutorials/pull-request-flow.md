# Pull Request Workflow

This is the intended reviewer workflow for a repo that stores Collector configs in Git.

## 1. Validate locally before opening the PR

```bash
./bin/meridian validate -c otel/collector.yaml
./bin/meridian check -c otel/collector.yaml
```

## 2. Review the structural delta

```bash
./bin/meridian diff --base origin/main --head HEAD --new otel/collector.yaml
```

## 3. Enable the action in CI

```yaml
jobs:
  meridian:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
    steps:
      - uses: actions/checkout@v4
      - uses: salman-frs/meridian/action@v1
        with:
          config: otel/collector.yaml
```

## 4. Review the CI evidence

Use:

- the GitHub step summary
- the PR comment
- the uploaded artifact bundle

The point is durable review evidence, not just pass/fail status.
