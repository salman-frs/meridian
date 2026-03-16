# Graph

## Summary

`meridian graph` turns the Collector config into a graph model so engineers can review pipeline wiring directly rather than infer it from nested YAML.

## Why it exists

Topology changes are hard to review in raw configuration. Reviewers need to see:

- which receivers feed which pipelines
- which processors sit in the path
- where exporters and connectors sit
- whether the shape of the configuration changed materially

## How it works

Meridian builds a graph model from the normalized config representation and renders:

- Mermaid by default
- DOT or SVG when requested
- a table view in terminal output when `--view table` is used

Graph artifacts are also generated during runtime-oriented commands when requested.

## What it proves

Graph output proves how Meridian interpreted the config model. It is a review aid, not a runtime guarantee.

## Inputs and outputs

Important flags:

- `--render mermaid|dot|svg|none`
- `--view table`
- `--out`

Typical outputs:

- stdout graph view
- `graph.mmd`
- optional `graph.svg`

## Related pages

- [Diff](diff.md)
- [Artifact Model](../core-concepts/artifact-model.md)
