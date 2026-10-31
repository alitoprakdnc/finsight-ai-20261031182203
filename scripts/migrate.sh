#!/usr/bin/env bash
#
# scripts/migrate.sh
#
# Runs Goose migrations against the configured PostgreSQL instance.
# Usage:
#   ./scripts/migrate.sh [command] [options]
#
# Commands (default: up):
#   up               Apply all pending migrations
#   down             Roll back the most recent migration
#   status           Show current migration status
#   reset            Roll back all migrations and re‑apply from scratch
#   version <num>    Migrate to a specific version (e.g. ./scripts/migrate.sh version 20240101)
#   help             Show this help message
#
# Environment variables (all required unless DB_URL is set):
#   DB_HOST          PostgreSQL host
#   DB_PORT          PostgreSQL port (default: 5432)
#   DB_USER          PostgreSQL user
#   DB_PASSWORD      PostgreSQL password
#   DB_NAME          PostgreSQL database name
#   DB_SSLMODE       SSL mode (disable|require|verify-ca|verify-full) – default: disable
#   DB_URL           Full connection string (overrides the above variables)
#
# The script expects the `goose` binary to be available in $PATH.
# It will exit with a non‑zero status if any step fails.

set -euo pipefail

# -------------------------- Helper Functions --------------------------

log() {
    local level="$1"
    shift
    printf "[%s] %s\n" "$(date +'%Y-%m-%d %H:%M:%S')" "$*"
}

die() {
    log "ERROR" "$@"
    exit 1
}

show_help() {
    grep '^#' "$0" | cut -c4-
    exit 0
}

# -------------------------- Argument Parsing -------------------------

if [[ "${#@}" -eq 0 ]]; then
    CMD="up"
else
    CMD="${1}"
    shift
fi

# Validate command
case "$CMD" in
    up|down|status|reset|version|help) ;;
    *) die "Unsupported command: $CMD. Use 'help' for usage." ;;
esac

# -------------------------- Environment Loading ---------------------

# Load .env if present (useful for local dev)
if [[ -f ".env" ]]; then
    # shellcheck disable=SC1091
    set -a
    # shellcheck source=/dev/null
    source ".env"
    set +a
fi

# Build connection string
if [[ -n "${DB_URL:-}" ]]; then
    PG_DSN="$DB_URL"
else
    # Required vars
    : "${DB_HOST:?Missing DB_HOST}"
    : "${DB_USER:?Missing DB_USER}"
    : "${DB_PASSWORD:?Missing DB_PASSWORD}"
    : "${DB_NAME:?Missing DB_NAME}"
    DB_PORT="${DB_PORT:-5432}"
    DB_SSLMODE="${DB_SSLMODE:-disable}"

    PG_DSN="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"
fi

# -------------------------- Goose Binary Check ---------------------

if ! command -v goose >/dev/null 2>&1; then
    die "goose binary not found in PATH. Install it (https://github.com/pressly/goose) and ensure it's executable."
fi

# -------------------------- Migration Execution --------------------

MIGRATION_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../migrations" && pwd)"

run_goose() {
    local args=("$@")
    log "INFO" "Running: goose -dir \"$MIGRATION_DIR\" postgres \"$PG_DSN\" ${args[*]}"
    # Use exec to forward signals correctly
    goose -dir "$MIGRATION_DIR" postgres "$PG_DSN" "${args[@]}"
}

case "$CMD" in
    up)
        run_goose up
        ;;
    down)
        run_goose down
        ;;
    status)
        run_goose status
        ;;
    reset)
        # Reset = down all then up
        log "INFO" "Resetting database: rolling back all migrations"
        run_goose down
        log "INFO" "Re‑applying migrations from scratch"
        run_goose up
        ;;
    version)
        if [[ "${#@}" -eq 0 ]]; then
            die "Missing version number. Usage: $0 version <VERSION>"
        fi
        VERSION="${1}"
        shift
        run_goose up-to "$VERSION"
        ;;
    help)
        show_help
        ;;
esac

log "INFO" "Migration command '$CMD' completed successfully."
exit 0