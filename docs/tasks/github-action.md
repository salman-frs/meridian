# Use The GitHub Action

Meridian ships a composite action at [`action/action.yml`](https://github.com/salman-frs/meridian/blob/main/action/action.yml).

## Minimal usage

```yaml
- uses: salman-frs/meridian/action@v1
  with:
    config: examples/basic/collector.yaml
    engine: auto
```

For local development inside this repository, use `uses: ./action`.

## Inputs

- `config`
- `engine`
- `mode`
- `collector_image`
- `timeout`
- `env_file`
- `assertions`
- `render_graph`
- `comment_mode`
- `artifact_retention_days`

## Behavior

The action:

- builds the Meridian binary from the checked-out repository
- runs `meridian ci`
- uploads `meridian-artifacts`, `summary.md`, and `report.json`
- writes the GitHub step summary
- updates one PR comment marked with `<!-- meridian-comment -->` when comment mode is enabled

## Caveats

- `engine: containerd` is intended for Linux runners with `nerdctl`
- the shipped action depends on `actions/upload-artifact@v4`, so GHES is not part of the current compatibility contract
