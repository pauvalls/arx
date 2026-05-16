#!/bin/sh
# arx GitHub Action entrypoint
# GitHub Actions passes input variables as INPUT_<UPPERCASED_NAME> env vars.
# e.g. input "path" → env var INPUT_PATH, input "config" → INPUT_CONFIG
# ARX_BIN is set by action.yml and points to the built binary.

# Do NOT use set -e: we need to capture arx's exit code without aborting.

# Resolve the arx binary path
ARX="${ARX_BIN:-arx}"

# Output directory for generated files
OUTPUT_DIR="${GITHUB_WORKSPACE:-.}"

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

if [ "${FORMAT}" = "sarif" ]; then
    # SARIF output: redirect to file for upload-sarif action.
    # Always succeed — SARIF file contains violations data for upload.
    # arx exits 1 on violations; we DON'T want to fail the action here.
    echo "Running: ${ARX} check --format sarif --config ${CONFIG_PATH} ${PROJECT_PATH}"
    "${ARX}" check --format sarif --config "${CONFIG_PATH}" "${PROJECT_PATH}" > "${OUTPUT_DIR}/arx-audit.sarif" || true
    echo "SARIF output written to ${OUTPUT_DIR}/arx-audit.sarif"
    EXIT_CODE=0
else
    # Other formats: use --ci for CI-friendly exit codes, propagate result
    echo "Running: ${ARX} check --ci --format ${FORMAT} --config ${CONFIG_PATH} ${PROJECT_PATH}"
    "${ARX}" check --ci --format "${FORMAT}" --config "${CONFIG_PATH}" "${PROJECT_PATH}"
    EXIT_CODE=$?
fi

echo "::endgroup::"

# Generate architecture diagram if requested (runs even on audit failure)
if [ "${DIAGRAM}" = "true" ]; then
    echo "::group::Generating architecture diagram"
    echo "Running: ${ARX} diagram --format mermaid ${PROJECT_PATH}"
    "${ARX}" diagram --format mermaid "${PROJECT_PATH}" > "${OUTPUT_DIR}/arx-architecture.mmd" 2>/dev/null || true
    if [ -f "${OUTPUT_DIR}/arx-architecture.mmd" ] && [ -s "${OUTPUT_DIR}/arx-architecture.mmd" ]; then
        echo "Diagram saved to arx-architecture.mmd"
    else
        echo "Warning: diagram generation produced no output"
    fi
    echo "::endgroup::"
fi

# For SARIF, always exit 0 (file upload handles reporting).
# For other formats, propagate arx's exit code for CI pipelines.
exit ${EXIT_CODE}
