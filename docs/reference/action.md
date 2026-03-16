# GitHub Action Reference

The composite action lives at `action/action.yml`.

## Inputs

| Input | Required | Default | Notes |
| --- | --- | --- | --- |
| `config` | yes | none | path to the Collector config in the checked-out workspace |
| `engine` | no | `auto` | `auto`, `docker`, `containerd` |
| `mode` | no | `safe` | `safe`, `tee`, `live` |
| `collector_image` | no | pinned contrib image | runtime Collector image |
| `timeout` | no | `30s` | overall runtime timeout |
| `env_file` | no | empty | optional dotenv file |
| `assertions` | no | empty | `v1` assertions or `v2` contracts file |
| `render_graph` | no | `mermaid` | runtime graph artifact mode |
| `comment_mode` | no | `update` | `off`, `create`, `update` |
| `artifact_retention_days` | no | `7` | uploaded artifact retention |

## Behavior

The action:

- builds Meridian from source in the checked-out repository
- runs `meridian ci`
- uploads `meridian-artifacts`, `summary.md`, and `report.json`
- writes the GitHub step summary
- can update one PR comment marked with `<!-- meridian-comment -->`

## Permissions

For PR comment updates, the workflow needs:

- `contents: read`
- `pull-requests: write`
