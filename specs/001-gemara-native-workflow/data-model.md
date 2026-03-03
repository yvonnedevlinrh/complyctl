# Data Model: Gemara-Native Decoupled Workflow

**Branch**: `001-gemara-native-workflow` | **Date**: 2026-02-14 (updated 2026-02-27)

## Entities

### WorkspaceConfig

Workspace configuration file (YAML). Single mode using `PolicyEntry` objects (Session 2026-02-25d). No `pack` field, no separate `registry` section — each policy URL is a self-contained OCI reference including its own registry.

| Field | Type | Required | Description |
|:---|:---|:---|:---|
| `policies` | `[]PolicyEntry` | Yes (>=1) | Policies to fetch and evaluate. Each entry has a full OCI URL and optional shortname ID |
| `variables` | `map[string]string` | No | Global variables — workspace-scoped config (e.g., `workspace: ./.complytime/scan`). Passed to providers during Generate RPC (R48, R49) |
| `targets` | `[]TargetConfig` | No | Systems/environments to scan |

**Source**: `internal/complytime/config.go` — `WorkspaceConfig` struct
**Note**: `parameters` / `ParameterOverride` removed (Session 2026-02-24c). Policy-defined defaults used directly; no admin overrides. `pack` field and `RegistryConfig` removed (Session 2026-02-25d) — each policy URL contains its own registry. `evaluator_config` replaced by three-tier variable model (R48).
**File path**: `./complytime.yaml` (static convention, like `go.mod` — not configurable via flags, R50)
**Validation**: Non-empty policies list. Unique policy URLs. Unique effective IDs (explicit `id` or derived). Target policy IDs must reference a declared policy by effective ID.

**Authentication**: Zero custom code. Credential resolution delegated to `oras.land/oras-credentials-go` via `credentials.NewStoreFromDocker()` (R6, R24). Reads `~/.docker/config.json` automatically — credHelpers, credsStore, and inline auths resolved by the library. Per-registry clients created dynamically by `get` using `UniqueRegistries()`.

### PolicyEntry (Session 2026-02-25d)

| Field | Type | Required | Description |
|:---|:---|:---|:---|
| `url` | `string` | Yes | Full OCI reference including registry (e.g., `registry.com/policies/nist-800-53-r5@v1.0`) |
| `id` | `string` | No | User-chosen shortname. If omitted, auto-derived from last URL path segment via `EffectiveID()` |

**`EffectiveID()` method**: Returns explicit `id` if set; otherwise parses `url` via `ParsePolicyRef()`, splits `Repository` by `/`, returns the last segment (e.g., `registry.com/policies/nist-800-53-r5@v1.0` → `nist-800-53-r5`).

**`FindPolicy(policies []PolicyEntry, policyID string)`**: Resolves a user-provided policy identifier against the policies list. Match priority: (1) effective ID, (2) exact URL, (3) repository path. Returns `*PolicyEntry, bool`.

**`PolicyIDs(policies []PolicyEntry)`**: Creates a map of effective IDs → `*PolicyEntry` for fast lookup.

**`UniqueRegistries(policies []PolicyEntry)`**: Extracts distinct registry URLs from all policy entries. Used by `doctor` for registry reachability probes and by `get` for per-registry client creation.

**Parsing**: `ParsePolicyRef(raw string)` uses `oras.land/oras-go/v2` reference parsing to decompose a full OCI URL into `PolicyRef{Registry, Repository, Version}`.

**Source**: `internal/complytime/config.go` — `PolicyEntry` struct, `EffectiveID()`, `FindPolicy()`, `PolicyIDs()`, `UniqueRegistries()`, `ParsePolicyRef()`

### ~~RegistryConfig~~ (Removed — Session 2026-02-25d)

Global `RegistryConfig` removed. Each `PolicyEntry.url` contains its own registry. `get` dynamically creates per-registry OCI clients. Supersedes standalone-mode `registry` section and pack-mode `pack` field.

### ~~PolicyConfig~~ (Removed — Session 2026-02-25d)

