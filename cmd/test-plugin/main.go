// SPDX-License-Identifier: Apache-2.0

// test-plugin is a standalone gRPC plugin for E2E testing.
// It implements the plugin.Plugin interface with predefined responses
// and registers via plugin.Serve().
//
// Build:
//
//	go build -o complyctl-provider-test ./cmd/test-plugin
//
// The resulting binary follows the complyctl-provider-* naming convention
// so the plugin discovery system finds it automatically when placed in
// the providers directory (~/.complytime/providers/ or a test-specific path).
//
// This binary is NOT referenced by any production code.
package main

import (
	"context"

	"github.com/complytime/complyctl/pkg/plugin"
)

// Compile-time check
var _ plugin.Plugin = (*testEvaluator)(nil)

// testEvaluator returns predefined responses for all RPCs.
type testEvaluator struct {
	requirementIDs []string
}

func (t *testEvaluator) Describe(_ context.Context, _ *plugin.DescribeRequest) (*plugin.DescribeResponse, error) {
	return &plugin.DescribeResponse{
		Healthy:                 true,
		Version:                 "test-v0.1.0",
		RequiredGlobalVariables: []string{"workspace"},
	}, nil
}

func (t *testEvaluator) Generate(_ context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	t.requirementIDs = make([]string, 0, len(req.Configuration))
	for _, cfg := range req.Configuration {
		t.requirementIDs = append(t.requirementIDs, cfg.RequirementID)
	}
	return &plugin.GenerateResponse{Success: true}, nil
}

func (t *testEvaluator) Scan(_ context.Context, req *plugin.ScanRequest) (*plugin.ScanResponse, error) {
	assessments := make([]plugin.AssessmentLog, 0, len(t.requirementIDs))
	for _, reqID := range t.requirementIDs {
		assessments = append(assessments, plugin.AssessmentLog{
			RequirementID: reqID,
			Steps: []plugin.Step{
				{
					Name:    "test-check",
					Result:  plugin.ResultPassed,
					Message: "predefined pass from test-plugin",
				},
			},
			Message:    "all checks passed",
			Confidence: plugin.ConfidenceLevelHigh,
		})
	}
	return &plugin.ScanResponse{Assessments: assessments}, nil
}

func (t *testEvaluator) Export(_ context.Context, _ *plugin.ExportRequest) (*plugin.ExportResponse, error) {
	return &plugin.ExportResponse{
		Success:      false,
		ErrorMessage: "test plugin does not support export",
	}, nil
}

func main() {
	plugin.Serve(&testEvaluator{})
}
