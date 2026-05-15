#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_PATH="${1:-${ROOT_DIR}/.github/downstream-go-consumers.json}"
MARKET_KIT_MODULE="${MARKET_KIT_MODULE:-github.com/solobat/market-kit}"
MARKET_KIT_TAG="${MARKET_KIT_TAG:-}"
GH_TOKEN="${GH_TOKEN:-${GITHUB_TOKEN:-}}"
BRANCH_SUFFIX="${MARKET_KIT_TAG//./-}"

if [[ -z "${MARKET_KIT_TAG}" ]]; then
  echo "MARKET_KIT_TAG is required" >&2
  exit 1
fi

if [[ -z "${GH_TOKEN}" ]]; then
  echo "GH_TOKEN or GITHUB_TOKEN is required" >&2
  exit 1
fi

if [[ ! -f "${CONFIG_PATH}" ]]; then
  echo "downstream config not found: ${CONFIG_PATH}" >&2
  exit 1
fi

git config --global user.name "github-actions[bot]"
git config --global user.email "41898282+github-actions[bot]@users.noreply.github.com"

pr_body() {
  local repo="$1"
  cat <<EOF
This PR was opened automatically after \`${MARKET_KIT_MODULE}\` released \`${MARKET_KIT_TAG}\`.

- updates \`${MARKET_KIT_MODULE}\` to \`${MARKET_KIT_TAG}\`
- runs \`go mod tidy\`

Triggered from [${GITHUB_REPOSITORY:-market-kit}@${MARKET_KIT_TAG}](https://github.com/${GITHUB_REPOSITORY:-solobat/market-kit}/releases/tag/${MARKET_KIT_TAG}).

Please run the normal verification flow for \`${repo}\` before merging.
EOF
}

while IFS= read -r item; do
  repo="$(jq -r '.repo' <<<"${item}")"
  base_branch="$(jq -r '.branch // "main"' <<<"${item}")"
  module_dir="$(jq -r '.module_dir // "."' <<<"${item}")"
  branch_name="codex/market-kit-${BRANCH_SUFFIX}"
  worktree="$(mktemp -d)"
  repo_dir="${worktree}/repo"

  echo "---"
  echo "Updating ${repo} (${module_dir}) to ${MARKET_KIT_TAG}"

  git clone --depth 1 --branch "${base_branch}" "https://x-access-token:${GH_TOKEN}@github.com/${repo}.git" "${repo_dir}"

  pushd "${repo_dir}/${module_dir}" >/dev/null
  current_version="$(go list -m -f '{{.Version}}' "${MARKET_KIT_MODULE}" 2>/dev/null || true)"
  if [[ "${current_version}" == "${MARKET_KIT_TAG}" ]]; then
    echo "${repo} already uses ${MARKET_KIT_TAG}"
    popd >/dev/null
    rm -rf "${worktree}"
    continue
  fi

  GOTOOLCHAIN=auto go get "${MARKET_KIT_MODULE}@${MARKET_KIT_TAG}"
  GOTOOLCHAIN=auto go mod tidy
  popd >/dev/null

  if git -C "${repo_dir}" diff --quiet -- "${module_dir}/go.mod" "${module_dir}/go.sum"; then
    echo "No dependency diff for ${repo}"
    rm -rf "${worktree}"
    continue
  fi

  git -C "${repo_dir}" checkout -b "${branch_name}"
  git -C "${repo_dir}" add "${module_dir}/go.mod" "${module_dir}/go.sum"
  git -C "${repo_dir}" commit -m "chore(deps): bump market-kit to ${MARKET_KIT_TAG}"
  git -C "${repo_dir}" push --force-with-lease origin "${branch_name}"

  existing_pr="$(gh pr list --repo "${repo}" --head "${branch_name}" --json url --jq '.[0].url // ""')"
  if [[ -n "${existing_pr}" ]]; then
    echo "PR already exists for ${repo}: ${existing_pr}"
  else
    gh pr create \
      --repo "${repo}" \
      --base "${base_branch}" \
      --head "${branch_name}" \
      --title "chore(deps): bump market-kit to ${MARKET_KIT_TAG}" \
      --body "$(pr_body "${repo}")"
  fi

  rm -rf "${worktree}"
done < <(jq -c '.[]' "${CONFIG_PATH}")
