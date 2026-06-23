#!/usr/bin/env sh
# ANS install script (Linux / macOS / WSL)
# Usage: curl -fsSL https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.sh | sh
# SPDX-License-Identifier: Apache-2.0
set -eu

REPO="Linky-Link-Linky/Agent-Nervous-System"
BINARY="ans"

# Allow version pin via ANS_VERSION env var
VERSION="${ANS_VERSION:-latest}"
if [ "$VERSION" = "latest" ]; then
  BASE="https://github.com/${REPO}/releases/latest/download"
else
  BASE="https://github.com/${REPO}/releases/${VERSION}/download"
fi

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

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "${TMP}/${BINARY}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "${TMP}/${BINARY}" | awk '{print $1}')"
else
  echo "Error: no sha256sum or shasum found" >&2; exit 1
fi

if [ "$ACTUAL" != "$EXPECTED" ]; then
  echo "Checksum mismatch: expected $EXPECTED, got $ACTUAL" >&2; exit 1
fi

# Install — try /usr/local/bin first, then sudo, then $HOME/.local/bin
for DEST_DIR in "/usr/local/bin" "$HOME/.local/bin" "$HOME/bin"; do
  mkdir -p "$DEST_DIR" 2>/dev/null || true
  if [ -w "$DEST_DIR" ] && cp "${TMP}/${BINARY}" "${DEST_DIR}/${BINARY}" 2>/dev/null; then
    DEST="${DEST_DIR}/${BINARY}"
    break
  fi
  if command -v sudo >/dev/null 2>&1 && sudo cp "${TMP}/${BINARY}" "${DEST_DIR}/${BINARY}" 2>/dev/null; then
    DEST="${DEST_DIR}/${BINARY}"
    break
  fi
done

if [ -z "${DEST:-}" ]; then
  mkdir -p "$HOME/.local/bin"
  cp "${TMP}/${BINARY}" "$HOME/.local/bin/${BINARY}"
  DEST="$HOME/.local/bin/${BINARY}"
fi

echo ""
echo " ANS installed: $DEST" | sed "s|$HOME|~|g"

# -- PATH setup -----------------------------------------------------------
INSTALL_DIR="$(dirname "$DEST")"

ensure_path_entry() {
  case ":$PATH:" in
    *":$1:"*) return 0 ;;  # already in PATH
    *) return 1 ;;
  esac
}

if ensure_path_entry "$INSTALL_DIR"; then
  : # already in PATH, nothing to do
else
  # Determine shell config file
  SHELL_NAME="${SHELL##*/}"
  case "$SHELL_NAME" in
    zsh)  RC_FILE="${ZDOTDIR:-$HOME}/.zshrc" ;;
    bash) RC_FILE="$HOME/.bashrc" ;;
    dash|sh) RC_FILE="$HOME/.profile" ;;
    *)    RC_FILE="$HOME/.profile" ;;
  esac

  LINE="export PATH=\"\$PATH:$INSTALL_DIR\""

  if [ -f "$RC_FILE" ] && grep -qsF "$INSTALL_DIR" "$RC_FILE" 2>/dev/null; then
    : # already in config file but not yet in this session's PATH
  elif [ -f "$RC_FILE" ] && [ -w "$RC_FILE" ]; then
    printf '\n%s\n' "$LINE" >> "$RC_FILE"
    echo " Added to $RC_FILE"
  elif [ -w "$HOME" ]; then
    printf '\n%s\n' "$LINE" >> "$RC_FILE" 2>/dev/null || true
    echo " Created $RC_FILE with PATH entry"
  fi

  echo ""
  echo " To use 'ans' now, run:"
  echo "   export PATH=\"\$PATH:$INSTALL_DIR\""
  echo ""
  echo " Or open a new terminal window (it will pick up $RC_FILE)."
  echo ""
fi

echo " Start the daemon:   ans start"
echo " Register an agent:  ans register --name my-agent --version 1.0.0"
echo " View the chain:     ans chain"
echo ""
"$DEST" version 2>/dev/null || echo " Run 'ans version' to verify."
