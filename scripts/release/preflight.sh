#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <tag>"
  exit 1
fi

tag="$1"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

if [[ ! "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$ ]]; then
  echo "invalid release tag: ${tag} (expected semver, e.g. v0.5.1)"
  exit 1
fi

if ! test -f .github/release-notes-template.md; then
  echo "missing .github/release-notes-template.md"
  exit 1
fi

echo "[preflight] tag: ${tag}"
echo "[preflight] go test ./... -count=1"
go test ./... -count=1

if [[ "$(go env CGO_ENABLED)" == "1" ]]; then
  echo "[preflight] go test -race ./... -count=1"
  go test -race ./... -count=1
else
  echo "[preflight] skip go test -race (CGO_ENABLED=$(go env CGO_ENABLED))"
fi

echo "[preflight] go vet ./..."
go vet ./...

echo "[preflight] passed"
