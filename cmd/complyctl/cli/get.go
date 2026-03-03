// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/complytime/complyctl/internal/cache"
	"github.com/complytime/complyctl/internal/complytime"
	"github.com/complytime/complyctl/internal/registry"
)

type getOptions struct {
	*Common
	timeout  time.Duration
	cacheDir string
}

func getCmd(common *Common) *cobra.Command {
	o := &getOptions{
		Common: common,
	}
	cmd := &cobra.Command{
		Use:               "get [flags]",
		Short:             "Fetch new/modified policies from OCI registry and update cache",
		SilenceUsage:      true,
		Example:           "complyctl get",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := o.validate(); err != nil {
				return err
			}
			if err := o.complete(); err != nil {
				return err
			}
			return o.run(cmd.Context())
		},
	}
	cmd.Flags().DurationVarP(&o.timeout, "timeout", "t", complytime.DefaultCommandTimeout, "Maximum time for the get operation (e.g. 5m, 10m, 1h)")
	return cmd
}

func (o *getOptions) validate() error {
	return nil
}

func (o *getOptions) complete() error {
	var err error
	o.cacheDir, err = complytime.ResolveCacheDir()
	if err != nil {
		return fmt.Errorf("failed to resolve cache directory: %w", err)
	}
	return nil
}

func (o *getOptions) run(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	workspace := complytime.NewWorkspace()
	if err := workspace.LoadAndValidate(); err != nil {
		return fmt.Errorf("failed to load workspace config: %w", err)
	}

	cfg := workspace.Config()

	cacheMgr := cache.NewCache(o.cacheDir)

	state, err := cache.LoadState(o.cacheDir)
	if err != nil {
		logger.Error("Cache state load failed", "cache_dir", o.cacheDir, "error", err)
		return fmt.Errorf("failed to load cache state: %w", err)
	}

	credFunc, err := registry.NewCredentialFunc()
	if err != nil {
		logger.Error("Credential resolution failed", "error", err)
		return fmt.Errorf("authentication setup failed: %w", err)
	}

	logger.Info("Starting policy synchronization", "policy_count", len(cfg.Policies))

	total := len(cfg.Policies)
	synced := 0
	for i, entry := range cfg.Policies {
		ref := complytime.ParsePolicyRef(entry.URL)
		version := ref.Version

		client := registry.NewClient(ref.Registry, credFunc)
		source := cache.NewRegistrySource(client)
		sync := cache.NewSync(cacheMgr, state, source)

		if version == "" {
			logger.Info("Resolving latest version", "policy", entry.EffectiveID())
			_, resolvedVersion, resolveErr := client.DefinitionVersion(ctx, ref.Repository)
			if resolveErr != nil {
				logger.Warn("Version resolution failed, falling back to 'latest'",
					"policy", entry.EffectiveID(), "error", resolveErr)
				version = "latest"
			} else {
				version = resolvedVersion
				logger.Info("Resolved version", "policy", entry.EffectiveID(), "version", version)
			}
		}

		fmt.Fprintf(os.Stderr, "Syncing policy %d/%d: %s... ", i+1, total, entry.EffectiveID())
		logger.Info("Syncing policy", "policy", ref.Repository, "version", version)
		if err := sync.SyncPolicy(ctx, ref.Repository, version); err != nil {
			fmt.Fprintln(os.Stderr, "failed")
			suggestMsg := suggestCachedPolicyIDs(o.cacheDir, ref.Repository)
			logger.Error("Policy sync failed", "policy", ref.Repository, "error", err)
			return fmt.Errorf("failed to sync policy %s: %w%s", ref.Repository, err, suggestMsg)
		}
		fmt.Fprintln(os.Stderr, "done")
		synced++
		logger.Info("Policy synced", "policy", entry.EffectiveID())
	}

	logger.Info("Synchronization completed", "synced", synced, "total", total)
	fmt.Fprintln(os.Stderr, "Synchronization completed.")
	return nil
}

func suggestCachedPolicyIDs(cacheDir, failedPolicyID string) string {
	state, err := cache.LoadState(cacheDir)
	if err != nil || len(state.Policies) == 0 {
		return ""
	}
	cached := make([]string, 0, len(state.Policies))
	for id := range state.Policies {
		if id != failedPolicyID {
			cached = append(cached, id)
		}
	}
	if len(cached) == 0 {
		return ""
	}
	return fmt.Sprintf(" (cached policies: %v)", cached)
}
