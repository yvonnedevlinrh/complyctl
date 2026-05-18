## ADDED Requirements

### Requirement: Scan exits non-zero on operational errors

The `complyctl scan` command SHALL exit with a non-zero exit code when one or more providers report operational errors via `ScanResponse.errors`. Policy violations (ResultFailed) SHALL NOT cause a non-zero exit code.

#### Scenario: All targets scanned, some policies failed
- **GIVEN** a provider that evaluates all targets and returns assessments with ResultFailed but no operational errors
- **WHEN** the user runs `complyctl scan`
- **THEN** the command SHALL exit with code 0

#### Scenario: Some targets had operational errors
- **GIVEN** a provider that returns partial assessments and `errors: ["target 'staging': clone failed"]`
- **WHEN** the user runs `complyctl scan`
- **THEN** the command SHALL write reports from the partial results AND exit with a non-zero code

#### Scenario: Provider does not populate errors field
- **GIVEN** a provider that returns `ScanResponse` with an empty `errors` field (pre-existing provider)
- **WHEN** the user runs `complyctl scan`
- **THEN** the command SHALL behave identically to current behavior (exit 0 if RPC succeeded)

#### Scenario: Fatal pre-scan error
- **GIVEN** a provider that returns `(nil, error)` from the Scan RPC
- **WHEN** the user runs `complyctl scan`
- **THEN** the command SHALL exit with a non-zero code and include the error in operational warnings

### Requirement: Operational errors displayed as distinct warnings

Operational errors from `ScanResponse.errors` SHALL be displayed as `WARNING:` lines to stderr before the scan summary table. They MUST NOT be mixed into the assessment results table.

#### Scenario: Operational errors with partial results
- **GIVEN** a provider that returns 3 assessments and 1 operational error
- **WHEN** the scan output is rendered
- **THEN** the warnings section SHALL appear before the scan summary, and the assessment table SHALL contain only the 3 evaluation results

#### Scenario: No operational errors
- **GIVEN** a provider that returns assessments with an empty `errors` field
- **WHEN** the scan output is rendered
- **THEN** no warnings section SHALL appear (output identical to current behavior)

### Requirement: Reports written before error exit

When operational errors are present, the scan command SHALL write all output reports (evaluation log, format reports, scan summary) BEFORE returning the non-zero exit code. Partial results MUST be available to the operator.

#### Scenario: Partial results with operational errors
- **GIVEN** a provider that returns assessments for 2 of 3 targets and 1 operational error
- **WHEN** the user runs `complyctl scan --format pretty`
- **THEN** the evaluation log and markdown report SHALL be written with the 2 successful target results, AND the command SHALL exit non-zero afterward

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
