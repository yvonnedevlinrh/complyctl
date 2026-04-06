// SPDX-License-Identifier: Apache-2.0

package plugin_test

import (
	"context"

	"github.com/complytime/complyctl/pkg/plugin"
)

// Compile-time check: mockClient must implement Plugin
var _ plugin.Plugin = (*mockClient)(nil)

// mockClient provides an in-memory mock plugin.Plugin for testing only.
// Like real providers, it stores requirement IDs during Generate and uses
// them during Scan (R47).
type mockClient struct {
	requirementIDs []string
}

// newMockClient creates a new mockClient instance for tests.
func newMockClient() *mockClient {
	return &mockClient{}
}

func (m *mockClient) Describe(_ context.Context, _ *plugin.DescribeRequest) (*plugin.DescribeResponse, error) {
	return &plugin.DescribeResponse{
		Healthy: true,
		Version: "mock-v1",
	}, nil
}

func (m *mockClient) Generate(_ context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	m.requirementIDs = make([]string, 0, len(req.Configuration))
	for _, cfg := range req.Configuration {
		m.requirementIDs = append(m.requirementIDs, cfg.RequirementID)
	}
	return &plugin.GenerateResponse{Success: true}, nil
}

func (m *mockClient) Scan(_ context.Context, _ *plugin.ScanRequest) (*plugin.ScanResponse, error) {
	assessments := make([]plugin.AssessmentLog, 0, len(m.requirementIDs))
	for _, reqID := range m.requirementIDs {
		assessments = append(assessments, plugin.AssessmentLog{
			RequirementID: reqID,
			Steps: []plugin.Step{
				{Name: "mock-check", Result: plugin.ResultPassed, Message: "mock check passed"},
			},
			Message:    "mock passed",
			Confidence: plugin.ConfidenceLevelHigh,
		})
	}
	return &plugin.ScanResponse{Assessments: assessments}, nil
}