Replaced by `PolicyEntry`. The separate `id` + `version` fields are now a single `url` (full OCI reference with optional version tag/digest) plus an optional `id` shortname.

### TargetConfig

| Field | Type | Required | Description |
|:---|:---|:---|:---|
| `id` | `string` | Yes | Target identifier |
| `policies` | `[]string` | Yes (>=1) | Policy effective IDs applicable to this target (Session 2026-02-25d; was `policy_ids`) |
| `variables` | `map[string]string` | No | Target variables — per-target runtime config (credentials, profile, kubeconfig) (R48) |

**Target variables (R48)**: `variables` contains per-target runtime configuration owned by the system admin. These define *how* to reach the target and *how* to configure the scan for this target (e.g., `profile: cis_workstation_l1`, `kubeconfig: /path/to/kubeconfig`). Passed to scanning providers via `Target.variables` in Scan RPC. Providers declare required variable *names* via `DescribeResponse.required_target_variables` (R51); `complyctl doctor` validates these keys exist. Documentation of valid *values* remains out-of-band (provider README).

**Validation rules**:
- No duplicate target IDs
- All `policies` entries must reference a declared policy by effective ID
- No duplicate policy effective IDs within a single target

### ~~ParameterOverride~~ (Removed — Session 2026-02-24c)

Parameter overrides removed. Policy-defined defaults are used directly. Scanning providers evaluate against accepted values from the policy without admin-configurable selections.

---

### PolicyCache

Local OCI Layout store per policy ID using `oras-go/v2/content/oci`. Each policy gets its own OCI Image Layout directory. Sync uses `oras.Copy()` from remote registry to local store.

```text
~/.complytime/
├── state.json                          # Sync state tracking (custom)
└── policies/
    └── {policy-id}/
        ├── index.json                  # OCI Image Index (tags → manifests)
        ├── oci-layout                  # OCI Layout marker {"imageLayoutVersion": "1.0.0"}
        └── blobs/
            └── sha256/
                ├── {manifest-digest}   # OCI manifest (references layer blobs)
                ├── {config-digest}     # OCI config blob
                ├── {layer-digest-1}    # Gemara layer content (bundles, guidelines, etc.)
                ├── {layer-digest-2}    # ...
                └── ...
```

**Source**: `internal/cache/cache.go` (wraps `oci.New()`), `internal/cache/state.go`
**Sync**: `internal/cache/sync.go` uses `oras.Copy(ctx, remoteRepo, ref, localStore, ref)`. Digests computed client-side by oras — no dependency on server `Docker-Content-Digest` header (R28).
**Reading**: `internal/policy/loader.go` resolves tags via `store.Resolve()`, fetches manifest via `store.Fetch()`, then iterates layer descriptors matching by media type (R25, R27):
- `application/vnd.gemara.catalog.v1+yaml` → catalog content
- `application/vnd.gemara.guidance.v1+yaml` → guidance content
- `application/vnd.gemara.policy.v1+yaml` → policy/assessment content

### CacheState (`state.json`)

| Field | Type | Description |
|:---|:---|:---|
| `last_sync` | `string` (ISO 8601) | Timestamp of last successful sync |
| `policies` | `map[string]PolicyState` | Per-policy sync state |

### PolicyState

| Field | Type | Description |
|:---|:---|:---|
| `version` | `string` | Cached version tag |
| `digest` | `string` | OCI manifest digest (SHA256) |
| `synced_at` | `string` (ISO 8601) | When this policy was last synced |

---

### DependencyGraph

In-memory resolved graph of Gemara Layers 1-3 for a specific policy. Supports multi-evaluator routing — each assessment plan's `evaluation-methods[].executor.id` determines which plugin handles that plan's requirements (R32).

