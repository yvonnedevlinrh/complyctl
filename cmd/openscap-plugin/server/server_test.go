// SPDX-License-Identifier: Apache-2.0

package server

import (
	"errors"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/stretchr/testify/assert"

	"github.com/complytime/complyctl/pkg/plugin"
)

func TestMapResultStatus(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedResult plugin.Result
		expectedError  error
	}{
		{"pass", "pass", plugin.ResultPassed, nil},
		{"fixed", "fixed", plugin.ResultPassed, nil},
		{"fail", "fail", plugin.ResultFailed, nil},
		{"error", "error", plugin.ResultError, nil},
		{"unknown", "unknown", plugin.ResultError, nil},
		{"invalid", "invalid", plugin.ResultError, errors.New("couldn't match invalid")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mapResultStatus(tt.input)
			assert.Equal(t, tt.expectedResult, result)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseCheck(t *testing.T) {
	tests := []struct {
		name           string
		xmlContent     string
		expectedResult string
		expectedError  error
	}{
		{
			name:           "Valid/ExpectedFormat",
			xmlContent:     `<check-content-ref name="oval:ssg-audit_perm_change_success:def:1"/>`,
			expectedResult: "audit_perm_change_success",
		},
		{
			name:           "Invalid/UnexpectedFormat",
			xmlContent:     `<check-content-ref name="ovalssg-audit_perm_change_success:def:1"/>`,
			expectedResult: "",
			expectedError:  errors.New("check id \"ovalssg-audit_perm_change_success:def:1\" is in unexpected format"),
		},
		{
			name:           "Invalid/NoNameAttribute",
			xmlContent:     `<check-content-ref/>`,
			expectedResult: "",
			expectedError:  errors.New("check-content-ref node has no 'name' attribute"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := xmlquery.Parse(strings.NewReader(tt.xmlContent))
			assert.NoError(t, err)
			check, err := parseCheck(node.SelectElement("check-content-ref"))
			assert.Equal(t, tt.expectedResult, check)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
