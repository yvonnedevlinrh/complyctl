## Why

`complyctl scan` cannot distinguish operational errors from evaluation results. The `ScanResponse` proto now carries a structured `errors` field (added in `bug/scan-proto`) and the manager exposes `RouteScanResult()` with `HasErrors()`. But the CLI still calls `RouteScan()`, discards operational errors, and exits 0 regardless of whether providers reported coverage gaps.

A scan that silently fails to evaluate 3 of 5 targets produces the same exit code as a clean scan. CI pipelines and operators have no signal that compliance coverage is incomplete.

## What Changes

- `scanSingleTarget()` switches from `RouteScan()` to `RouteScanResult()`, propagating `ScanResult.Errors` upward through the call chain.
- `complyctl scan` exits non-zero when `ScanResult.HasErrors()` is true (operational failures). Policy violations (ResultFailed) continue to exit 0 — findings are expected output, not command failures.
- `FormatScanSummary()` renders operational errors as a distinct warnings block before the assessment table.

## Capabilities

### New Capabilities
- `scan-error-exit-code`: `complyctl scan` exits non-zero when providers report operational errors (coverage gaps). Exit 0 means the scan completed — policy violations are data, not failures.
- `scan-error-display`: Operational errors rendered as a distinct warnings section in scan output, separated from the evaluation results table.

### Modified Capabilities
- `scan-summary`: `FormatScanSummary` accepts operational errors and renders them before the assessment table.
- `scan-single-target`: `scanSingleTarget` returns structured `ScanResult` instead of flat `[]AssessmentLog`.

### Removed Capabilities
None.

## Impact

- **CLI surface**: Exit code semantics change. Previously exit 0 for all successful RPC calls. Now exit non-zero when `ScanResult.Errors` is non-empty. Backward compatible for providers that don't populate the `errors` field (empty errors = exit 0).
- **Code**: `cmd/complyctl/cli/scan.go` (scan execution chain), `internal/output/scan_summary.go` (error display).
- **Tests**: Unit tests for error propagation, exit code behavior, and error display formatting.
- **No dependency changes**.

## Constitution Alignment

Assessed against the ComplyTime Constitution.

### I. Autonomous Collaboration

**Assessment**: PASS

Operational errors surface as structured data (`ScanResult.Errors`) alongside machine-parseable assessment results. No information is lost or buried in log-only channels.

### II. Composability First

**Assessment**: PASS

`RouteScanResult()` is additive — `RouteScan()` preserved as a wrapper. `FormatScanSummary` gains an optional parameter. No mandatory dependencies introduced.

### III. Observable Quality

**Assessment**: PASS

This change improves observability. Operational errors that were previously silent (absorbed into synthetic assessment entries) now produce distinct, machine-parseable output and a non-zero exit code.

### IV. Testability

**Assessment**: PASS

`ScanResult.HasErrors()` is independently testable. Error propagation through the scan chain is testable via mock providers. `FormatScanSummary` with errors is testable in isolation.
