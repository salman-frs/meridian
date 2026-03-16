# Exit Codes

Meridian keeps exit behavior intentionally simple.

| Code | Meaning |
| --- | --- |
| `0` | success |
| `1` | unclassified failure |
| `2` | user input, validation, or assertion/contract loading failure |
| `3` | runtime, semantic execution, artifact, or environment failure |

Use exit codes as broad classes, then inspect stderr, `summary.md`, `report.json`, and the artifact bundle for exact cause.
