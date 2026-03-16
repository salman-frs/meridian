# First Local Run

## 1. Build Meridian

```bash
go build -o ./bin/meridian ./cmd/meridian
```

## 2. Validate a known-good example

```bash
./bin/meridian validate -c examples/basic/collector.yaml
```

If you need env interpolation, add:

```bash
./bin/meridian validate -c collector.yaml --env-file .env --env OTLP_ENDPOINT=localhost:4317
```

## 3. Run the end-to-end check

```bash
./bin/meridian check -c examples/basic/collector.yaml
```

## 4. Inspect the result

```bash
./bin/meridian debug summary
./bin/meridian debug bundle
```

## 5. Save machine-readable output when scripting

```bash
./bin/meridian ci -c examples/basic/collector.yaml --format json --json-file /tmp/report.json
```
