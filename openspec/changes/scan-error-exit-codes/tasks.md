<!--
  [P] marks tasks eligible for parallel execution.
  Add [P] when a task: (a) touches different files from
  other [P] tasks in the group, (b) has no dependency
  on prior tasks in the group, (c) can safely execute
  without ordering constraints.
  Do NOT add [P] when tasks modify the same file —
  parallel workers will cause merge conflicts.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 1. Wire RouteScanResult into CLI scan chain

- [x] 1.1 Add `scanOutput` struct to `cmd/complyctl/cli/scan.go` with `assessments`, `assessmentTargets`, and `errors` fields (per D5)
- [x] 1.2 Update `scanSingleTarget()` to call `mgr.RouteScanResult()` instead of `mgr.RouteScan()`, return `ScanResult.Errors` alongside assessments
- [x] 1.3 Update `scanAllTargets()` to collect errors from each `scanSingleTarget()` call into `scanOutput.errors`
- [x] 1.4 Update `executeScan()` to return `scanOutput` instead of `([]AssessmentLog, []string, error)`
- [x] 1.5 Update `runScanAndReport()` to accept `scanOutput`, write reports from assessments, then check errors

## 2. Implement error display and exit code

- [x] 2.1 [P] Add `FormatOperationalWarnings(errors []string) string` to `internal/output/scan_summary.go` — renders `WARNING: N operational error(s)` block
- [x] 2.2 In `runScanAndReport()`, print operational warnings to stderr before scan summary if errors are non-empty
- [x] 2.3 In `runScanAndReport()`, return `fmt.Errorf` with error count summary when `scanOutput.errors` is non-empty (triggers non-zero exit via cobra)

## 3. Tests

- [x] 3.1 [P] Add unit test: `scanSingleTarget` with mock provider returning `ScanResult.Errors` propagates errors
- [x] 3.2 [P] Add unit test: `FormatOperationalWarnings` renders distinct warnings block
- [x] 3.3 [P] Add unit test: `FormatOperationalWarnings` with empty errors returns empty string
- [x] 3.4 [P] Add unit test: clean scan (no errors) produces exit 0
- [x] 3.5 [P] Add unit test: scan with policy violations but no operational errors produces exit 0

## 4. Verify

- [x] 4.1 Run `go build ./...` and `make test-unit`
- [x] 4.2 Run `make lint`
- [x] 4.3 Update CHANGELOG.md with entry under `### Fixed`
