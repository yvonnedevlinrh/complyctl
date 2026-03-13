# Contract: Reusable CRAP Load Analysis Workflow

## Interface Type

GitHub Actions reusable workflow (`workflow_call`)

## Caller Contract

### Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `go-version-file` | string | no | `'./go.mod'` | Path to go.mod for Go version detection |
| `gaze-version` | string | no | `'latest'` | Gaze version tag to install via `go install` |
| `baseline-file` | string | no | `'.gaze/baseline.json'` | Path to committed baseline thresholds file |
| `packages` | string | no | `'./...'` | Go packages to analyse |
| `coverprofile` | string | no | `'coverage.out'` | Path to coverage profile |
| `new-function-threshold` | number | no | `30` | CRAP score ceiling for new functions with no baseline entry |
| `post-comment` | boolean | no | `true` | Whether to post/update a PR comment with results |

> **Note:** Per-function CRAP/GazeCRAP thresholds (both default to 15) are
> managed by gaze internally. The workflow uses `gaze report --format=json`
> which runs CRAP, Quality, Classification, and Docscan analysis in a single
> invocation.

### Required Permissions

| Permission | Level | Purpose |
|------------|-------|---------|
| `contents` | `read` | Checkout code and read baseline file |
| `pull-requests` | `write` | Post/update PR comments (when `post-comment` is true) |

### Outputs

| Output | Type | Description |
|--------|------|-------------|
| `status` | string | `pass` or `fail` |
| `crapload-count` | number | Number of functions at or above CRAP threshold |
| `gaze-crapload-count` | number | Number of functions at or above GazeCRAP threshold |
| `regressions-count` | number | Number of functions that regressed vs baseline |
| `improvements-count` | number | Number of functions that improved vs baseline |

## Behaviour Contract

### Preconditions

- Calling repository MUST be a Go project with a valid `go.mod`
- Go test suite MUST be runnable via `go test ./...`

### Postconditions

- If `post-comment` is true and the trigger is a pull request, exactly one comment is posted or updated (identified by a marker string)
- Exit code is 0 if all thresholds are met, non-zero otherwise
- Coverage profile is generated if not already present

### Invariants

- The workflow MUST NOT modify the repository contents (read-only analysis)
- The workflow MUST NOT push commits or create branches
- Existing PR comments from previous runs MUST be updated, not duplicated

## Usage Example

```yaml
jobs:
  crapload:
    uses: complytime/complyctl/.github/workflows/reusable_crapload_analysis.yml@main
    permissions:
      contents: read
      pull-requests: write
    with:
      new-function-threshold: 25
```
