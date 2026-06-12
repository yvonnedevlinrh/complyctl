## 1. Plan ID Threading

- [x] 1.1 Add `reqToPlan map[string]string` field to `Evaluator` struct and accept it in `NewEvaluator()` (`internal/output/evaluator.go`)
- [x] 1.2 Build reverse map (`reqToPlan`) from `extractPlanToReqMap` data in `runScanAndReport()` (`cmd/complyctl/cli/scan.go`)
- [x] 1.3 Pass `reqToPlan` through `processScanOutput()` and `buildEvaluators()` into `NewEvaluator()` (`cmd/complyctl/cli/scan.go`)
- [x] 1.4 Populate `Plan: &gemara.EntryMapping{ReferenceId: e.policyID, EntryId: planID}` in `providerToGemaraAssessment()` using `reqToPlan` lookup (`internal/output/evaluator.go`)
- [x] 1.5 Add unit tests for plan field population: plan present, plan absent, plan omitted when no mapping exists (`internal/output/evaluator_test.go`)

## 2. Complypack OCI Ref Threading

- [x] 2.1 Add `complypackRef string` field to `Evaluator` struct and accept it in `NewEvaluator()` (`internal/output/evaluator.go`)
- [x] 2.2 Build evaluator-id to complypack OCI ref map from `State.Complypacks` and complypack cache in `runScanAndReport()` or `processScanOutput()` (`cmd/complyctl/cli/scan.go`)
- [x] 2.3 Pass complypack OCI ref through `buildEvaluators()` into `NewEvaluator()` (`cmd/complyctl/cli/scan.go`)
- [x] 2.4 Add unit test verifying complypack ref is threaded to the evaluator (`cmd/complyctl/cli/cli_test.go`)

## 3. Shadow Struct for Step Serialization

- [x] 3.1 Define shadow structs mirroring `gemara.EvaluationLog`, `gemara.ControlEvaluation`, and `gemara.AssessmentLog` with `Steps []string` instead of `Steps []AssessmentStep` (`internal/output/evaluator.go`)
- [x] 3.2 Add `formatStepIdentity(complypackRef, stepName string) string` helper that produces `{ref}#{name}` or bare `{name}` (`internal/output/evaluator.go`)
- [x] 3.3 Add conversion function from `gemara.EvaluationLog` to shadow struct, extracting step names from closures via a stored name slice or parallel tracking (`internal/output/evaluator.go`)
- [x] 3.4 Update `Write()` to marshal the shadow struct instead of the gemara struct (`internal/output/evaluator.go`)
- [x] 3.5 Add unit tests for `formatStepIdentity()`: with OCI ref, without OCI ref, empty step name (`internal/output/evaluator_test.go`)
- [x] 3.6 Add unit test for `Write()` verifying serialized YAML contains step identity strings instead of function pointer names (`internal/output/evaluator_test.go`)

## 4. Step Name Tracking

- [x] 4.1 Store step names alongside closures in `providerToGemaraAssessment()` so shadow struct conversion can access them (add `stepNames []string` field to `Evaluator` or accumulate per-assessment) (`internal/output/evaluator.go`)
- [x] 4.2 Add unit test verifying step names are preserved through the conversion pipeline (`internal/output/evaluator_test.go`)

## 5. Integration Verification

- [x] 5.1 Run `make test-unit` and verify all new and existing tests pass
- [x] 5.2 Run `make lint` and fix any linter warnings
- [ ] 5.3 Verify evaluation log YAML output in devcontainer with OPA and Ampel providers: confirm `plan` field present and `steps` contain identity strings (or bare names if providers have not yet been updated)
