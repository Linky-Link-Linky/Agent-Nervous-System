#!/usr/bin/env sh
# ANS install script
# Usage: curl -fsSL https://ans-project.github.io/install.sh | sh
# SPDX-License-Identifier: MIT
set -eu

REPO="ans-project/ans"
INSTALL_DIR="/usr/local/bin"
BINARY="ans"

case "$(uname -s)" in
  Linux)  OS="linux"  ;;
  Darwin) OS="darwin" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows"; BINARY="ans.exe" ;;
  *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

ASSET="ans_${OS}_${ARCH}"
[ "$OS" = "windows" ] && ASSET="${ASSET}.exe"
BASE="https://github.com/${REPO}/releases/latest/download"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ANS for ${OS}/${ARCH}..."
curl -fsSL "${BASE}/${ASSET}" -o "${TMP}/${BINARY}"
curl -fsSL "${BASE}/checksums.txt" -o "${TMP}/checksums.txt"

EXPECTED="$(grep -F "$ASSET" "${TMP}/checksums.txt" | awk '{print $1}')"
if [ -z "$EXPECTED" ]; then
  echo "Error: checksum not found for $ASSET" >&2; exit 1
fi

chmod +x "${TMP}/${BINARY}"

# Verify checksum robustly
if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "${TMP}/${BINARY}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "${TMP}/${BINARY}" | awk '{print $1}')"
else
  echo "Error: neither sha256sum nor shasum found" >&2; exit 1
fi

if [ "$ACTUAL" != "$EXPECTED" ]; then
  echo "Checksum mismatch: expected $EXPECTED, got $ACTUAL" >&2; exit 1
fi

# Install — try direct copy, then sudo, then fall back to ~/bin
if cp "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}" 2>/dev/null; then
  DEST="${INSTALL_DIR}/${BINARY}"
elif sudo cp "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}" 2>/dev/null; then
  DEST="${INSTALL_DIR}/${BINARY}"
else
  mkdir -p "$HOME/bin"
  cp "${TMP}/${BINARY}" "$HOME/bin/${BINARY}"
  DEST="$HOME/bin/${BINARY}"
  echo "Installed to $DEST — add $HOME/bin to your PATH if not already present"
fi

echo "ANS installed: $DEST"
"$DEST" start
echo "Run 'ans chain' to view the receipt chain."
