#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 3 ]]; then
  echo "usage: $0 <tag> [health_url] [version_url]"
  echo "example: $0 v0.5.0 http://127.0.0.1:18790/health http://127.0.0.1:18790/version"
  exit 1
fi

tag="$1"
health_url="${2:-http://127.0.0.1:18790/health}"
version_url="${3:-http://127.0.0.1:18790/version}"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

if [[ ! "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$ ]]; then
  echo "invalid tag format: ${tag}"
  exit 1
fi

git fetch --tags --quiet
if ! git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
  echo "tag not found: ${tag}"
  exit 1
fi

echo "[rollback] checkout ${tag}"
git checkout "${tag}"

echo "[rollback] build binary"
go build -o golem ./cmd/golem

echo "[rollback] rollback build is ready (tag=${tag})"
echo "[rollback] post-deploy verification commands:"
echo "curl -fsS ${health_url}"
echo "curl -fsS ${version_url}"
