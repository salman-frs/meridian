# Docs Authoring

Meridian documentation is built with MkDocs, Material for MkDocs, and `mike`.

## Local workflow

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r docs/requirements.txt
go run ./cmd/meridian-docs
mkdocs serve
```

## Rules

- write hand-authored content under the docs IA sections
- do not hand-edit files under `docs/reference/cli/` except `index.md`
- update Go command help text, then regenerate CLI docs
- keep examples short and copy-pasteable
- prefer task-focused docs over long narrative dumps

## CI expectations

The docs workflow:

- regenerates CLI docs
- fails if generated files drift
- runs `mkdocs build --strict`
