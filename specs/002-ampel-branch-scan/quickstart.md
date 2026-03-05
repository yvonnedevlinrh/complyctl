# Quickstart: AMPEL Branch Protection Scanning

## Prerequisites

1. complyctl installed and configured
2. AMPEL tools installed via `go install` and on PATH:
   ```bash
   go install github.com/carabiner-dev/snappy@latest
   go install github.com/carabiner-dev/ampel/cmd/ampel@latest
   export PATH=$PATH:$HOME/go/bin
   ```
3. A GitHub and/or GitLab personal access token with read access
   to target repositories:
   ```bash
   # For GitHub repositories
   export GITHUB_TOKEN=ghp_your_token_here
   # For GitLab repositories
   export GITLAB_TOKEN=glpat-your_token_here
   ```

## Setup

### 1. Install the plugin

Copy the plugin binary to the complyctl providers directory with
the required naming convention:

```bash
mkdir -p ~/.complytime/providers
cp complyctl-provider-ampel ~/.complytime/providers/
```

The plugin is discovered automatically by complyctl — no manifest
files or checksums are required.

### 2. Prepare granular AMPEL policies

Place granular AMPEL policy files (one JSON file per control)
in the policy directory. Sample policies are available in the
[complytime-demos](https://github.com/complytime/complytime-demos)
repository under `base_ansible_env/files/ampel-policies/`.

```bash
mkdir -p ~/.complytime/ampel/granular-policies
cp ampel-policies/*.json ~/.complytime/ampel/granular-policies/
```

### 3. Initialize workspace and configure targets

Run `complyctl init` to create the workspace configuration, then
add your policies and targets to `complytime.yaml`:

```bash
complyctl init
```

Edit `complytime.yaml` to add the ampel policy and target entries:

```yaml
targets:
  - id: myorg-myrepo
    policies:
      - branch-protection
    variables:
      url: https://github.com/myorg/myrepo
      specs: builtin:github/branch-rules.yaml
      branches: main
  - id: myorg-another-repo
    policies:
      - branch-protection
    variables:
      url: https://github.com/myorg/another-repo
      specs: builtin:github/branch-rules.yaml
      branches: main,develop
```

Each target requires `url` and `specs` variables. Use `builtin:`
prefix for embedded spec files or absolute paths for custom specs.
Multi-value fields (`specs`, `branches`) use comma-separated strings.

### 4. Generate AMPEL policies

```bash
complyctl generate --policy-id branch-protection
```

This matches assessment requirement IDs against the granular
AMPEL policies and merges the matching policies into a combined
bundle at `{workspace}/ampel/policy/complytime-ampel-policy.json`.

### 5. Scan repositories

```bash
complyctl scan --policy-id branch-protection
```

This scans each configured repository for branch protection
compliance and produces:
- Per-repository result files in `{workspace}/ampel/results/`
- Consolidated `assessment-results.json` in the workspace

## Workspace Structure After Scan

```text
~/.complytime/
├── complytime.yaml
├── ampel/
│   ├── granular-policies/
│   │   ├── BP-01.01-require-pull-request.json
│   │   ├── BP-02.01-minimum-approvals.json
│   │   └── ...
│   ├── policy/
│   │   └── complytime-ampel-policy.json
│   └── results/
│       ├── myorg-myrepo-main.json
│       └── myorg-another-repo-main.json
└── providers/
    └── complyctl-provider-ampel
```

## Custom Policy Location

To use an existing AMPEL policy directory, set the
`ampel_policy_dir` global variable in `complytime.yaml`:

```yaml
global_variables:
  ampel_policy_dir: /path/to/my/ampel/policies
```

## Verify Tool Installation

If the plugin reports missing tools, verify they are on PATH:

```bash
which ampel snappy
```