| Field | Type | Description |
|:---|:---|:---|
| `policy_id` | `string` | Root policy identifier |
| `version` | `string` | Resolved version |
| `controls` | `[]gemara.Control` | Layer 2 controls (from `go-gemara`). For OpenSCAP: each XCCDF rule = one Control containing one Assessment Requirement (the testable statement) + guideline-mappings to CIS items (`cis-*` pattern) |
| `guidelines` | `[]gemara.Guideline` | Layer 1 guidelines. For OpenSCAP: CIS benchmark items (e.g., `cis-1.1.1.1`) referenced by control guideline-mappings |
| `assessments` | `[]gemara.AssessmentPlan` | Layer 3 assessment plans from Policy. For OpenSCAP: OVAL check definitions describing how to evaluate each control's assessment requirement |

**Source**: `internal/policy/resolver.go` — `ResolvePolicyGraph()`
**Consumers**: `generate` command (extracts assessment configs, groups by evaluator, persists artifacts, outputs execution plan), `scan` command (auto-generates if needed, routes to plugins)
**Multi-evaluator (R32)**: `GroupByEvaluator()` groups assessment configs by per-plan `executor.id`, not a single policy-level evaluator. A single policy may dispatch to N evaluators.
**Policy layer parsing (R39)**: `parsePolicyLayer` accepts only `gemara.Policy` with `adherence.assessment-plans`. Returns `(policyLayerResult, error)`. Structured and legacy format support removed.

---

### AssessmentConfiguration

Configuration extracted from the policy graph and passed to plugins via Generate RPC.

| Field | Type | Description |
|:---|:---|:---|
| `plan_id` | `string` | Assessment plan identifier |
| `requirement_id` | `string` | Requirement identifier |
| `evaluator_id` | `string` | Target evaluator (maps to plugin) |
| `parameters` | `map[string]string` | Key-value parameters |

**Source**: `internal/policy/assessment.go` — `ExtractAssessmentConfigs()`, `GroupByEvaluator()`, `ValidateGlobalVars()`

**Test variables (R48)**: `parameters` contains test configuration (test variables) from Layer 3 Gemara policy. These are owned by the policy author and define *what* a test checks (e.g., `var_password_hashing_algorithm: SHA512`). Passed to scanning providers via Generate RPC alongside global variables.

**Parameters (R23, updated Session 2026-02-24c)**: Parameters use policy-defined defaults. No user-configurable parameter overrides. Scanning providers evaluate against accepted values from the policy directly.

**Scan contract (R47)**: Scan RPC receives targets only (target ID + target variables). No requirement_ids — scanning provider evaluates all requirements from Generate-time state.

**Global variable validation (R48, R49)**: `ValidateGlobalVars()` checks that required global variables are present in the workspace config `variables` section before dispatching Generate RPC. Called in both `generate.go` and `scan.go` after `GroupByEvaluator()`. Replaces `ValidateEvaluatorConfig()`.

**Note (R30, Session 2026-02-25)**: AssessmentConfiguration replaces the former `policytype.Policy` / `oscalPolicy` type. In the OpenSCAP plugin, `requirement_id` maps 1:1 to Assessment Requirements within Controls (each XCCDF rule = one Control + one Assessment Requirement in the Gemara Control Catalog). `parameters` keys map to XCCDF variable IDs. Assessment Plans describe OVAL checks.

---

### GenerationState (persisted)

Tracks the policy cache digest at generation time for freshness detection (R37). Persisted after `complyctl generate` (or auto-generate within `scan`). Read by `scan` to determine whether to reuse or regenerate.

| Field | Type | Description |
|:---|:---|:---|
| `policy_id` | `string` | Policy that was generated |
| `policy_digest` | `string` | OCI manifest digest (SHA256) at generation time |
| `generated_at` | `string` (ISO 8601) | When generation occurred |
| `evaluator_ids` | `[]string` | Evaluator IDs used during generation |

