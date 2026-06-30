## ADDED Requirements

### Requirement: Get command invalidates generation cache on complypack re-fetch

When `complyctl get` fetches a complypack that replaces an existing cached version, the command SHALL invalidate the workspace generation state for all policies that reference the affected evaluator-id. The command SHALL also remove the workspace evaluator artifact directory for the affected evaluator-id.

#### Scenario: Complypack re-sync triggers generation invalidation

- **GIVEN** a workspace with generation state referencing evaluator "opa" and evaluator artifacts at `.complytime/opa/`
- **WHEN** `complyctl get` fetches a new complypack for evaluator "opa" (`fetched == true`)
- **THEN** the generation state file(s) whose `evaluator_ids` include "opa" SHALL be deleted
- **AND** the workspace evaluator artifact directory `.complytime/opa/` SHALL be removed
- **AND** a message SHALL be emitted to stderr indicating the invalidation occurred

#### Scenario: Complypack incremental skip does not invalidate

- **GIVEN** a workspace with generation state referencing evaluator "opa"
- **WHEN** `complyctl get` determines the remote complypack digest matches the local cache (`fetched == false`)
- **THEN** the generation state SHALL NOT be modified
- **AND** the workspace evaluator artifact directory SHALL NOT be removed

#### Scenario: Invalidation preserves unrelated evaluator state

- **GIVEN** a workspace with generation state for evaluators "opa" and "ampel"
- **WHEN** `complyctl get` fetches a new complypack for evaluator "opa" only
- **THEN** only generation state files referencing "opa" SHALL be deleted
- **AND** generation state files referencing only "ampel" SHALL be preserved
- **AND** the `.complytime/ampel/` artifact directory SHALL NOT be removed

#### Scenario: No generation state directory exists

- **GIVEN** a workspace with no `.complytime/generation/` directory
- **WHEN** `complyctl get` fetches a complypack
- **THEN** the invalidation SHALL complete without error (no-op)

#### Scenario: No evaluator artifact directory exists

- **GIVEN** a workspace with generation state but no `.complytime/{evaluator}/` directory
- **WHEN** `complyctl get` fetches a complypack for that evaluator
- **THEN** the generation state SHALL be deleted
- **AND** the artifact removal SHALL complete without error (no-op for missing dir)

## MODIFIED Requirements

None — existing generation freshness checking via `IsFresh()` remains as the secondary defense layer. This change adds eager invalidation as a primary defense.

## REMOVED Requirements

None.
