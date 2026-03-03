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
	"github.com/complytime/complyctl/internal/output"
	"github.com/complytime/complyctl/internal/policy"
	"github.com/complytime/complyctl/internal/terminal"
	"github.com/complytime/complyctl/pkg/plugin"
)

type generateOptions struct {
	*Common
	policyID  string
	timeout   time.Duration
	cacheDir  string
	pluginDir string
}

func generateCmd(common *Common) *cobra.Command {
	o := &generateOptions{
		Common: common,
	}
	cmd := &cobra.Command{
		Use:               "generate [flags]",
		Short:             "Generate policy graph and invoke plugins",
		Example:           `complyctl generate --policy-id nist-800-53-r5`,
		SilenceUsage:      true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := o.complete(); err != nil {
				return err
			}
			return o.run(cmd.Context())
		},
	}
	cmd.Flags().StringVarP(&o.policyID, "policy-id", "p", "", "Policy ID to generate (see complyctl list)")
	cmd.Flags().DurationVarP(&o.timeout, "timeout", "t", complytime.DefaultCommandTimeout, "Maximum time for the generate operation (e.g. 5m, 10m, 1h)")
	if err := cmd.MarkFlagRequired("policy-id"); err != nil {
		logger.Error("Failed to mark policy-id as required", "error", err)
	}
	if err := cmd.RegisterFlagCompletionFunc("policy-id", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		logger.Error("Failed to register policy-id completion", "error", err)
	}
	return cmd
}

func (o *generateOptions) complete() error {
	var err error
	o.cacheDir, err = complytime.ResolveCacheDir()
	if err != nil {
		return fmt.Errorf("failed to resolve cache directory: %w", err)
	}
	o.pluginDir, err = complytime.ResolvePluginDir()
	if err != nil {
		return fmt.Errorf("failed to resolve plugin directory: %w", err)
	}
	return nil
}

func (o *generateOptions) run(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	ws := complytime.NewWorkspace()
	if err := ws.LoadAndValidate(); err != nil {
		return fmt.Errorf("failed to load workspace config: %w", err)
	}

	cfg := ws.Config()

	cacheMgr := cache.NewCache(o.cacheDir)
	loader := policy.NewLoader(cacheMgr)
	resolver := policy.NewResolver(loader)

	entry, found := complytime.FindPolicy(cfg.Policies, o.policyID)
	if !found {
		return fmt.Errorf("policy %q not found in config — run complyctl list to see available policy IDs", o.policyID)
	}
	ref := complytime.ParsePolicyRef(entry.URL)

	version, err := loader.ResolveVersion(ref.Repository, ref.Version)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Resolved %s version: %s\n", ref.Repository, version)

	graph, err := resolver.ResolvePolicyGraph(ref.Repository, version)
	if err != nil {
		return fmt.Errorf("failed to resolve policy graph: %w", err)
	}

	mgr, err := plugin.NewManager(o.pluginDir, logger)
	if err != nil {
		return fmt.Errorf("plugin manager init failed: %w", err)
	}
	defer mgr.Cleanup()

	if err := mgr.LoadPlugins(); err != nil {
		return fmt.Errorf("plugin discovery failed: %w", err)
	}

	plugins := mgr.ListPlugins()
	if len(plugins) == 0 {
		return fmt.Errorf("no plugins found in %s (Describe may have failed)", o.pluginDir)
	}

	configs := policy.ExtractAssessmentConfigs(ref.Repository, graph)

	groups := policy.GroupByEvaluator(configs, graph)
	globalVars := cfg.Variables

	eid := entry.EffectiveID()
	var policyTargets []complytime.TargetConfig
	for _, t := range cfg.Targets {
		for _, p := range t.Policies {
			if p == eid {
				policyTargets = append(policyTargets, t)
			}
		}
	}

	spin := terminal.NewSpinner("Generating policy artifacts...")
	spin.Start()

	var evaluatorIDs []string
	var planRows []output.ExecutionPlanRow
	for evalID, group := range groups {
		for _, target := range policyTargets {
			if err := mgr.RouteGenerate(ctx, evalID, globalVars, target.Variables, group.Configs); err != nil {
				spin.Stop()
				return err
			}
		}
		evaluatorIDs = append(evaluatorIDs, evalID)

		status := "healthy"
		if _, lookupErr := mgr.GetPlugin(evalID); lookupErr != nil {
			status = "ERROR"
		}

		for _, target := range policyTargets {
			planRows = append(planRows, output.ExecutionPlanRow{
				TargetID:         target.ID,
				ProviderID:       evalID,
				RequirementCount: len(group.Configs),
				Status:           status,
			})
		}
	}

	spin.Stop()

	cacheState, err := cache.LoadState(o.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to load cache state: %w", err)
	}
	policyState, _ := cacheState.GetPolicyState(ref.Repository)
	genState := policy.NewGenerationState(ref.Repository, policyState.Digest, evaluatorIDs)
	if err := policy.SaveGenerationState(".", ref.Repository, genState); err != nil {
		return fmt.Errorf("failed to save generation state: %w", err)
	}

	fmt.Print(output.FormatExecutionPlan(eid, planRows))
	return nil
}