**Source**: `internal/policy/generation_state.go` — `GenerationState`, `SaveState()`, `LoadState()`, `IsFresh(currentDigest)`
**File path**: `{workspace}/.complytime/generation/{policy-id}.json` — one file per policy, mirrors cache layout. Directory created on first generate.
**Freshness check**: Compare `policy_digest` against `PolicyState.digest` from `state.json`. Match → fresh (reuse). Mismatch → stale (warn + regenerate).
**~~Dry-run persistence~~**: ~~`scan --dry-run` persists `GenerationState`~~ (removed Session 2026-02-26e — `--dry-run` dropped; `generate` is the only execution plan preview). `generate` persists `GenerationState` after invoking Generate RPC. Plugins create artifacts as side effects; persisting state ensures a subsequent `scan` reuses them.

---

### ExecutionPlan (output of `complyctl generate`)

Structured plain-text execution plan output after graph resolution and plugin preparation (FR-007, Session 2026-02-26e). ~~`scan --dry-run` removed~~ (Session 2026-02-26e).

**ExecutionPlanRow** (one stanza per target-provider combination):

| Field | Type | Description |
|:---|:---|:---|
| `target_id` | `string` | Target identifier from workspace config |
| `provider_id` | `string` | Evaluator/provider identifier from assessment plan |
| `requirement_count` | `int` | Number of requirements routed to this provider for this target |
| `status` | `string` | `healthy`, `unhealthy`, or `ERROR` (no matching plugin) |

**Source**: `internal/output/execution_plan.go` — `FormatExecutionPlan()`
**Consumer**: `cmd/complyctl/cli/generate.go` — printed to stdout after graph resolution
**Rendering (Session 2026-02-26e)**: Structured plain-text block with indented labeled lines. Intro line with policy ID, one indented stanza per target-provider pair (Provider, Requirements count, Status), conclusion line. Uses `fmt.Fprintf` with alignment — no lipgloss, no table. Supersedes R59 lipgloss table design.

**~~EvaluatorRoute~~** and **~~TargetScope~~**: Removed (Session 2026-02-26d). Merged into `ExecutionPlanRow`.

---

### ScanningProvider (runtime)

Discovered gRPC scanning provider with lifecycle management. No sidecar manifest files — all metadata derived at runtime (R19). User-facing terminology: "scanning provider"; code-level: `plugin.Plugin` (R46).

| Field | Type | Description |
|:---|:---|:---|
| `plugin_id` | `string` | Derived from executable name |
| `evaluator_id` | `string` | `plugin_id` minus `complyctl-provider-` prefix |
| `path` | `string` | Filesystem path to executable |
| `healthy` | `bool` | Describe result |
| `version` | `string` | Reported by Describe RPC |
| `required_global_variables` | `[]string` | Global variable names declared by provider (R51). Empty = no requirements |
| `required_target_variables` | `[]string` | Target variable names declared by provider (R51). Empty = no requirements |
| `client` | `PluginClient` | gRPC client interface |

**Source**: `internal/plugin/manager.go`, `internal/plugin/client.go`
**Discovery (FR-029, Session 2026-02-27)**: Executable files matching `complyctl-provider-*` in two directories. User directory (`~/.complytime/providers/`) checked first, then system directory (`/usr/libexec/complytime/providers/`) as fallback. User-installed providers take precedence over system-installed providers with the same evaluator ID. System directory follows FHS convention for RPM-installed executables.
**Note**: ~~PluginManifest~~ removed (Session 2026-02-14c, R19). No YAML sidecar files. No checksum verification (R20). User-facing: "scanning provider"; code-level: `plugin` package (R46).

---

### EvaluationLog (output)

Gemara Layer 4 output — always produced on scan. Uses `go-gemara` `EvaluationLog` type directly.

| Field | Type | Description |
|:---|:---|:---|
| `metadata` | `gemara.Metadata` | Log metadata (title, datetime, actors) |
| `evaluations` | `[]gemara.ControlEvaluation` | Results grouped by control |

**ControlEvaluation** (one per control):

| Field | Type | Description |
|:---|:---|:---|
| `name` | `string` | Control identifier |
| `result` | `gemara.Result` | Aggregated result for this control |
| `control` | `gemara.EntryMapping` | Reference to the control (`EntryId` + `ReferenceId` = policy ID) |
| `assessment-logs` | `[]gemara.AssessmentLog` | Per-requirement assessment results within this control |

