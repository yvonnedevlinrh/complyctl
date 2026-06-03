# Changelog

## Unreleased

### Breaking Changes

- `--format otel` removed from `complyctl scan`. Export is now triggered
  via the `COMPLYTIME_EXPORT_ENABLED=true` environment variable and works
  alongside any `--format` flag.

### Added

- Complypack pull support: `complyctl get` fetches complypack OCI
  artifacts declared in the `complypacks` section of `complytime.yaml`.
  Cached complypacks are dispatched to providers by evaluator-id during
  generation. `complyctl providers` shows cached complypack versions.
  `complyctl doctor` checks complypack availability.
- `complyctl scan` accepts a positional `[target]` argument to scope
  scans to a single target. When a target has exactly one policy, the
  `--policy-id` flag is inferred automatically.
- Policy resolver supports both split-layer and Gemara bundle-format OCI
  artifacts. `internal/policy/loader.go` gained `LoadBundleFiles()` and
  `DetectManifestShape()`.
- `complyctl scan` exits non-zero on operational errors from providers.
  `ScanResponse.errors` proto field surfaces provider-side errors.
- Devcontainer configuration for interactive CLI testing during PR
  reviews (`.devcontainer/`, `docs/TESTING_ENVIRONMENT.md`,
  `make test-devcontainer`). Supports GitHub Codespaces, DevPod, and
  VS Code Dev Containers.
- Cross-repo integration test infrastructure validating the complyctl + Ampel
  provider pipeline end-to-end (`tests/cross-repo/`, `make test-cross-repo`).
- CI workflow `ci_cross_repo_integration.yml` that builds complyctl from the PR
  branch and complytime-providers from main, then runs the full get → generate →
  scan pipeline with real snappy and ampel binaries.
- Minimal test Gemara policy (`policies/test-branch-protection`) seeded in the
  mock OCI registry for integration testing.
- OPA provider test content in devcontainer: Gemara testdata (catalog +
  policy with `executor.id: opa`), OPA complypack artifact (Rego policies
  + `complytime-mapping.json`), `test-opa-bp` policy-id and
  `test-k8s-deployment` target in workspace configuration. OPA command
  reference added to `docs/TESTING_ENVIRONMENT.md`.

### Fixed

- Scan reports now resolve assessment plan IDs to requirement IDs,
  ensuring output displays meaningful identifiers instead of internal
  plan references. Affects EvaluationLog, OSCAL, SARIF, and Markdown
  output formats.
