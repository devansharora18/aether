#!/usr/bin/env bash
set -euo pipefail

# install-aether.sh - curl|bash installer for Aether
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/devansharora18/aether/main/install-aether.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/devansharora18/aether/main/install-aether.sh | bash -s -- --prefix ~/.local --no-sudo
#
# This script downloads the repo, runs install.sh, then cleans up.

usage() {
    cat <<'EOF'
Usage: install-aether.sh [install.sh args]

Examples:
  ./install-aether.sh
  ./install-aether.sh --prefix ~/.local --no-sudo
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi

if ! command -v tar >/dev/null 2>&1; then
    echo "Error: tar is required to install Aether." >&2
    exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
    echo "Error: curl is required to install Aether." >&2
    exit 1
fi

tmp_dir="$(mktemp -d)"
cleanup() {
    rm -rf "$tmp_dir"
}
trap cleanup EXIT

archive_url="https://codeload.github.com/devansharora18/aether/tar.gz/main"

echo "Downloading Aether..."
curl -fsSL "$archive_url" | tar -xz -C "$tmp_dir"

repo_dir="$tmp_dir/aether-main"
if [[ ! -d "$repo_dir" ]]; then
    echo "Error: extracted repository not found." >&2
    exit 1
fi

cd "$repo_dir"

if [[ ! -x "./install.sh" ]]; then
    echo "Error: install.sh not found in repo." >&2
    exit 1
fi

./install.sh "$@"
