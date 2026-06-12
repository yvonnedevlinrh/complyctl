## Why

After PR #575 fixes the structural bug where evaluation logs showed `steps: []`, two identity gaps remain in the serialized output. Steps display Go function pointer names (e.g., `...providerStepToGemara.func1`) instead of meaningful identifiers, and the `plan` field on `gemara.AssessmentLog` is never populated. This breaks the requirement-to-plan-to-step traceability chain that Gemara evaluation logs are designed to provide. Without it, consumers of evaluation logs cannot trace an assessment result back to the specific plan that was executed or the specific check that produced it.

## What Changes

- Populate the `gemara.AssessmentLog.Plan` field (`*EntryMapping`) with the assessment plan ID, linking each assessment result to the policy plan that triggered it
- Thread plan ID from the dependency graph through the scan pipeline into the `Evaluator`, preserving plan identity alongside the resolved requirement ID
- Accept complypack OCI ref (repository + digest) in the `Evaluator` and combine it with `Step.Name` to produce stable, digest-pinned step identity strings at serialization time
- Override step serialization in `Write()` using a shadow struct so that `gemara.AssessmentStep` function types render as human-readable identity strings (e.g., `registry.example.com/complypacks/opa@sha256:abc123#kubernetes.run_as_nonroot`) instead of Go function pointer names

## Capabilities

### New Capabilities
- `eval-log-traceability`: Populating the assessment plan field and producing meaningful step identity strings in evaluation log output

### Modified Capabilities

## Impact

- `internal/output/evaluator.go`: New fields on `Evaluator` struct (plan map, complypack OCI ref), updated `providerToGemaraAssessment()`, shadow struct in `Write()`
- `cmd/complyctl/cli/scan.go`: Thread reverse plan map and complypack OCI ref from cache state into evaluator construction
- `internal/cache/state.go`: Read-only usage of existing `State.Complypacks` entries (no changes to state itself)
- No proto changes (`api/plugin/plugin.proto`): step identity uses existing `Step.name` field
- No go-gemara changes: `AssessmentStep` stays a function type; shadow struct handles serialization
- Provider-side (`complytime-providers`): Providers must set `Step.Name` to complypack mapping IDs -- tracked separately in complytime-providers, not in this change
