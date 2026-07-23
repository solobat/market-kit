#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

runtime_registry_path="${MARKET_KIT_RUNTIME_REGISTRY_PATH:-data/runtime_generated_registry.json}"
review_output_path="${MARKET_KIT_RUNTIME_REGISTRY_REVIEW_PATH:-data/runtime_generated_registry.review.md}"
bootstrap_sources="${MARKET_KIT_BOOTSTRAP_SOURCES:-binance,binance-web3,bybit,okx,bitget,gate,hyperliquid}"
bootstrap_timeout="${MARKET_KIT_BOOTSTRAP_TIMEOUT:-45s}"

mkdir -p "$(dirname "${runtime_registry_path}")"
if [[ -n "${review_output_path}" ]]; then
  mkdir -p "$(dirname "${review_output_path}")"
fi

go run ./cmd/market-kit-curate-slipstream \
  --bootstrap \
  --source-name market-kit-bootstrap \
  --sources "${bootstrap_sources}" \
  --timeout "${bootstrap_timeout}" \
  --output "${runtime_registry_path}" \
  --review-output "${review_output_path}"

echo "bootstrapped runtime registry: ${runtime_registry_path}"
if [[ -n "${review_output_path}" ]]; then
  echo "wrote review report: ${review_output_path}"
fi
