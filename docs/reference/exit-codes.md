# Exit Codes

Meridian keeps exit semantics scriptable and simple.

## Stable codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Unclassified failure |
| `2` | User input, validation, or contract/assertion loading failure |
| `3` | Runtime, semantic execution, artifact, or environment failure |

## Guidance

Treat exit codes as coarse failure classes. For exact cause, inspect:

- stderr
- `summary.md`
- `report.json`
- bundle artifacts such as `collector.log`