Each `AssessmentLog` within a `ControlEvaluation` contains `requirement` (`EntryMapping` with `EntryId` = requirement ID, `ReferenceId` = policy ID), `steps`, `result`, `confidence`, and `message`.

**Source**: `internal/output/evaluator.go` — builds `*gemara.EvaluationLog` using `gemara.ControlEvaluation` and `gemara.AssessmentLog`
**File path**: `{workspace}/{WorkspaceDir}/{ScanOutputDir}/evaluation-log.yaml` — always written to hidden directory (diagnostic artifact). Path always printed to terminal regardless of `--format` (R58). Formatted reports (`--format oscal|pretty|sarif`) are written to CWD (user-facing deliverables, R58).
**Format**: YAML (serialized via `goccy/go-yaml`)

### ScanSummary (terminal output)

Post-scan report-style output displayed in the terminal after every scan (FR-037). Not persisted to file — terminal output only.

**Design (Session 2026-02-26e)**: Report-style layout: (1) intro text (policy ID, target(s), total requirement count), (2) plain aligned text table of non-passing results with 4 columns, (3) conclusion text (compact inline totals, EvaluationLog path, formatted report path when `--format` specified). Passed results appear in the totals line only — not in the table. All rendering via `ShowPlainTable` — lipgloss removed.

**ScanSummaryEntry** (one row per non-passing result):

| Field | Type | Description |
|:---|:---|:---|
| `requirement_id` | `string` | Requirement identifier from `AssessmentLog.RequirementId` |
| `control_id` | `string` | Control identifier resolved via `reqToControl` map |
| `result` | `Result` (enum) | Aggregated assessment outcome (failed, skipped, error only) |
| `status_emoji` | `string` | Emoji indicator: ❌, ⏭️, ⚠️ (from `consts.StatusFailed`, etc.) |
| `message` | `string` | From `AssessmentLog.Steps[].Message` — the step whose result matches the aggregated outcome (first match) |

**Table columns**: `Requirement ID`, `Control ID`, `Status` (emoji), `Message`.

**Result aggregation**: Delegates to go-gemara's built-in results aggregation function. No custom `aggregateResultFromSteps()` in complyctl.

**Message source**: Scanning provider's `AssessmentLog.Steps[].Message` field directly. Provider authors control failure text. No requirement title substitution.

**Sort order**: Non-passing rows ordered by severity: ❌ failed (1) → ⚠️ error (2) → ⏭️ skipped (3). Passed results not listed individually.

**Totals line**: Compact inline below the table: `44 ✅  3 ❌  2 ⏭️  1 ⚠️`. Includes passed count.

**Source**: `internal/output/scan_summary.go` — `FormatScanSummary(assessments, reqToControl, policyID, targetIDs)`
**Consumer**: `cmd/complyctl/cli/scan.go` — printed to stdout after `eval.Write()`. EvaluationLog path included in conclusion text. Formatted report path included when `--format` is specified (report in CWD, R58).
**Rendering (Session 2026-02-26e)**: Plain aligned text table via `ShowPlainTable` in all contexts (TTY and non-TTY). Lipgloss removed. No `--pretty` flag.

**Constants** (from `internal/complytime/consts.go`, R42, R43):
- `ScanOutputDir = "scan"` — scan output subdirectory name
- `StatusPassed = "✅"` — passed emoji
- `StatusFailed = "❌"` — failed emoji
- `StatusSkipped = "⏭️"` — skipped emoji
- `StatusError = "⚠️"` — error emoji

---

### DoctorResult (terminal output)

Pre-flight diagnostics output from `complyctl doctor` (FR-039, R44, R55). Not persisted to file — terminal output only. Supports `--verbose` flag for per-provider variable detail (R55).

**CheckResult** (one per diagnostic check):

