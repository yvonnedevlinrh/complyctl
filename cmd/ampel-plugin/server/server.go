package server

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/complytime/complyctl/cmd/ampel-plugin/config"
	"github.com/complytime/complyctl/cmd/ampel-plugin/convert"
	"github.com/complytime/complyctl/cmd/ampel-plugin/results"
	"github.com/complytime/complyctl/cmd/ampel-plugin/scan"
	"github.com/complytime/complyctl/cmd/ampel-plugin/targets"
	"github.com/complytime/complyctl/cmd/ampel-plugin/toolcheck"
	"github.com/complytime/complyctl/pkg/plugin"
)

// ScanRunner is used by Scan to execute scan commands.
// It defaults to scan.ExecRunner{} and can be overridden for testing.
var ScanRunner scan.CommandRunner = scan.ExecRunner{}

// SkipToolCheck disables tool presence validation. Used in tests.
var SkipToolCheck bool

var _ plugin.Plugin = (*PluginServer)(nil)

// PluginServer implements the plugin.Plugin interface for the AMPEL plugin.
type PluginServer struct{}

// New returns a new PluginServer.
func New() *PluginServer {
	return &PluginServer{}
}

// Describe returns the plugin metadata and health status.
func (s *PluginServer) Describe(_ context.Context, _ *plugin.DescribeRequest) (*plugin.DescribeResponse, error) {
	return &plugin.DescribeResponse{
		Healthy:                 true,
		Version:                 "0.1.0",
		RequiredTargetVariables: []string{"github_token"},
	}, nil
}

// Generate matches requirement IDs from the assessment configurations against
// granular AMPEL policy files and merges the matched policies into a single
// bundle for scan.
func (s *PluginServer) Generate(_ context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	logger := hclog.Default()

	if len(req.Configuration) == 0 {
		return &plugin.GenerateResponse{
			Success:      false,
			ErrorMessage: "no assessment configurations provided",
		}, nil
	}

	if err := checkRequiredTools(logger); err != nil {
		return &plugin.GenerateResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	if err := config.EnsureDirectories(); err != nil {
		return &plugin.GenerateResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("directory setup failed: %v", err),
		}, nil
	}

	logger.Info("generating AMPEL policy")

	sourceDir := config.PolicyDirPath()
	outputDir := config.GeneratedPolicyDirPath()

	granular, err := convert.LoadGranularPolicies(sourceDir)
	if err != nil {
		return &plugin.GenerateResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("loading granular policies: %v", err),
		}, nil
	}

	matched, warnings := convert.MatchPolicies(req.Configuration, granular)
	for _, w := range warnings {
		logger.Warn(w)
	}

	if len(matched) == 0 {
		logger.Info("no matching policies found, skipping policy generation")
		return &plugin.GenerateResponse{Success: true}, nil
	}

	bundle := convert.MergeToBundle(matched)
	if err := convert.WritePolicy(bundle, outputDir); err != nil {
		return &plugin.GenerateResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("writing AMPEL policy bundle: %v", err),
		}, nil
	}

	logger.Info("AMPEL policy bundle written", "path", outputDir, "policies", len(matched))
	return &plugin.GenerateResponse{Success: true}, nil
}

// Scan invokes the AMPEL toolchain to scan target repositories and returns
// standardized assessment results.
func (s *PluginServer) Scan(_ context.Context, req *plugin.ScanRequest) (*plugin.ScanResponse, error) {
	logger := hclog.Default()

	if len(req.Targets) == 0 {
		return nil, fmt.Errorf("no targets provided")
	}

	if err := checkRequiredTools(logger); err != nil {
		return nil, err
	}

	if err := config.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("directory setup failed: %w", err)
	}

	logger.Info("scanning target repositories")

	targetsPath := config.TargetsFilePath()
	targetConfig, err := targets.LoadTargetsByID(targetsPath)
	if err != nil {
		return nil, fmt.Errorf("loading targets: %w", err)
	}

	generatedDir := config.GeneratedPolicyDirPath()
	resultsDir := config.ResultsDirPath()
	specDir := config.SpecDirPath()

	if err := scan.WriteSpecFiles(specDir); err != nil {
		return nil, fmt.Errorf("writing spec files: %w", err)
	}

	scanCfg := scan.ScanConfig{
		PolicyPath: filepath.Join(generatedDir, convert.PolicyFileName),
		OutputDir:  resultsDir,
		SpecDir:    specDir,
	}

	var repoResults []*results.PerRepoResult

	for _, target := range req.Targets {
		entry, ok := targetConfig.Targets[target.TargetID]
		if !ok {
			availableIDs := make([]string, 0, len(targetConfig.Targets))
			for id := range targetConfig.Targets {
				availableIDs = append(availableIDs, id)
			}
			return nil, fmt.Errorf("unknown target ID %q, available targets: %s",
				target.TargetID, strings.Join(availableIDs, ", "))
		}

		for _, repo := range entry.Repositories {
			if len(repo.Specs) == 0 {
				logger.Warn("skipping repository with no specs defined", "url", repo.URL)
				continue
			}

			for _, branch := range repo.Branches {
				for _, specRef := range repo.Specs {
					specPath := scan.ResolveSpecPath(specRef, scanCfg.SpecDir)
					logger.Info("scanning repository", "url", repo.URL, "branch", branch, "spec", specRef)

					rawResult, err := scan.ScanRepository(repo, branch, specPath, scanCfg, ScanRunner)
					if err != nil {
						logger.Error("scan failed", "repo", repo.URL, "branch", branch, "spec", specRef, "error", err)
						errResult := &results.PerRepoResult{
							Repository: repo.URL,
							Branch:     branch,
							Status:     "error",
							Error:      err.Error(),
						}
						repoResults = append(repoResults, errResult)
						if writeErr := results.WritePerRepoResult(errResult, resultsDir); writeErr != nil {
							logger.Error("failed to write error result", "error", writeErr)
						}
						continue
					}

					parsed, err := results.ParseAmpelOutput(rawResult.Output, repo.URL, branch)
					if err != nil {
						logger.Error("failed to parse scan output", "repo", repo.URL, "error", err)
						errResult := &results.PerRepoResult{
							Repository: repo.URL,
							Branch:     branch,
							Status:     "error",
							Error:      err.Error(),
						}
						repoResults = append(repoResults, errResult)
						if writeErr := results.WritePerRepoResult(errResult, resultsDir); writeErr != nil {
							logger.Error("failed to write error result", "error", writeErr)
						}
						continue
					}

					repoResults = append(repoResults, parsed)
				}
			}
		}
	}

	scanResponse := results.ToScanResponse(repoResults)
	logger.Info("scan complete", "repositories_scanned", len(repoResults))
	return scanResponse, nil
}

// checkRequiredTools validates that all required AMPEL tools are on PATH.
func checkRequiredTools(logger hclog.Logger) error {
	if SkipToolCheck {
		return nil
	}
	missing, err := toolcheck.CheckTools()
	if err != nil {
		return fmt.Errorf("checking required tools: %w", err)
	}
	if len(missing) > 0 {
		logger.Warn("required tools missing", "tools", missing)
		return toolcheck.FormatMissingToolsError(missing)
	}
	return nil
}
