#!/usr/bin/env sh
# ANS install script (Linux / macOS / WSL)
# Usage: curl -fsSL https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.sh | sh
# SPDX-License-Identifier: Apache-2.0
set -eu

REPO="Linky-Link-Linky/Agent-Nervous-System"
BINARY="ans"
VERSION="${ANS_VERSION:-latest}"

# --- Daytona-inspired emerald theme ---
EMERALD='\033[38;2;46;204;113m'
YELLOW='\033[38;2;241;196;15m'
RED='\033[38;2;231;76;60m'
GRAY='\033[38;5;243m'
MUTED='\033[38;5;236m'
BOLD='\033[1m'
RESET='\033[0m'

step()   { printf "  ${EMERALD}%s.${RESET} ${BOLD}%s${RESET}\n" "$1" "$2"; }
done_()  { printf "  ${EMERALD}\xe2\x97\x8f${RESET} %s\n" "$1"; }
warn()   { printf "  ${YELLOW}${BOLD}!${RESET} %s\n" "$1"; }
cmd_()   { printf "    ${GRAY}\$${RESET} %s\n" "$1"; }
info()   { printf "     ${GRAY}%s${RESET}\n" "$1"; }
err_()   { printf "  ${RED}\xe2\x9c\x97${RESET} %s\n" "$1" >&2; }
banner() {
  printf "\n"
  printf "  ${MUTED}────────────────────────────────────────${RESET}\n"
  printf "  ${EMERALD}  \xe2\x9c\xa6${RESET}  ${BOLD}Agent Nervous System${RESET}\n"
  printf "  ${GRAY}  Secure AI Agent Auditing${RESET}\n"
  printf "  ${MUTED}────────────────────────────────────────${RESET}\n"
  printf "\n"
}

banner
step 1 "Detecting your system..."

# --- OS detection ---
case "$(uname -s)" in
  Linux)  OS="linux"  ;;
  Darwin) OS="darwin" ;;
  *)
    err_ "Unsupported OS: $(uname -s)"
    exit 1
    ;;
esac

# --- Arch detection ---
case "$(uname -m)" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    err_ "Unsupported arch: $(uname -m)"
    exit 1
    ;;
esac

info "${OS}/${ARCH}"

ASSET="ans_${OS}_${ARCH}"
[ "$OS" = "windows" ] && ASSET="${ASSET}.exe"

if [ "$VERSION" = "latest" ]; then
  BASE="https://github.com/${REPO}/releases/latest/download"
else
  BASE="https://github.com/${REPO}/releases/download/${VERSION}"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

done_ "System detected"

# --- Download ---
step 2 "Downloading ANS for ${OS}/${ARCH}..."
if ! curl -fsSL "${BASE}/${ASSET}" -o "${TMP}/${BINARY}"; then
  err_ "Download failed — check your internet connection"
  exit 1
fi
done_ "Downloaded ${ASSET}"

# --- Optional checksum ---
if curl -fsSL "${BASE}/checksums.txt" -o "${TMP}/checksums.txt" 2>/dev/null; then
  EXPECTED="$(grep -F "$ASSET" "${TMP}/checksums.txt" | awk '{print $1}')"
  if [ -n "$EXPECTED" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL="$(sha256sum "${TMP}/${BINARY}" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL="$(shasum -a 256 "${TMP}/${BINARY}" | awk '{print $1}')"
    else
      warn "No sha256sum found — skipping verification"
    fi
    if [ -n "${ACTUAL:-}" ] && [ "$ACTUAL" != "$EXPECTED" ]; then
      err_ "Checksum mismatch!"
      exit 1
    fi
    done_ "Checksum verified"
  fi
else
  warn "Checksum file not available — skipped"
fi

chmod +x "${TMP}/${BINARY}"

# --- Install ---
step 3 "Installing binaries..."

DEST=""
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

if [ -z "$DEST" ]; then
  mkdir -p "$HOME/.local/bin"
  cp "${TMP}/${BINARY}" "$HOME/.local/bin/${BINARY}"
  DEST="$HOME/.local/bin/${BINARY}"
fi

done_ "Installed to $(echo "$DEST" | sed "s|$HOME|~|g")"

# --- PATH setup ---
step 4 "Checking PATH..."
INSTALL_DIR="$(dirname "$DEST")"

ensure_path() {
  case ":$PATH:" in
    *":$1:"*) return 0 ;;
    *) return 1 ;;
  esac
}

if ensure_path "$INSTALL_DIR"; then
  done_ "Already in PATH"
else
  : "${HOME:=/root}"
  : "${SHELL:=sh}"
  SHELL_NAME="${SHELL##*/}"
  case "$SHELL_NAME" in
    zsh)  RC_FILE="${ZDOTDIR:-$HOME}/.zshrc" ;;
    bash) RC_FILE="$HOME/.bashrc" ;;
    dash|sh) RC_FILE="$HOME/.profile" ;;
    *)    RC_FILE="$HOME/.profile" ;;
  esac

  LINE="export PATH=\"\$PATH:$INSTALL_DIR\""

  if [ -f "$RC_FILE" ] && grep -qsF "$INSTALL_DIR" "$RC_FILE" 2>/dev/null; then
    warn "Already in $RC_FILE but not in current session"
  elif [ -f "$RC_FILE" ] && [ -w "$RC_FILE" ]; then
    printf '\n%s\n' "$LINE" >> "$RC_FILE"
    done_ "Added to $RC_FILE"
  elif [ -w "$HOME" ]; then
    printf '\n%s\n' "$LINE" >> "$RC_FILE" 2>/dev/null || true
    done_ "Created $RC_FILE with PATH entry"
  else
    warn "Could not write to $RC_FILE"
  fi

  printf "\n  ${YELLOW}To use 'ans' now, run:${RESET}\n"
  cmd_ "export PATH=\"\$PATH:$INSTALL_DIR\""
  printf "  ${GRAY}Or open a new terminal window.${RESET}\n"
fi

# --- Auto-start daemon ---
step 6 "Starting the ANS daemon..."
if "$DEST" init 2>/dev/null && "$DEST" start 2>/dev/null; then
  done_ "Daemon started"
else
  warn "Could not auto-start daemon — run 'ans start' manually"
fi

# --- Success message ---
printf "\n"
printf "  ${MUTED}────────────────────────────────────────${RESET}\n"
printf "  ${EMERALD}  \xe2\x9c\xa6${RESET}  ${BOLD}ANS is installed!${RESET}\n"
printf "  ${MUTED}────────────────────────────────────────${RESET}\n"
printf "\n"
printf "  ${BOLD}Quick start:${RESET}\n"
printf "\n"
cmd_ "ans"
printf "  ${GRAY}  Opens the live dashboard (full-screen TUI)${RESET}\n"
printf "\n"
cmd_ "ans register"
printf "  ${GRAY}  Register an AI agent (name and version auto-generated)${RESET}\n"
printf "\n"
cmd_ "ans chain"
printf "  ${GRAY}  View the receipt chain${RESET}\n"
printf "\n"
cmd_ "ans init --service"
printf "  ${GRAY}  Auto-start ANS at system boot${RESET}\n"
printf "\n"
cmd_ "ans update"
printf "  ${GRAY}  Self-update to the latest release${RESET}\n"
printf "\n"
cmd_ "ans tui"
printf "  ${GRAY}  Launch the full-screen TUI dashboard${RESET}\n"
printf "\n"
printf "  ${EMERALD}Need help? Run: ans doctor${RESET}\n"
printf "\n"