| Field | Type | Description |
|:---|:---|:---|
| `name` | `string` | Check identifier (e.g., `config`, `provider/{id}`, `policy/{id}`, `registry/{host}`, `variables/{id}`). Config check includes structural validation + target-policy cross-references (R50). Policy checks compare cached vs. remote version (R55). Variables checks validate Describe-declared `required_global_variables` against `config.variables` and `required_target_variables` against relevant `config.targets[].variables` using policy → evaluator → target mapping from cache (R51, R52) |
| `status` | `CheckStatus` (enum) | `pass`, `fail`, `warn` |
| `message` | `string` | Human-readable result (e.g., `complytime.yaml valid`, `v1.0.0 (latest)`, `cached v1.0.0, available v1.1.0`) |
| `blocking` | `bool` | Whether failure blocks exit code 0 |

**CheckStatus** (enum):

| Value | Emoji | Semantic |
|:---|:---|:---|
| `pass` | ✅ | Check passed |
| `fail` | ❌ | Check failed (blocking) |
| `warn` | ⚠️ | Check produced a non-blocking warning |

**Check categories (R55 updated)**:

| Check | Name Pattern | Blocking | Pass | Warn | Fail |
|:---|:---|:---|:---|:---|:---|
| Config | `config` | Yes | `complytime.yaml valid` | — | `config not found` / `validation failed` |
| Provider health | `provider/{id}` | Yes | `healthy (v1.2.0)` | — | `unhealthy` / `Describe failed` |
| Policy version | `policy/{id}` | No | `v1.0.0 (latest)` | `cached v1.0.0, available v1.1.0 — run complyctl get` | — |
| Registry reachability | `registry/{host}` | No | — | `unreachable — version check skipped` | — |
| Variables (default) | `variables/{id}` | Yes | `3/3 global vars, 2/2 target vars` | — | `2/3 global vars — missing workspace` |
| Variables (verbose) | `variables/{id}` | Yes | Expanded: per-key status lines | — | Expanded: per-key status lines with missing keys |
| Cache | `cache` | Yes | `N cached policy store(s)` | — | `policy cache not found` |

**`--verbose` flag (R55)**: Expands per-provider variable detail to show all expected keys and their resolved status. Does not expand version comparison output. Policy evaluation periods (active start/end dates) are a future `--verbose` candidate.

**Source**: `internal/doctor/doctor.go` — `Run()`
**Consumer**: `cmd/complyctl/cli/doctor.go` — printed to stdout as emoji + message per check
**Exit code**: 0 if all blocking checks pass. Non-zero if any blocking check fails. Policy staleness and registry unreachability are warnings (non-blocking).

---

### AssessmentLog (per requirement)

| Field | Type | Description |
|:---|:---|:---|
| `requirement_id` | `string` | Evaluated requirement |
| `steps` | `[]Step` | Ordered execution steps |
| `message` | `string` | Summary message |
| `confidence` | `ConfidenceLevel` (enum) | Confidence level: not_set, undetermined, low, medium, high (matches go-gemara) |
| `result` | `Result` (enum) | Assessment outcome: unspecified, passed, failed, skipped, error. Derived from step results at the domain layer — not a proto wire field. Uses the same `Result` enum as `Step.result`. |

### Step (per assessment step)

| Field | Type | Description |
|:---|:---|:---|
| `name` | `string` | Step name/identifier |
| `result` | `Result` (enum) | Step outcome |
| `message` | `string` | Step result message |

**Source**: `specs/001-gemara-native-workflow/contracts/plugin.proto` — `Step` message

### Result (enum)

| Value | Proto | Description |
|:---|:---|:---|
| `unspecified` | `RESULT_UNSPECIFIED` | Default/zero value — result not yet determined |
| `passed` | `RESULT_PASSED` | Requirement satisfied |
| `failed` | `RESULT_FAILED` | Requirement not satisfied |
| `skipped` | `RESULT_SKIPPED` | Requirement excluded by tailoring |
| `error` | `RESULT_ERROR` | Evaluation failed (plugin error, timeout, etc.) |

