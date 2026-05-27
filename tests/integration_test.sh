#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Integration test for the Gemara-native complyctl workflow.
# Exercises: get → list → generate → scan (oscal, pretty, sarif)
#
# Run locally:  make test-integration
# Run directly: ./tests/integration_test.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="${REPO_ROOT}/bin/complyctl"
MOCK_REGISTRY="${REPO_ROOT}/bin/mock-oci-registry"
TEST_PLUGIN="${REPO_ROOT}/bin/complyctl-provider-test"

POLICY_ID="policies/test-branch-protection"
REGISTRY_PORT="${GEMARA_SERVICE_PORT:-8765}"
REGISTRY_URL="http://localhost:${REGISTRY_PORT}"

WORK_DIR=""
TEST_HOME=""
REGISTRY_PID=""
PASSED=0
FAILED=0

cleanup() {
    if [[ -n "${REGISTRY_PID}" ]]; then
        kill "${REGISTRY_PID}" 2>/dev/null || true
        wait "${REGISTRY_PID}" 2>/dev/null || true
    fi
    [[ -n "${WORK_DIR}" ]] && rm -rf "${WORK_DIR}"
    [[ -n "${TEST_HOME}" ]] && rm -rf "${TEST_HOME}"
}
trap cleanup EXIT

# --- Assertions ---

pass() {
    PASSED=$((PASSED + 1))
    echo "  PASS: $1" >&2
}

fail() {
    FAILED=$((FAILED + 1))
    echo "  FAIL: $1" >&2
}

assert_contains() {
    local label="$1" haystack="$2" needle="$3"
    if echo "${haystack}" | grep -q "${needle}"; then
        pass "${label}"
    else
        fail "${label}: expected output to contain '${needle}'"
        echo "  --- actual output ---" >&2
        echo "${haystack}" | head -20 >&2
        echo "  ---" >&2
    fi
}

# Prints the matching file path to stdout for capture.
# Status messages go to stderr.
assert_file_exists() {
    local label="$1" pattern="$2"
    local match dir base
    dir="$(dirname "${pattern}")"
    base="$(basename "${pattern}")"
    match=$(find "${dir}" -maxdepth 1 -name "${base}" -print -quit 2>/dev/null) || true
    if [[ -n "${match}" && -s "${match}" ]]; then
        pass "${label}"
        echo "${match}"
    else
        fail "${label}: no non-empty file matching ${pattern}"
        echo ""
    fi
}

assert_json_eq() {
    local label="$1" file="$2" query="$3" expected="$4"
    local actual
    actual=$(jq -r "${query}" "${file}" 2>/dev/null) || true
    if [[ "${actual}" == "${expected}" ]]; then
        pass "${label}"
    else
        fail "${label}: expected '${expected}', got '${actual}'"
    fi
}

assert_json_gte() {
    local label="$1" file="$2" query="$3" min="$4"
    local actual
    actual=$(jq -r "${query}" "${file}" 2>/dev/null) || true
    if [[ "${actual}" -ge "${min}" ]] 2>/dev/null; then
        pass "${label}"
    else
        fail "${label}: expected >= ${min}, got '${actual}'"
    fi
}

# --- Prerequisites ---

echo "=== Prerequisites ==="
for bin in "${BINARY}" "${MOCK_REGISTRY}" "${TEST_PLUGIN}"; do
    if [[ ! -x "${bin}" ]]; then
        echo "FATAL: ${bin} not found. Run 'make build build-test-plugin' first."
        exit 1
    fi
done

for cmd in jq curl; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
        echo "FATAL: '${cmd}' is required but not installed."
        exit 1
    fi
done
echo "  All prerequisites met."

# --- Setup ---

echo "=== Setup ==="
TEST_HOME="$(mktemp -d)"
WORK_DIR="$(mktemp -d)"
export HOME="${TEST_HOME}"

mkdir -p "${TEST_HOME}/.complytime/providers"
cp "${TEST_PLUGIN}" "${TEST_HOME}/.complytime/providers/complyctl-provider-ampel"
echo "  HOME=${TEST_HOME}"
echo "  WORK=${WORK_DIR}"

GEMARA_SERVICE_PORT="${REGISTRY_PORT}" "${MOCK_REGISTRY}" &
REGISTRY_PID=$!

for _ in $(seq 1 15); do
    if curl -sf "${REGISTRY_URL}/v2/" >/dev/null 2>&1; then
        break
    fi
    sleep 0.5
done
if ! curl -sf "${REGISTRY_URL}/v2/" >/dev/null 2>&1; then
    echo "FATAL: mock registry did not start on ${REGISTRY_URL}"
    exit 1
fi
echo "  Mock registry ready (PID ${REGISTRY_PID})"

cat > "${WORK_DIR}/complytime.yaml" <<EOF
policies:
  - url: ${REGISTRY_URL}/${POLICY_ID}
    id: ${POLICY_ID}
