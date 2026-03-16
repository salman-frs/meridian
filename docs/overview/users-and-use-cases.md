# Users and Use Cases

## Summary

Meridian serves multiple engineering roles, but they do not use the product in the same way. Good documentation should make those paths explicit.

## Platform and observability engineers

This is Meridian's primary audience.

Typical use cases:

- editing shared Collector configs in Git
- verifying route, processor, or exporter changes locally
- producing artifacts that other reviewers can inspect
- capturing the difference between expected negative behavior and an actual regression

These users usually care most about `validate`, `check`, `diff`, and artifact quality.

## Reviewers

Reviewers often do not run the CLI themselves. They consume the output from CI or from an artifact bundle attached to a pull request.

They care about:

- clear risk highlights
- graph artifacts
- semantic validation stage results
- runtime evidence and contract results
- summaries that explain failure without requiring reruns

## CI maintainers

CI maintainers care about consistency and scriptability.

They need:

- predictable exit codes
- machine-readable JSON output
- PR-comment and step-summary support
- durable artifact uploads
- enough context to debug failures from the workflow logs and bundle

## Meridian maintainers

Maintainers need deeper confidence in Meridian itself. That is why the repository includes a repo-owned k3s fixture and a dedicated regression workflow outside the normal user path.

## Read next

- [Main Features](main-features.md)
- [Recommended Workflows](../workflows/index.md)
