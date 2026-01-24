#!/usr/bin/env bash
set -euo pipefail

tag="${1:-}"
checksums_path="${2:-}"

if [[ -z "${tag}" || -z "${checksums_path}" ]]; then
  echo "usage: scripts/update-homebrew-formula.sh <tag> <checksums.txt>" >&2
  exit 2
fi

if [[ ! -f "${checksums_path}" ]]; then
  echo "checksums file not found: ${checksums_path}" >&2
  exit 2
fi

version="${tag#v}"
formula_path="Formula/gion.rb"
formula_dir="$(dirname "${formula_path}")"

sha_for() {
  local asset="$1"
  awk -v asset="${asset}" '($2 == asset) { print $1; found=1 } END { if (!found) exit 3 }' "${checksums_path}"
}

macos_arm64_sha="$(sha_for "gion_${tag}_macos_arm64.tar.gz")"
macos_x64_sha="$(sha_for "gion_${tag}_macos_x64.tar.gz")"
linux_arm64_sha="$(sha_for "gion_${tag}_linux_arm64.tar.gz")"
linux_x64_sha="$(sha_for "gion_${tag}_linux_x64.tar.gz")"

mkdir -p "${formula_dir}"

cat >"${formula_path}" <<EOF
class Gion < Formula
  desc "Inventory-driven Git workspaces with guardrails"
  homepage "https://github.com/tasuku43/gion"
  license "MIT"

  version "${version}"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/tasuku43/gion/releases/download/${tag}/gion_${tag}_macos_arm64.tar.gz"
      sha256 "${macos_arm64_sha}"
    else
      url "https://github.com/tasuku43/gion/releases/download/${tag}/gion_${tag}_macos_x64.tar.gz"
      sha256 "${macos_x64_sha}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/tasuku43/gion/releases/download/${tag}/gion_${tag}_linux_arm64.tar.gz"
      sha256 "${linux_arm64_sha}"
    else
      url "https://github.com/tasuku43/gion/releases/download/${tag}/gion_${tag}_linux_x64.tar.gz"
      sha256 "${linux_x64_sha}"
    end
  end

  def install
    bin.install "gion", "giongo"
  end

  test do
    system "#{bin}/gion", "--version"
    system "#{bin}/giongo", "--version"
  end
end
EOF

echo "updated ${formula_path} for ${tag}"