**Source**: `specs/001-gemara-native-workflow/contracts/plugin.proto` — `Result` enum

---

## Relationships

```text
WorkspaceConfig (Session 2026-02-25d)
  ├── 1:N PolicyEntry (url + optional id)
  │       ├── url ──→ OCI Registry (full reference, self-contained)
  │       └── EffectiveID() → explicit id OR derived from last URL path segment
  ├── variables (global variables — workspace-scoped, R48)
  └── 1:N TargetConfig
              ├── policies (effective IDs referencing PolicyEntry)
              └── variables (target variables — per-target, R48)

PolicyEntry ──(get, per-registry client)──→ PolicyCache
                                              └── DependencyGraph (resolved in-memory)
                                                     ├── Controls
                                                     ├── Guidelines
                                                     └── AssessmentRequirements
                                                            └── AssessmentConfiguration (test variables from policy, R48)
                                                                   └── N:1 ScanningProvider (via per-plan evaluator_id, R32)

complyctl providers ──→ ScanningProvider[] (discovery from ~/.complytime/providers/ + /usr/libexec/complytime/providers/, user precedence)
complyctl doctor ──→ DoctorResult (config + target-policy cross-refs + providers
                     + per-policy version comparison: cached vs remote latest (R55)
                     + per-provider config summary: resolved/missing counts (R55)
                     + Describe-declared variable validation via policy→evaluator→target mapping (R51)
                     + --verbose expands per-provider variable key detail (R55))
                     UniqueRegistries() extracts distinct registries for version queries
                     Requires policy cache (R52) — errors if cache missing

complyctl init ──→ create complytime.yaml (config-only, PolicyEntry prompts, errors if exists)
# User runs get and doctor separately after init
# Pack CLI commands (pack init, build, push, pull) deferred to 002

complyctl generate ──→ DependencyGraph
                       + global vars (workspace config)
                       + test vars (policy assessment plan)
                     ──→ Generate RPC ──→ GenerationState (persisted, R37)
                                       ──→ ExecutionPlan (stdout, R36)

complyctl scan ──→ check GenerationState.IsFresh()
                    ├── fresh → reuse artifacts
                    ├── stale → warn + auto-generate
                    └── missing → auto-generate
                  ──→ brief summary line (FR-034)
                  ──→ Scan RPC (targets only — R47) ──→ AssessmentLog[] ──→ EvaluationLog
                  ──→ ScanSummary (report-style: intro → 4-col plain table → totals + paths, FR-037)

[scan --dry-run removed Session 2026-02-26e — use complyctl generate for plan preview]

ScanningProvider ──(Generate RPC: global + test vars)──→ configured state
                 ──(Scan RPC: targets only)──→ AssessmentLog[] ──→ EvaluationLog (.complytime/scan/, R58)
                                               ├── OSCAL (--format oscal → CWD, R58)
                                               ├── Markdown (--format pretty → CWD, R58)
                                               └── SARIF (--format sarif → CWD, R58)

PackManifest (types in 001 data model only; ALL CLI deferred to 002; R53)
  ├── 1:1 PlatformConfig
  ├── 1:N PackPolicyEntry (each with own OCI URL) ──→ PolicyEntry (subset selected by consumer)
  ├── 1:N PackProviderEntry ──→ ScanningProvider (bundled binaries)
  └── 1:N SystemDependency ──→ doctor checks (shell commands, deferred to 002)
```

### PackManifest (types in 001 data model only; ALL CLI deferred to 002)

Pack manifest file (YAML). Declares what a comply-pack contains — developer-owned, immutable after build, ships in the pack. Separate from `complytime.yaml` (runtime config). All pack CLI commands (`pack init`, build, push, pull) deferred to 002. Pack builder is separate from `complyctl` runtime (Session 2026-02-25d). See R53, Session 2026-02-25b.

