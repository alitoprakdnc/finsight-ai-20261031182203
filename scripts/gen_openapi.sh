#!/usr/bin/env bash
#
# scripts/gen_openapi.sh
#
# Generates Go server stubs from the OpenAPI specification.
# The generated code lives in internal/api/api_gen.go and is
# formatted with gofmt. The script is idempotent and safe to
# run in CI pipelines.
#
# Requirements:
#   - oapi-codegen (installed via go install if missing)
#   - gofmt (part of the Go toolchain)
#
# Usage:
#   ./scripts/gen_openapi.sh
#
# The script exits with a non‑zero status on any failure.

set -euo pipefail

# ----------------------------------------------------------------------
# Helper functions
# ----------------------------------------------------------------------
log() {
  printf '[%s] %s\n' "$(date +"%Y-%m-%d %H:%M:%S")" "$*"
}

die() {
  log "ERROR: $*"
  exit 1
}

# ----------------------------------------------------------------------
# Resolve paths
# ----------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

OPENAPI_SPEC="${ROOT_DIR}/pkg/openapi/openapi.yaml"
OUTPUT_DIR="${ROOT_DIR}/internal/api"
OUTPUT_FILE="${OUTPUT_DIR}/api_gen.go"

# ----------------------------------------------------------------------
# Preconditions
# ----------------------------------------------------------------------
[[ -f "${OPENAPI_SPEC}" ]] || die "OpenAPI spec not found at ${OPENAPI_SPEC}"

# Ensure output directory exists
mkdir -p "${OUTPUT_DIR}"

# ----------------------------------------------------------------------
# Ensure oapi-codegen is available
# ----------------------------------------------------------------------
if ! command -v oapi-codegen >/dev/null 2>&1; then
  log "oapi-codegen not found, installing via go install..."
  # Pin to a known stable version; adjust as needed.
  GO111MODULE=on go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.13.0 ||
    die "Failed to install oapi-codegen"
fi

# ----------------------------------------------------------------------
# Generate the Go stubs
# ----------------------------------------------------------------------
log "Generating Go server stubs from ${OPENAPI_SPEC}"
# -generate types,server creates both the data structures and the Gin‑compatible handler interface.
# -package api matches the internal/api package where the rest of the code lives.
oapi-codegen -generate types,server -package api -o "${OUTPUT_FILE}" "${OPENAPI_SPEC}" ||
  die "oapi-codegen failed"

# ----------------------------------------------------------------------
# Run gofmt (and goimports if available) to keep the code tidy
# ----------------------------------------------------------------------
if command -v gofmt >/dev/null 2>&1; then
  log "Running gofmt on generated file"
  gofmt -w "${OUTPUT_FILE}"
fi

if command -v goimports >/dev/null 2>&1; then
  log "Running goimports on generated file"
  goimports -w "${OUTPUT_FILE}"
fi

log "OpenAPI generation completed successfully"
log "Generated file: ${OUTPUT_FILE}"
exit 0