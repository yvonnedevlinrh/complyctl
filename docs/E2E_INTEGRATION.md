# E2E & Integration Tests

This document provides an overview of the End-to-End (E2E) and integration tests for `complyctl`.

## Overview

The E2E (End-to-End) and integration tests are designed to verify that the `complyctl` commands function correctly
in a real-world environment. The E2E tests, located in [e2e_test.go](../tests/e2e/e2e_test.go), are written in Go
and utilize the testing package. These tests execute the complyctl binary and validate its output through assertions.
The integration tests, on the other hand, are implemented within the GitHub workflow file
[integration_test.yml](../.github/workflow/integration_test.yml).

## How to Run Tests

The test is triggered whenever a PR is created or updated or merged to `main` branch. It begins by starting a Fedora container, then builds `complyctl`, downloads the test data for Fedora OSCAL content, sets up the plugin, and automatically runs the test cases.
If you'd like to use different test data, you can update the input parameters in both [e2e_test.yml](https://github.com/complytime/complyctl/blob/main/.github/workflows/e2e_test.yml#L29) and [integration_test.yml](https://github.com/complytime/complyctl/blob/main/.github/workflows/integration_test.yml#L11)

## Test Cases

The following is a list of the existing E2E test cases:

- **TestComplyctlHelp**: Tests the `complyctl --help` command and ensures that the help message is displayed correctly.
- **TestComplyctlList**: Tests the `complyctl list` command and ensures that the list of frameworks is displayed correctly.
- **TestComplyctlInfo**: Tests the `complyctl info` command with different arguments and ensures that the correct information is displayed.
- **TestComplyctlPlan**: Tests the `complyctl plan` command with different arguments and ensures that the assessment plan is generated correctly.
- **TestComplyctlGenerate**: Tests the `complyctl generate` command and ensures that the policy is generated correctly.
- **TestComplyctlScan**: Tests the `complyctl scan` command and ensures that the scan is performed correctly and the results are generated.
- **TestComplyctlCustomizePlanWorkflow**: Tests a complete workflow of customizing an assessment plan, generating a policy, and scanning the environment.

The following is a list of the existing integration test cases:
- **Running whole workflow:** Run `list`, `plan`, `generate`, `scan` of `complyctl` twice with OSCAL fedora cusp contents, first with original OSCAL assessment-plan, second with customized OSCAL assessment-plan.
- **Validate scan result:** For both original and customized runs, validate rule `xccdf_org.ssgproject.content_rule_file_permissions_etc_passwd` scan result. First it should `fail` (when permissions are 666), after setting permissions to 644, it should `pass`.
- **Validate original assessment-plan** via [go-oscal](https://github.com/defenseunicorns/go-oscal)
- **Validate customized assessment-plan** via [go-oscal](https://github.com/defenseunicorns/go-oscal)

## How to Add New Test Cases

To add a new E2E test case, you can follow these steps:

1. Open the `tests/e2e/e2e_test.go` file.
2. Create a new function with the `Test` prefix (e.g., `TestMyNewCommand`).
3. Use the `os/exec` package to run the `complyctl` command with the desired arguments.
4. Use the `assert` and `require` packages from `testify` to assert the output of the command.
5. Run the tests to ensure that your new test case passes.

To add new test steps for integration test, directly edit the [integration_test.yml](../.github/workflow/integration_test.yml)
