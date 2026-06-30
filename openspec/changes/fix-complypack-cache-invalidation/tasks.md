## 1. Add generation state invalidation helpers

- [x] 1.1 Add `InvalidateForEvaluator(baseDir, evaluatorID string) (warnings []string, _ error)` to `internal/policy/generation_state.go` — walks `.complytime/generation/` tree (supports nested policy IDs), loads each `.json`, removes files whose `EvaluatorIDs` slice contains the target evaluator-id. Returns nil if the generation directory does not exist. Reports skipped files as warnings.
- [x] 1.2 Add `RemoveEvaluatorArtifacts(baseDir, evaluatorID string) error` to `internal/policy/generation_state.go` — calls `os.RemoveAll` on `{baseDir}/.complytime/{evaluatorID}/`. Returns nil if the directory does not exist. Validates evaluatorID against path traversal.
- [x] 1.3 [P] Add unit tests in `internal/policy/generation_state_test.go`:
  - `TestInvalidateForEvaluator_RemovesMatchingState`
  - `TestInvalidateForEvaluator_PreservesUnrelatedState`
  - `TestInvalidateForEvaluator_NoGenerationDir`
  - `TestInvalidateForEvaluator_MalformedJSON` (verifies warning returned)
  - `TestInvalidateForEvaluator_UnreadableFile` (verifies warning returned)
  - `TestInvalidateForEvaluator_NestedPolicyID`
  - `TestInvalidateForEvaluator_NestedNonJSONPreserved`
  - `TestInvalidateForEvaluator_IgnoresNonJSONFiles`
  - `TestRemoveEvaluatorArtifacts_RemovesDir`
  - `TestRemoveEvaluatorArtifacts_DirNotExist`
  - `TestRemoveEvaluatorArtifacts_RejectsPathTraversal`

## 2. Thread baseDir through complypack sync in get.go

- [x] 2.1 Add `baseDir string` parameter to `syncComplypacks` method on `getOptions`
- [x] 2.2 Add `baseDir string` parameter to `syncAllComplypacks` function
- [x] 2.3 Add `baseDir string` parameter to `syncSingleComplypack` function
- [x] 2.4 Update `syncAll` to pass `baseDir` (already available from `run()` via closure or parameter threading)

## 3. Wire invalidation on complypack fetch

- [x] 3.1 In `syncSingleComplypack`, when `fetched == true`: read `state.Complypacks[ref.Repository].EvaluatorID`
- [x] 3.2 Call `policy.InvalidateForEvaluator(baseDir, evaluatorID)` — log skip-warnings at Debug level, log error at Warn level, do not fail
- [x] 3.3 Call `policy.RemoveEvaluatorArtifacts(baseDir, evaluatorID)` — log warning on error, do not fail
- [x] 3.4 Emit stderr message: `"Complypack %s updated — generation cache invalidated for %s\n"`

## 4. Integration test

- [x] 4.1 [P] Add tests in `cmd/complyctl/cli/cli_test.go`:
  - `TestInvalidateGenerationForComplypack_InvalidatesOnFetch`
  - `TestInvalidateGenerationForComplypack_NestedPolicyID`
  - `TestInvalidateGenerationForComplypack_NoOpWhenRepositoryUnknown`
  - `TestInvalidateGenerationForComplypack_NoOpWhenEvaluatorIDEmpty`

## 5. Verification

- [x] 5.1 Run `go test -race ./...` and confirm all tests pass
- [x] 5.2 Run `go vet ./...` and confirm no vet errors
- [x] 5.3 Verify #551 reproduction steps no longer produce the stale-path error (local repro with mock-oci-registry)
