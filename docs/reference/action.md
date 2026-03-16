# GitHub Action Reference

The composite action lives at [`action/action.yml`](https://github.com/salman-frs/meridian/blob/main/action/action.yml).

## Inputs

| Input | Required | Default | Notes |
| --- | --- | --- | --- |
| `config` | yes | none | Collector config path in the checked-out workspace |
| `engine` | no | `auto` | `auto`, `docker`, `containerd` |
| `mode` | no | `safe` | `safe`, `tee`, `live` |
| `collector_image` | no | pinned contrib image | Used for runtime execution |
| `timeout` | no | `30s` | Overall runtime timeout |
| `env_file` | no | empty | Optional dotenv file |
| `assertions` | no | empty | `v1` assertions or `v2` contracts file |
| `render_graph` | no | `mermaid` | Runtime graph artifact mode |
| `comment_mode` | no | `update` | `off`, `create`, `update` |
| `artifact_retention_days` | no | `7` | Passed to artifact upload |

## Outputs and side effects

The action does not publish custom action outputs. Its side effects are:

- uploaded artifacts
- a GitHub step summary
- an optional PR comment update

## Permissions

For PR comment updates, the calling workflow needs `pull-requests: write`.
