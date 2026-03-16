# Meridian Action

This composite action builds Meridian from the checked-out repository, runs `meridian ci`, uploads artifacts, and can update a PR comment marked with `<!-- meridian-comment -->`.

Use `uses: ./action` for local development inside this repository. External repositories should use `salman-frs/meridian/action@v1`.

Reference documentation now lives in the docs site:

- [GitHub Action task guide](https://salman-frs.github.io/meridian/tasks/github-action/)
- [GitHub Action reference](https://salman-frs.github.io/meridian/reference/action/)