variables:
  workspace: /tmp/integration-workspace
targets:
  - id: integration-target
    policies:
      - ${POLICY_ID}
    variables:
      env: test
EOF
echo "  Workspace config written."

run_complyctl() {
    (cd "${WORK_DIR}" && "${BINARY}" "$@" 2>&1)
}

# --- Tests ---

echo ""
echo "=== get ==="
OUT="$(run_complyctl get)"
echo "${OUT}"
assert_contains "get: sync completed" "${OUT}" "Synchronization completed."
assert_file_exists "get: oci-layout exists" \
    "${TEST_HOME}/.complytime/policies/policies/test-branch-protection/oci-layout" >/dev/null
assert_file_exists "get: state.json exists" \
    "${TEST_HOME}/.complytime/state.json" >/dev/null

echo ""
echo "=== list ==="
OUT="$(run_complyctl list)"
echo "${OUT}"
assert_contains "list: shows policy" "${OUT}" "test-branch-protection"

echo ""
echo "=== generate ==="
OUT="$(run_complyctl generate --policy-id "${POLICY_ID}")"
echo "${OUT}"
assert_contains "generate: completed" "${OUT}" "Generation completed."

echo ""
echo "=== scan --format oscal ==="
rm -rf "${WORK_DIR}/.complytime/scan"
OUT="$(run_complyctl scan --policy-id "${POLICY_ID}" --format oscal)"
echo "${OUT}"
assert_contains "scan oscal: completed" "${OUT}" "requirements:"

assert_file_exists "scan oscal: evaluation-log exists" \
    "${WORK_DIR}/.complytime/scan/evaluation-log-*.yaml" >/dev/null

OSCAL_FILE="$(assert_file_exists "scan oscal: assessment-results exists" \
    "${WORK_DIR}/assessment-results-*.json")"

if [[ -n "${OSCAL_FILE}" ]]; then
    assert_json_eq "scan oscal: oscal-version is 1.1.3" \
        "${OSCAL_FILE}" '.["assessment-results"].metadata["oscal-version"]' "1.1.3"
    assert_json_gte "scan oscal: at least 1 result entry" \
        "${OSCAL_FILE}" '.["assessment-results"].results | length' 1
fi

echo ""
echo "=== scan --format pretty ==="
rm -rf "${WORK_DIR}/.complytime/scan"
OUT="$(run_complyctl scan --policy-id "${POLICY_ID}" --format pretty)"
echo "${OUT}"
assert_contains "scan pretty: completed" "${OUT}" "requirements:"

MD_FILE="$(assert_file_exists "scan pretty: markdown report exists" \
    "${WORK_DIR}/report-*.md")"

if [[ -n "${MD_FILE}" ]]; then
    MD_CONTENT="$(cat "${MD_FILE}")"
    assert_contains "scan pretty: has report header" "${MD_CONTENT}" "Compliance Scan Report"
fi

echo ""
echo "=== scan --format sarif ==="
rm -rf "${WORK_DIR}/.complytime/scan"
OUT="$(run_complyctl scan --policy-id "${POLICY_ID}" --format sarif)"
echo "${OUT}"
assert_contains "scan sarif: completed" "${OUT}" "requirements:"

SARIF_FILE="$(assert_file_exists "scan sarif: sarif file exists" \
    "${WORK_DIR}/scan-*.sarif.json")"

if [[ -n "${SARIF_FILE}" ]]; then
    assert_json_eq "scan sarif: version is 2.1.0" \
        "${SARIF_FILE}" '.version' "2.1.0"
fi

echo ""
echo "=== scan (default, no --format) ==="
rm -rf "${WORK_DIR}/.complytime/scan"
rm -f "${WORK_DIR}"/assessment-results-* "${WORK_DIR}"/report-* "${WORK_DIR}"/scan-*
OUT="$(run_complyctl scan --policy-id "${POLICY_ID}")"
echo "${OUT}"
assert_contains "scan default: completed" "${OUT}" "requirements:"
assert_file_exists "scan default: evaluation-log exists" \
    "${WORK_DIR}/.complytime/scan/evaluation-log-*.yaml" >/dev/null

NO_OSCAL=$(ls "${WORK_DIR}/assessment-results-"* 2>/dev/null || true)
NO_MD=$(ls "${WORK_DIR}/report-"* 2>/dev/null || true)
NO_SARIF=$(ls "${WORK_DIR}/scan-"* 2>/dev/null || true)
if [[ -z "${NO_OSCAL}" && -z "${NO_MD}" && -z "${NO_SARIF}" ]]; then
    pass "scan default: no formatted output without --format"
else
    fail "scan default: formatted output found without --format"
fi

# --- Summary ---

echo ""
echo "==============================="
echo "  Passed: ${PASSED}"
echo "  Failed: ${FAILED}"
echo "==============================="

if [[ "${FAILED}" -gt 0 ]]; then
    exit 1
fi
