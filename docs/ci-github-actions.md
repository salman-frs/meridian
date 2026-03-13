# GitHub Actions

Meridian ships a composite action in [action/action.yml](../action/action.yml).

## Minimal configuration

```yaml
- uses: salman-frs/meridian/action@v1
  with:
    config: examples/basic/collector.yaml
```

For local development inside this repository, use `uses: ./action`.

## Supported inputs

- `config`
- `mode`
- `collector_image`
- `timeout`
- `env_file`
- `assertions`
- `render_graph`
- `comment_mode`
- `artifact_retention_days`

## Behavior

- builds the Meridian binary
- runs `meridian ci`
- writes `report.json`, `summary.md`, and optional `graph.svg`
- uploads the artifact bundle
- writes the GitHub step summary
- updates one PR comment with the `<!-- meridian-comment -->` marker when enabled
