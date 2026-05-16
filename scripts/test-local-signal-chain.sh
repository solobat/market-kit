#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
MARKET_KIT_DIR="${MARKET_KIT_DIR:-$ROOT_DIR/market-kit}"
VERIDEX_DIR="${VERIDEX_DIR:-$ROOT_DIR/veridex/backend}"
TRADFI_MONITOR_DIR="${TRADFI_MONITOR_DIR:-$ROOT_DIR/tradfi-monitor}"
MARKET_WATCH_DIR="${MARKET_WATCH_DIR:-$ROOT_DIR/market-platform/derived/market-watch}"

run_test() {
  local label="$1"
  local workdir="$2"
  local cmd="$3"

  printf '\n==> %s\n' "$label"
  (
    cd "$workdir"
    eval "$cmd"
  )
}

run_test "market-kit identity + shared signal fixtures" "$MARKET_KIT_DIR" "go test ./identity/... ./signaltest/..."
run_test "tradfi-monitor veridex notifier chain" "$TRADFI_MONITOR_DIR" "go test ./internal/core/..."
run_test "market-watch veridex notifier chain" "$MARKET_WATCH_DIR" "go test ./internal/notifier/..."
run_test "veridex webhook ingestion replay" "$VERIDEX_DIR" "go test ./internal/api/..."

printf '\nLocal signal chain tests passed.\n'
