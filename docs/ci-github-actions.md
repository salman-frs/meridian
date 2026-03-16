# GitHub Actions

Meridian ships a composite action in [action/action.yml](../action/action.yml).

## Minimal configuration

```yaml
- uses: salman-frs/meridian/action@v1
  with:
    config: examples/basic/collector.yaml
    engine: auto
```

For local development inside this repository, use `uses: ./action`.

## Supported inputs

- `config`
- `engine`
- `mode`
- `collector_image`
- `timeout`
- `env_file`
- `assertions` (`v1` assertions or `v2` contracts/fixtures file)
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

Use `engine: containerd` only on Linux runners with `nerdctl` available. `engine: auto` keeps Docker as the preferred runtime when both engines are present, including macOS hosts that also have Lima installed.

## Notes

- The action uses `actions/upload-artifact@v4`, which is not supported on GHES.
- `meridian ci --format json` now keeps stdout machine-readable; annotations are emitted separately so JSON parsers do not have to strip workflow commands first.
