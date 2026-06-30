## Context

`complyctl get` calls `ComplypackCache.Store()` which does `os.RemoveAll(finalDir)` followed by `os.Rename(tmpDir, finalDir)`. This atomic replacement destroys any provider-extracted `content/` subdirectory that lived inside the version directory. The workspace generation cache (`.complytime/generation/*.json`) and evaluator artifact directories (`.complytime/{evaluator}/`) are not touched by `get`, causing the next `complyctl scan` to skip Generate and read stale paths.

Issue #583 (closed) added complypack digest tracking to `GenerationState.IsFresh()` as a lazy detection mechanism. However, this fails when:
1. The remote digest is unchanged (same manifest, re-pushed content)
2. Provider artifacts reference absolute paths into the replaced cache directory

Eager invalidation at fetch time eliminates both failure modes.

## Goals / Non-Goals

### Goals
- When a complypack is re-fetched, eagerly invalidate workspace generation state for the affected evaluator-id
- Remove stale evaluator artifact directories that may contain dead path references
- Provide clear stderr feedback when invalidation occurs
- Maintain backward compatibility (workspaces without complypacks are unaffected)

### Non-Goals
- Evicting old complypack versions from cache (follow-up: #646)
- Fixing `LookupByEvaluatorID` non-determinism (follow-up: #645)
- Validating duplicate evaluator-ids across repos (follow-up: #647)
- Adding `--force` flag to `get` (follow-up: #649)
- Invalidating on policy re-sync (lower severity, policies don't do destructive replace)

## Decisions

### D1: Eager invalidation in `get`, not lazy detection in `scan`

**Decision**: When `SyncComplypack` returns `fetched == true`, immediately delete generation state files and evaluator artifacts in the workspace.

**Rationale**: Lazy detection via `IsFresh()` already exists but has failure modes (unchanged digest, stale paths in provider artifacts). Eager invalidation is a belt-and-suspenders approach that guarantees correctness. The cost is minimal — deleting a few JSON files and a directory.

**Alternatives considered**: Strengthen `evaluatorArtifactsExist()` to check content validity (e.g., verify referenced paths in scan-config.json). Rejected — too coupled to provider internals; complyctl should not parse provider-specific config formats.

### D2: Thread `baseDir` through sync functions

**Decision**: Add a `baseDir string` parameter to `syncComplypacks()`, `syncAllComplypacks()`, and `syncSingleComplypack()`. The value comes from `o.ResolveWorkspace()` which is already called in `run()`.

**Rationale**: The workspace path is needed to locate `.complytime/generation/` and `.complytime/{evaluator}/`. Passing it explicitly keeps the dependency visible. Using a struct field on `getOptions` was considered but rejected to avoid widening the struct's surface for a single use case.

### D3: Resolve evaluator-id from state after sync

**Decision**: After `SyncComplypack` returns `fetched == true`, read `state.Complypacks[repository].EvaluatorID` to get the evaluator-id for invalidation.

**Rationale**: The evaluator-id is only known after unpacking the complypack artifact. `SyncComplypack` already calls `state.UpdateComplypackState(repository, version, digest, evaluatorID)` before returning, so the state is guaranteed to have the correct evaluator-id at this point.

### D4: Invalidation helper scans generation/*.json files

**Decision**: `InvalidateForEvaluator(baseDir, evaluatorID)` reads all `.json` files in `{baseDir}/.complytime/generation/`, unmarshals each to check the `evaluator_ids` field, and removes files where the slice contains the target evaluator-id.

**Rationale**: Generation state files are keyed by policy repository (not evaluator-id), so a scan is necessary to find affected files. The generation directory is small (one file per configured policy), so the scan is cheap.

### D5: Non-fatal invalidation errors are logged, not returned

**Decision**: If invalidation fails (e.g., permission error on removal), log a warning to stderr and continue. Do not fail the `get` command.

**Rationale**: The complypack was successfully fetched and cached. Failing `get` due to a workspace cleanup issue would be confusing. The next `scan` will trigger regeneration anyway via the digest mismatch path (secondary defense).

## Risks / Trade-offs

- **[`get` now touches workspace]** Previously `get` only wrote to the global cache (`~/.complytime/`). Now it also modifies workspace-local files. This is acceptable — the alternative is silent corruption.
- **[Scan after interrupted invalidation]** If `get` crashes between writing the new complypack and completing invalidation, stale state may persist. Mitigation: the digest-based `IsFresh()` check in `scan` acts as a secondary defense.
- **[Multi-workspace scenarios]** If multiple workspaces reference the same complypack, only the resolved workspace is invalidated. Other workspaces will rely on lazy digest detection. This is acceptable for the initial fix; a follow-up could invalidate all known workspaces.
