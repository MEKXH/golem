#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 4 ]]; then
  echo "usage: $0 <tag> <linux_binary> <windows_binary> <output_file>"
  exit 1
fi

tag="$1"
linux_bin="$2"
windows_bin="$3"
output_file="$4"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

template_file=".github/release-notes-template.md"
if [[ ! -f "${template_file}" ]]; then
  echo "missing release notes template: ${template_file}"
  exit 1
fi

repo_url="https://github.com/${GITHUB_REPOSITORY:-MEKXH/golem}"
previous_tag="$(git tag --sort=-version:refname | awk -v cur="${tag}" '$0 != cur { print; exit }')"
if [[ -n "${previous_tag}" ]]; then
  compare_line="- Compare: ${repo_url}/compare/${previous_tag}...${tag}"
  range_spec="${previous_tag}..${tag}"
else
  compare_line="- Compare: N/A (first release tag)"
  range_spec=""
fi

if [[ -n "${range_spec}" ]]; then
  changelog="$(git log "${range_spec}" --pretty=format:'- %s (%h)')"
else
  changelog="$(git log -n 20 --pretty=format:'- %s (%h)')"
fi
if [[ -z "${changelog}" ]]; then
  changelog="- No commit messages found for this range."
fi

checksum_linux="$(sha256sum "${linux_bin}" | awk '{print $1}')"
checksum_windows="$(sha256sum "${windows_bin}" | awk '{print $1}')"
checksum_linux="${checksum_linux#\\}"
checksum_windows="${checksum_windows#\\}"
checksums="$(printf '%s  %s\n%s  %s\n' "${checksum_linux}" "$(basename "${linux_bin}")" "${checksum_windows}" "$(basename "${windows_bin}")")"

{
  echo "## Golem ${tag}"
  echo
  echo "- Date: $(date -u +'%Y-%m-%d')"
  echo "${compare_line}"
  echo
  cat "${template_file}"
} > "${output_file}"

AUTO_CHANGELOG="${changelog}" AUTO_CHECKSUMS="${checksums}" awk '
  /<!-- AUTO_CHANGELOG -->/ {
    print ENVIRON["AUTO_CHANGELOG"]
    next
  }
  /<!-- AUTO_CHECKSUMS -->/ {
    print ENVIRON["AUTO_CHECKSUMS"]
    next
  }
  { print }
' "${output_file}" > "${output_file}.tmp"

mv "${output_file}.tmp" "${output_file}"
echo "generated release notes: ${output_file}"
