# Inspect Artifacts

Meridian's debug commands operate on the latest run by default.

## Summary

```bash
./bin/meridian debug summary
```

## Bundle manifest

Human form:

```bash
./bin/meridian debug bundle
```

JSON form:

```bash
./bin/meridian debug bundle --format json
```

## Collector logs

```bash
./bin/meridian debug logs
```

## Capture samples

```bash
./bin/meridian debug capture
```

## Target a specific run

```bash
./bin/meridian debug summary --output ./meridian-artifacts --run ./meridian-artifacts/runs/20260316-123456.000000000-4242
```
