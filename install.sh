#!/usr/bin/env bash
set -euo pipefail

# install.sh - build and install the aether binary
# Usage:
#   ./install.sh [--prefix /usr/local] [--dest /custom/bin] [--no-sudo] [--build-only]
# Examples:
#   ./install.sh                # build and install to /usr/local/bin (uses sudo if needed)
#   ./install.sh --prefix $HOME/.local --no-sudo
#   ./install.sh --build-only   # only build, don't install
#
# Behavior:
# - Builds the Go project located in the same directory as this script.
# - By default installs the resulting `aether` binary to /usr/local/bin.
# - Use --prefix to change the prefix (DEST defaults to PREFIX/bin).
# - Use --dest to set the exact installation directory (overrides --prefix).
# - Use --no-sudo to avoid invoking sudo when creating directories or moving the binary.
# - Use --build-only to only produce the built binary in ./build/aether and skip installation.

PREFIX="/usr/local"
DEST_DIR=""
USE_SUDO=true
BUILD_ONLY=false
BUILD_DIR="build"
BINARY_NAME="aether"

print_usage() {
    cat <<EOF
Usage: $0 [options]

Options:
  --prefix DIR     Install prefix (default: /usr/local). Binary will go to PREFIX/bin unless --dest is used.
  --dest DIR       Exact directory to install the binary into (overrides --prefix).
  --no-sudo        Do not attempt to use sudo when installing to system directories.
  --build-only     Only build the binary, do not install it.
  -h, --help       Show this help and exit.
EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --prefix)
            if [[ $# -lt 2 ]]; then
                echo "Missing value for --prefix" >&2
                exit 2
            fi
            PREFIX="$2"
            shift 2
            ;;
        --dest)
            if [[ $# -lt 2 ]]; then
                echo "Missing value for --dest" >&2
                exit 2
            fi
            DEST_DIR="$2"
            shift 2
            ;;
        --no-sudo)
            USE_SUDO=false
            shift
            ;;
        --build-only)
            BUILD_ONLY=true
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            echo "Unknown argument: $1" >&2
            print_usage
            exit 2
            ;;
    esac
done

if [[ -z "$DEST_DIR" ]]; then
    DEST_DIR="$PREFIX/bin"
fi

# Move to script directory (assumes the repo / project root is here)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Ensure Go toolchain is available
if ! command -v go >/dev/null 2>&1; then
    echo "Error: Go toolchain not found in PATH. Please install Go and try again." >&2
    exit 1
fi

# Print Go version for visibility (non-fatal)
if command -v go >/dev/null 2>&1; then
    echo "Using $(go version)"
fi

echo "Building ${BINARY_NAME}..."

# Prepare build directory
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Build the project's main package. We intentionally build the package rooted at this directory
# so that the produced binary includes the main program present here.
if ! go build -o "$BUILD_DIR/$BINARY_NAME" .; then
    echo "Build failed." >&2
    exit 1
fi

echo "Built: $BUILD_DIR/$BINARY_NAME"

if [[ "$BUILD_ONLY" == true ]]; then
    echo "Build-only requested; not installing. Binary is at: $BUILD_DIR/$BINARY_NAME"
    exit 0
fi

# Ensure destination directory exists (create if necessary)
create_dir() {
    local d="$1"
    if [[ -d "$d" ]]; then
        return 0
    fi
    if [[ "$USE_SUDO" == true && "$(id -u)" -ne 0 ]]; then
        echo "Creating destination directory with sudo: $d"
        if ! sudo mkdir -p "$d"; then
            echo "Failed to create destination directory: $d" >&2
            exit 1
        fi
    else
        mkdir -p "$d"
    fi
}

create_dir "$DEST_DIR"

# Install (move) the binary
install_binary() {
    local src="$1"
    local dst_dir="$2"
    local dst="$dst_dir/$BINARY_NAME"

    if [[ "$USE_SUDO" == true && "$(id -u)" -ne 0 ]]; then
        echo "Installing to $dst_dir with sudo..."
        if ! sudo mv -f "$src" "$dst"; then
            echo "Failed to move binary to $dst" >&2
            exit 1
        fi
        if ! sudo chmod 755 "$dst"; then
            echo "Warning: failed to set permissions on $dst" >&2
        fi
    else
        echo "Installing to $dst_dir..."
        mv -f "$src" "$dst"
        chmod 755 "$dst"
    fi

    echo "Installed: $dst"
}

install_binary "$BUILD_DIR/$BINARY_NAME" "$DEST_DIR"

echo "Installation complete. You can run '${BINARY_NAME}' from the command line."

exit 0
