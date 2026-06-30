## Why

`complyctl get` re-syncs complypack artifacts via atomic directory replacement (`RemoveAll` + `Rename`), destroying any provider-extracted content inside `~/.complytime/complypacks/<evaluator>/<version>/`. The workspace generation cache (`.complytime/generation/`) and provider artifact directories (`.complytime/<evaluator>/`) are not invalidated. The next `complyctl scan` sees matching digests and existing artifact directories, prints "Reusing generated artifacts...", and the provider reads a dead path from stale configuration files.

This is tracked in [issue #551](https://github.com/complytime/complyctl/issues/551). The workaround requires manual deletion of generation state and evaluator artifacts before each scan after a re-fetch.

## What Changes

- When `complyctl get` fetches a complypack (`fetched == true`), it eagerly invalidates the workspace generation state for the affected evaluator-id and removes stale workspace artifacts for that evaluator.
- A new `InvalidateForEvaluator` helper scans generation state files and removes those referencing the updated evaluator-id.
- A new `RemoveEvaluatorArtifacts` helper removes the workspace evaluator artifact directory.
- The `syncComplypacks` call chain gains access to the workspace `baseDir` to perform invalidation.

## Capabilities

### New Capabilities
- `generation-invalidation`: Eager invalidation of workspace generation state and evaluator artifacts when a complypack is re-fetched, ensuring the next scan triggers a fresh Generate cycle.

### Modified Capabilities
- `get`: When a complypack fetch occurs, the command now invalidates workspace generation cache for the affected evaluator-id and emits a message to stderr.

### Removed Capabilities
- None. This is purely additive behavior on an existing code path.

## Impact

- `cmd/complyctl/cli/get.go` — `syncComplypacks`, `syncAllComplypacks`, `syncSingleComplypack` gain `baseDir` parameter; invalidation logic added when `fetched == true`
- `internal/policy/generation_state.go` — new `InvalidateForEvaluator()` and `RemoveEvaluatorArtifacts()` functions
- `internal/policy/generation_state_test.go` — new unit tests for invalidation helpers
- `cmd/complyctl/cli/cli_test.go` — integration test for the full invalidation flow

## Constitution Alignment

### I. Autonomous Collaboration

**Assessment**: PASS

The fix produces self-describing behavior: stderr messages inform the user when invalidation occurs. No manual intervention is needed after `get` to achieve correct scan behavior.

### II. Composability First

**Assessment**: PASS

Invalidation is scoped per-evaluator-id. Unrelated evaluators and their generation state are untouched. The helpers are standalone functions usable by other callers if needed.

### III. Observable Quality

**Assessment**: PASS

Invalidation emits a stderr message indicating which evaluator's cache was cleared. Generation state files are machine-parseable JSON. The behavior is deterministic and testable.

### IV. Testability

**Assessment**: PASS

Each helper (`InvalidateForEvaluator`, `RemoveEvaluatorArtifacts`) is independently testable with filesystem isolation via `t.TempDir()`. The integration flow is testable by mocking the sync return value.
