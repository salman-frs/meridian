# Docs Authoring

Meridian documentation is built with MkDocs, Material for MkDocs, `mike`, generated Cobra reference pages, and redirect support for published URLs.

## Local workflow

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r docs/requirements.txt
go run ./cmd/meridian-docs
mkdocs serve
```

## Rules

- keep generated CLI pages under `docs/reference/cli/` machine-generated
- update command help text in Go code, then regenerate
- keep hand-authored pages explanatory, not just list-based
- preserve redirects when moving published documentation URLs

## CI expectations

The docs workflow:

- regenerates CLI pages
- fails on generated-doc drift
- builds the site in strict mode
- publishes versioned docs with `mike`
