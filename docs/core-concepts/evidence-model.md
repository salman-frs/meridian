# Evidence Model

## Summary

Meridian is an evidence-producing tool. The right way to evaluate it is not “did the command run” but “what evidence did the run produce, and what does that evidence justify?”

## Why this exists

Collector review often collapses into intuition because the available signals are weak:

- YAML looks plausible
- the config validates somewhere else
- a reviewer assumes runtime behavior stayed intact

Meridian replaces that with layered evidence.

## The three evidence layers

### Repo-side evidence

Repo-side loading and validation operate on locally materializable inputs. This layer can catch structural problems, environment interpolation issues, and graph-visible topology shape.

### Collector-native evidence

Semantic validation asks a real Collector target to list components, validate the config, and best-effort render effective config with `print-config`.

### Runtime evidence

Runtime execution uses a patched harness, injects synthetic telemetry, captures outputs, and evaluates assertions or contracts against those captures.

## What Meridian proves and does not prove

Meridian proves that the selected inputs and target produced the observed evidence under the selected mode and engine. It does not prove production correctness beyond that scope.

The trust boundary is especially important in `safe` mode:

- strong evidence for flow and reviewability
- intentionally limited evidence for live backend behavior

## Read next

- [Semantic Validation](semantic-validation.md)
- [Runtime Modes and Patching](runtime-modes-and-patching.md)