| Field | Type | Required | Owner | Description |
|:---|:---|:---|:---|:---|
| `id` | `string` | Yes | Developer | Pack identifier (e.g., `fedora-compliance`) |
| `version` | `string` | Yes | Developer | Semantic version of the pack |
| `description` | `string` | No | Developer | Human-readable description |
| `platform` | `PlatformConfig` | No | Developer | Target OS and datastream path |
| `policies` | `[]PackPolicyEntry` | Yes (>=1) | Developer | Policies to bundle (each with its own OCI URL) |
| `providers` | `[]PackProviderEntry` | Yes (>=1) | Developer | Provider binaries to bundle |
| `system-dependencies` | `[]SystemDependency` | No | Developer | OS packages required at runtime |

**Source**: `internal/complytime/pack.go` — `PackManifest` struct (data model in 001). CLI creation deferred to 002
**File path**: `./complypack.yaml` (static convention, same directory as `complytime.yaml`)
**Relationship to WorkspaceConfig**: `complypack.yaml` and `complytime.yaml` are peers. `doctor` reads both when both exist (deferred to 002). `complypack.yaml` absent = current 001 behavior. `complytime.yaml` absent but `complypack.yaml` present = pack OK + config missing with remediation guidance (002).

### PlatformConfig

| Field | Type | Required | Description |
|:---|:---|:---|:---|
| `os` | `string` | Yes | Target operating system (e.g., `fedora`) |
| `datastream` | `string` | No | Absolute path to SCAP datastream XML |

### PackPolicyEntry

| Field | Type | Required | Description |
|:---|:---|:---|:---|
| `url` | `string` | Yes | Full OCI reference (registry + repo + version) |
| `id` | `string` | Yes | Policy shortname |
| `profile` | `string` | No | SSG profile name for OpenSCAP evaluator |
| `catalog` | `string` | No | Catalog identifier for grouping |
| `source` | `string` | No | Upstream source path for provenance tracking |

**Note**: `profile` and `catalog` are metadata that the pack developer provides for documentation, doctor validation, and provider routing. They do not override policy content — the Gemara policy layer is authoritative.

### PackProviderEntry

| Field | Type | Required | Description |
|:---|:---|:---|:---|
| `id` | `string` | Yes | Provider/evaluator identifier |
| `binary` | `string` | Yes | Binary name (e.g., `complyctl-provider-openscap`) |
| `source` | `string` | No | Build source (`build` for compiled, `bundled` for pre-built) |

### SystemDependency

| Field | Type | Required | Description |
|:---|:---|:---|:---|
| `name` | `string` | Yes | Package name |
| `check` | `string` | Yes | Shell command to verify installation |
| `install` | `string` | No | Installation command (for guidance only) |

**Doctor integration**: `doctor` runs each `check` command. Pass = dependency present. Fail = report the missing dependency and suggest `install` command.

---

## State Transitions

### Policy Lifecycle

```text
[Not Cached] ──(complyctl get)──→ [Cached]
[Cached] ──(complyctl get, new digest)──→ [Updated]
[Cached] ──(complyctl get, same digest)──→ [Cached] (no-op)
[Syncing] ──(failure)──→ [Rolled Back to Previous] (atomic)
```

### Generate/Scan Execution

```text
[Configured] ──(complyctl generate)──→ [Graph Resolved, Plugins Prepped, Artifacts Persisted, State Saved, Execution Plan Displayed]
[Configured] ──(complyctl scan, no state)──→ [Auto-Generate] ──→ [Scanning]
[Generated] ──(complyctl scan, fresh digest)──→ [Reuse Artifacts] ──→ [Scanning]
[Generated] ──(complyctl get updates policy)──→ [Stale]
[Stale] ──(complyctl scan)──→ [Warn + Auto-Regenerate] ──→ [Scanning]
[Scanning] ──(plugin responds)──→ [Results Collected]
[Results Collected] ──(format)──→ [Report Written]
[Plugin fails] ──→ [Requirements marked as error, continue with remaining]
[Evaluator ID not found] ──→ [Error: unmatched evaluator, list available plugins]
[scan --dry-run removed Session 2026-02-26e — use complyctl generate for plan preview]
```
