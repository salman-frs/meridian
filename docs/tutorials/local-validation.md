# Local End-To-End Walkthrough

This walkthrough uses the repo's basic example to exercise Meridian's normal local flow.

## 1. Build the CLI

```bash
go build -o ./bin/meridian ./cmd/meridian
```

## 2. Run validation

```bash
./bin/meridian validate -c examples/basic/collector.yaml
```

Observe:

- local-load stage result
- semantic stage result
- any findings grouped by severity

## 3. Run the confidence workflow

```bash
./bin/meridian check -c examples/basic/collector.yaml
```

## 4. Read the evidence

Open:

- `summary.md`
- `report.json`
- `config.patched.yaml`
- `graph.mmd`
- `captures/`

If the selected Collector supports `print-config`, also inspect `config.final.yaml`.
