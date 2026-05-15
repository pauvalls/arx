#!/bin/sh
# arx GitHub Action entrypoint
# GitHub Actions passes input variables as INPUT_<UPPERCASED_NAME> env vars.
# e.g. input "path" → env var INPUT_PATH, input "config" → INPUT_CONFIG
# ARX_BIN is set by action.yml and points to the built binary.

set -e

# Resolve the arx binary path
ARX="${ARX_BIN:-arx}"

# --- resolve inputs ---
PROJECT_PATH="${INPUT_PATH:-.}"
CONFIG_PATH="${INPUT_CONFIG:-arx.yaml}"
FORMAT="${INPUT_FORMAT:-sarif}"
BASELINE="${INPUT_BASELINE:-}"
DIAGRAM="${INPUT_DIAGRAM:-false}"

echo "::group::Arx Architecture Audit"
echo "  Path:     ${PROJECT_PATH}"
echo "  Config:   ${CONFIG_PATH}"
echo "  Format:   ${FORMAT}"
echo "  Baseline: ${BASELINE:-<none>}"
echo "  Diagram:  ${DIAGRAM}"
echo ""

# Note: --ci overrides --format to JSON in the current CLI.
# For SARIF output we must NOT pass --ci, since the flag forces JSON.
if [ "${FORMAT}" = "sarif" ]; then
    # SARIF output: redirect to file for upload-sarif action.
    # Use --format sarif (without --ci) to get actual SARIF JSON.
    echo "Running: ${ARX} check --format sarif --config ${CONFIG_PATH} ${PROJECT_PATH}"
    "${ARX}" check --format sarif --config "${CONFIG_PATH}" "${PROJECT_PATH}" > arx-audit.sarif
    EXIT_CODE=$?
    echo "SARIF output written to arx-audit.sarif"
else
    # Other formats: use --ci for CI-friendly exit codes
    echo "Running: ${ARX} check --ci --format ${FORMAT} --config ${CONFIG_PATH} ${PROJECT_PATH}"
    "${ARX}" check --ci --format "${FORMAT}" --config "${CONFIG_PATH}" "${PROJECT_PATH}"
    EXIT_CODE=$?
fi

echo "::endgroup::"

# Generate architecture diagram if requested (runs even on audit failure)
if [ "${DIAGRAM}" = "true" ]; then
    echo "::group::Generating architecture diagram"
    echo "Running: ${ARX} diagram --format mermaid ${PROJECT_PATH}"
    "${ARX}" diagram --format mermaid "${PROJECT_PATH}" > arx-architecture.mmd 2>/dev/null || true
    if [ -f arx-architecture.mmd ] && [ -s arx-architecture.mmd ]; then
        echo "Diagram saved to arx-architecture.mmd"
    else
        echo "Warning: diagram generation produced no output"
    fi
    echo "::endgroup::"
fi

# Exit with arx's exit code so workflow steps respect the result
exit ${EXIT_CODE}
