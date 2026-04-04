#!/usr/bin/env sh
# install.sh — install noteui and noteui-sync from GitHub Releases
set -e

REPO="atbuy/noteui"
DEFAULT_INSTALL_DIR="${HOME}/.local/bin"
INSTALL_DIR=""
VERSION=""
NO_SYNC=0
USE_SYSTEM=0

usage() {
  cat <<EOF
Usage: install.sh [OPTIONS]

Install noteui from GitHub Releases.

Options:
  --version <tag>   Install a specific release (e.g. v0.9.1). Default: latest
  --system          Install to /usr/local/bin instead of ~/.local/bin (requires sudo)
  --no-sync         Skip installing noteui-sync
  --help            Show this help message
EOF
}

# Parse arguments
while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="$2"
      shift 2
      ;;
    --version=*)
      VERSION="${1#*=}"
      shift
      ;;
    --system)
      USE_SYSTEM=1
      shift
      ;;
    --no-sync)
      NO_SYNC=1
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ "${USE_SYSTEM}" = "1" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="${DEFAULT_INSTALL_DIR}"
fi

# Detect OS
OS="$(uname -s)"
case "${OS}" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  *)
    echo "Unsupported operating system: ${OS}" >&2
    echo "This script supports Linux and macOS. For Windows, use install.ps1." >&2
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64)   ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

# Check for required tools
for tool in curl tar; do
  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "Required tool not found: ${tool}" >&2
    exit 1
  fi
done

# Fetch latest version if not specified
if [ -z "${VERSION}" ]; then
  echo "Fetching latest release version..."
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
  if [ -z "${VERSION}" ]; then
    echo "Failed to fetch latest release version from GitHub API." >&2
    exit 1
  fi
fi

echo "Installing noteui ${VERSION} (${OS}/${ARCH})..."
echo ""

ARCHIVE_BASENAME="noteui-${VERSION}-${OS}-${ARCH}"
ARCHIVE="${ARCHIVE_BASENAME}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

# Create install dir if needed
mkdir -p "${INSTALL_DIR}" 2>/dev/null || {
  echo "Cannot create ${INSTALL_DIR} — try --system or create the directory manually." >&2
  exit 1
}

# Temp dir with cleanup
TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT INT TERM

# Download
echo "Downloading ${ARCHIVE}..."
if ! curl -fsSL --progress-bar "${DOWNLOAD_URL}" -o "${TMPDIR}/${ARCHIVE}"; then
  echo "" >&2
  echo "Download failed. Check that ${VERSION} exists at:" >&2
  echo "  https://github.com/${REPO}/releases" >&2
  exit 1
fi

# Extract
echo "Extracting..."
tar -xzf "${TMPDIR}/${ARCHIVE}" -C "${TMPDIR}"

# Install noteui
NOTEUI_BIN="${TMPDIR}/noteui-${VERSION}-${OS}-${ARCH}"
if [ ! -f "${NOTEUI_BIN}" ]; then
  echo "Expected binary not found in archive: noteui-${VERSION}-${OS}-${ARCH}" >&2
  exit 1
fi
chmod +x "${NOTEUI_BIN}"

if [ "${USE_SYSTEM}" = "1" ]; then
  sudo mv "${NOTEUI_BIN}" "${INSTALL_DIR}/noteui"
else
  mv "${NOTEUI_BIN}" "${INSTALL_DIR}/noteui"
fi

# Install noteui-sync
if [ "${NO_SYNC}" = "0" ]; then
  SYNC_BIN="${TMPDIR}/noteui-sync-${VERSION}-${OS}-${ARCH}"
  if [ -f "${SYNC_BIN}" ]; then
    chmod +x "${SYNC_BIN}"
    if [ "${USE_SYSTEM}" = "1" ]; then
      sudo mv "${SYNC_BIN}" "${INSTALL_DIR}/noteui-sync"
    else
      mv "${SYNC_BIN}" "${INSTALL_DIR}/noteui-sync"
    fi
  fi
fi

echo ""
echo "Installed to ${INSTALL_DIR}/"

# PATH hint for ~/.local/bin if it's not on PATH
if [ "${USE_SYSTEM}" != "1" ]; then
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*)
      ;;
    *)
      echo ""
      echo "${INSTALL_DIR} is not on your PATH."
      echo "Add the following to your shell profile (~/.bashrc, ~/.zshrc, or ~/.profile):"
      echo ""
      echo '  export PATH="${HOME}/.local/bin:${PATH}"'
      echo ""
      echo "Then restart your terminal or run: source ~/.profile"
      ;;
  esac
fi

# Verify
echo ""
if command -v noteui >/dev/null 2>&1; then
  noteui --version
else
  "${INSTALL_DIR}/noteui" --version
fi
