## ADDED Requirements

### Requirement: Assessment plan field populated in evaluation logs
The evaluation log output SHALL populate the `plan` field on each `gemara.AssessmentLog` entry with an `EntryMapping` that links the assessment result to the assessment plan that triggered it. The `EntryMapping.ReferenceId` SHALL be the policy repository identifier and `EntryMapping.EntryId` SHALL be the assessment plan ID from the dependency graph.

#### Scenario: Plan field present in evaluation log YAML
- **WHEN** `complyctl scan` produces an evaluation log for a policy with assessment plans
- **THEN** each `assessment-logs` entry in the YAML output SHALL contain a `plan` field with `reference-id` set to the policy repository and `entry-id` set to the assessment plan ID

#### Scenario: Plan field absent when no plan ID is available
- **WHEN** `complyctl scan` produces an evaluation log and no plan-to-requirement mapping exists for a given assessment
- **THEN** the `plan` field SHALL be omitted from that `assessment-logs` entry (the field is optional per Gemara schema)

### Requirement: Step identity strings in evaluation logs
The evaluation log output SHALL serialize `gemara.AssessmentStep` entries as human-readable identity strings instead of Go function pointer names. Each step identity string SHALL combine the complypack OCI reference with the provider-supplied step name.

#### Scenario: Step identity with complypack OCI ref
- **WHEN** `complyctl scan` produces an evaluation log and a complypack OCI ref is available for the evaluator
- **THEN** each step in the `steps` array SHALL be serialized as `{repository}@{digest}#{step-name}` where `repository` and `digest` come from the complypack cache state and `step-name` comes from `Step.Name`

#### Scenario: Step identity without complypack
- **WHEN** `complyctl scan` produces an evaluation log and no complypack is configured for the evaluator
- **THEN** each step in the `steps` array SHALL be serialized as the bare `Step.Name` value from the provider response

#### Scenario: Step identity with empty Step.Name
- **WHEN** `complyctl scan` produces an evaluation log and the provider did not set `Step.Name`
- **THEN** the step SHALL be serialized as an empty string (or omitted if the serialization library skips empty values)

### Requirement: Traceability chain completeness
The evaluation log SHALL provide a complete traceability chain from requirement through plan to step for each assessment result.

#### Scenario: Full traceability chain
- **WHEN** `complyctl scan` produces an evaluation log for a policy with assessment plans and a provider that sets step names
- **THEN** the evaluation log SHALL contain: `requirement.entry-id` (the requirement ID), `plan.entry-id` (the assessment plan ID), and `steps` entries (step identity strings linking to the specific checks executed)

### Requirement: Backward compatibility
The evaluation log output SHALL remain valid when complypacks are not configured or when providers do not set step names.

#### Scenario: No complypack configured
- **WHEN** `complyctl scan` runs against a policy without complypacks
- **THEN** the evaluation log SHALL omit the complypack OCI ref prefix from step identities and the `plan` field SHALL still be populated if plan IDs are available from the dependency graph

#### Scenario: Provider does not set Step.Name
- **WHEN** `complyctl scan` runs and a provider returns steps without setting the `Name` field
- **THEN** the evaluation log SHALL still serialize steps (with empty or omitted identity strings) and SHALL still populate `plan` and `requirement` fields
