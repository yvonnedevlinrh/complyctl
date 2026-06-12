## Context

Evaluation logs produced by `complyctl scan` follow the Gemara `EvaluationLog` schema. The conversion from `provider.AssessmentLog` (data struct with `RequirementID`, `Steps []Step`) to `gemara.AssessmentLog` happens in `internal/output/evaluator.go`. PR #575 fixed the structural bug where `Steps` was always empty by wrapping `provider.Step` into `gemara.AssessmentStep` closures, but two identity problems remain:

1. `gemara.AssessmentLog.Plan` (`*EntryMapping`) is never populated. The assessment plan ID exists in the dependency graph (`policy.Assessment.ID`) and is used to build configs for providers, but it is discarded during post-scan ID resolution (`resolveAssessmentIDs` overwrites `RequirementID` with the resolved requirement ID, losing the plan ID).

2. `gemara.AssessmentStep` is a function type (`func(payload interface{}) (Result, string, ConfidenceLevel)`). When YAML-marshaled, Go serializes it as the runtime function pointer name (e.g., `github.com/complytime/complyctl/internal/output.providerStepsToGemara.providerStepToGemara.func1`). The step identity should be a meaningful string linking to the check that was run.

Key constraints:
- `gemara.AssessmentStep` is defined in go-gemara as a function type and cannot change
- No new proto fields on `Step` (CUE validation constraint)
- Providers already receive `Step.Name` via gRPC and can set it to complypack mapping IDs
- Complypack OCI refs (repository + digest) are tracked in `internal/cache/state.go` via `State.Complypacks`

## Goals / Non-Goals

**Goals:**
- Populate `gemara.AssessmentLog.Plan` with the assessment plan's `EntryMapping` so each assessment result links to its originating plan
- Produce human-readable, digest-pinned step identity strings in serialized evaluation logs
- Preserve backward compatibility: evaluation logs without complypacks still produce valid output (step names fall back to provider-supplied `Step.Name`)

**Non-Goals:**
- Changing the `gemara.AssessmentStep` function type in go-gemara
- Adding new proto fields to `Step` or `AssessmentConfiguration`
- Setting `Step.Name` to complypack mapping IDs on the provider side (that is a `complytime-providers` change)
- Modifying SARIF, OSCAL, or Markdown output formatters (this change scopes to evaluation log only)

## Decisions

### D1: Preserve plan IDs through a reverse map

**Decision**: Build a reverse map (`reqToPlan`: requirement_id -> plan_id) from the same `extractPlanToReqMap` data and pass it to `NewEvaluator()`.

**Rationale**: `resolveAssessmentIDs()` destructively overwrites `RequirementID` with the resolved requirement ID before evaluators run. Rather than changing the mutation order or adding a field to `provider.AssessmentLog`, a reverse map lets the evaluator look up the original plan ID from the already-resolved requirement ID.

**Alternatives considered**:
- Store original plan ID on `provider.AssessmentLog` before mutation: Requires changing a cross-package type for a single consumer. More invasive.
- Defer `resolveAssessmentIDs()` to after evaluator construction: Other formatters (SARIF, Markdown, scan summary) also depend on the resolved IDs. Reordering creates risk.
- Pass `planToReq` map directly and invert inside evaluator: Pushes knowledge of the naming inversion into the output package. The scan command already has both maps available.

### D2: Shadow struct for step serialization in Write()

**Decision**: In `Evaluator.Write()`, build a parallel struct that mirrors `gemara.EvaluationLog` but replaces `Steps []AssessmentStep` (function slice) with `Steps []string` (identity strings). Marshal the shadow struct instead of the gemara struct.

**Rationale**: `gemara.AssessmentStep` is a function type that cannot be meaningfully serialized by any YAML library. The shadow struct approach keeps the gemara domain types unchanged in memory (closures remain callable for any future in-process use) while controlling serialization output. This is a localized change in one method.

**Alternatives considered**:
- Custom YAML marshaler on `gemara.AssessmentLog`: Would require go-gemara changes or type aliasing hacks.
- Store step identity strings alongside closures in a parallel slice on `gemara.AssessmentLog`: Requires go-gemara type changes.
- Post-process the YAML bytes with string replacement: Fragile and dependent on marshal output format.

### D3: Step identity format

**Decision**: Step identity strings use the format `{oci-ref}#{step-name}` where `oci-ref` is `{repository}@{digest}` and `step-name` is the provider-supplied `Step.Name`. When no complypack OCI ref is available, fall back to bare `Step.Name`.

**Format**: `registry.example.com/complypacks/opa@sha256:abc123#kubernetes.run_as_nonroot`

**Rationale**: The OCI ref pins the exact complypack version, making step identity reproducible across runs. The `#` fragment separator is a natural delimiter that avoids ambiguity with OCI ref syntax. Bare `Step.Name` fallback ensures backward compatibility when no complypack is configured.

### D4: Thread complypack OCI ref via evaluator-id keyed lookup

**Decision**: In `scan.go`, after loading cache state, build a map from evaluator-id to complypack OCI ref string (`repository@digest`). Pass the OCI ref for each evaluator group into the evaluator construction. The `Evaluator` stores a single `complypackRef` string.

**Rationale**: `State.Complypacks` is keyed by repository, but evaluator groups are keyed by evaluator-id. The complypack cache's `LookupByEvaluatorID()` already bridges this gap. The cache state's `PolicyState.Digest` provides the digest. Building the OCI ref string at the scan command level keeps the evaluator package unaware of cache internals.

## Risks / Trade-offs

- [Plan ID uniqueness] The reverse map (`reqToPlan`) assumes a 1:1 mapping from requirement ID to plan ID. If multiple plans map to the same requirement ID, only one plan ID is preserved. -> Mitigation: The Gemara schema specifies one assessment plan per requirement-id in a given policy. This is a structural constraint, not a runtime concern.
- [Step.Name not set by providers] Until providers are updated to set `Step.Name` to complypack mapping IDs, step identity strings will be empty or generic. -> Mitigation: The complyctl-side change is forward-compatible. Step identity degrades gracefully to bare `Step.Name` or empty string. Provider updates are tracked in `complytime-providers`.
- [Shadow struct maintenance] The shadow struct duplicates parts of the gemara type hierarchy. If go-gemara adds fields to `AssessmentLog`, the shadow struct must be updated. -> Mitigation: The shadow struct is minimal (only replaces the `Steps` field type). A compile-time test can verify field coverage.
