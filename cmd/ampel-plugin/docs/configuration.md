# ampel-plugin Configuration

## Overview

The ampel-plugin reads its target repository configuration directly from `complytime.yaml`. Repositories to scan are defined inline under each target entry using the `repositories` field. This centralizes all configuration in a single file and supports per-repository authentication tokens.

## Configuration Reference

### Target repositories

Each target in `complytime.yaml` can include a `repositories` list:

```yaml
targets:
  - id: <target-id>
    policies:
      - <policy-id>
    repositories:
      - url: <repository-url>          # required, HTTPS URL to GitHub or GitLab
        branches: [<branch>, ...]      # required, list of branches to scan
        specs: [<spec-ref>, ...]       # required for scanning, snappy spec references
        access_token: <token>          # optional, per-repo auth token
```

### Field details

| Field | Required | Description |
|-------|----------|-------------|
| `url` | Yes | HTTPS URL to a GitHub or GitLab repository (e.g., `https://github.com/myorg/repo`) |
| `branches` | Yes | List of branch names to scan. Must be non-empty. |
| `specs` | Yes (for scanning) | List of snappy spec file references. Use `builtin:` prefix for embedded specs. Repositories without specs are skipped during scan. |
| `access_token` | No | Authentication token for this repository. Supports `${VAR}` env var expansion. |

### Granular policy directory

The `ampel_policy_dir` global variable controls where the plugin reads granular AMPEL policy source files:

```yaml
variables:
  ampel_policy_dir: /path/to/custom/policies  # optional
```

- **Default**: `.complytime/ampel/granular-policies/`
- **Generated output**: Always written to `.complytime/ampel/policy/` (unchanged)

## Examples

### GitHub repositories

```yaml
policies:
  - url: registry.example.com/policies/branch-protection@v1.0
    id: branch-protection

targets:
  - id: github-repos
    policies:
      - branch-protection
    repositories:
      - url: https://github.com/myorg/frontend
        branches: [main, develop]
        specs: [builtin:github/branch-rules.yaml]
      - url: https://github.com/myorg/backend
        branches: [main]
        specs: [builtin:github/branch-rules.yaml]
        access_token: ${BACKEND_GITHUB_TOKEN}
```

### GitLab repositories

```yaml
policies:
  - url: registry.example.com/policies/branch-protection@v1.0
    id: branch-protection

targets:
  - id: gitlab-repos
    policies:
      - branch-protection
    repositories:
      - url: https://gitlab.com/myorg/infrastructure
        branches: [main, release]
        specs: [builtin:github/branch-rules.yaml]
        access_token: ${GITLAB_API_TOKEN}
```

### Mixed platforms

A single target can scan repositories across both GitHub and GitLab, each with its own token:

```yaml
policies:
  - url: registry.example.com/policies/branch-protection@v1.0
    id: branch-protection

targets:
  - id: all-repos
    policies:
      - branch-protection
    repositories:
      - url: https://github.com/myorg/frontend
        branches: [main]
        specs: [builtin:github/branch-rules.yaml]
        access_token: ${GITHUB_PAT}
      - url: https://gitlab.com/myorg/infrastructure
        branches: [main, release]
        specs: [builtin:github/branch-rules.yaml]
        access_token: ${GITLAB_API_TOKEN}
```

## Token Authentication

### When `access_token` is set

The token value is expanded from environment variables at config load time (e.g., `${MY_TOKEN}` reads the `MY_TOKEN` env var). During scanning, the plugin detects the platform from the repository URL and injects the token into the snappy subprocess environment:

- `github.com` repositories: sets `GITHUB_TOKEN=<value>`
- `gitlab.com` repositories: sets `GITLAB_TOKEN=<value>`

### When `access_token` is omitted

Snappy inherits the parent process environment unchanged. It reads `GITHUB_TOKEN` or `GITLAB_TOKEN` directly from the environment. This is sufficient when all repositories use the same token.

### Which env vars are expected per platform

| Platform | Environment Variable |
|----------|---------------------|
| GitHub (`github.com`) | `GITHUB_TOKEN` |
| GitLab (`gitlab.com`) | `GITLAB_TOKEN` |

### Security considerations

- Tokens are expanded from environment variables at config load time. Never hardcode tokens directly in `complytime.yaml`.
- The `${VAR}` syntax fails with a clear error if the referenced environment variable is not set.
- Tokens are validated to reject newlines and null bytes (prevents header/env injection).
- Tokens are passed via environment variables to subprocess commands, not as command-line arguments.

## Granular Policy Directory

The `ampel_policy_dir` global variable specifies the directory containing granular AMPEL policy source files (one JSON file per control). This is a workspace-scoped setting shared across all targets.

```yaml
variables:
  ampel_policy_dir: /path/to/custom/policies
```

- **Default location**: `.complytime/ampel/granular-policies/`
- **Generated output location**: `.complytime/ampel/policy/` (not configurable, always separate from source policies)
- **Purpose**: Separates user-authored or tool-generated policy source files from the merged policy bundle produced by the plugin's `generate` phase.
