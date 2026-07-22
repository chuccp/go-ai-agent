#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# run.sh — Cross-platform launcher for go-ai-agent CLI
# Supports: Linux, macOS, Windows (Git Bash / WSL)
# Usage:    ./run.sh
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

# ── Colors (auto-detect TTY support) ────────────────────────────────────────
if [ -t 1 ]; then
  BOLD=$(tput bold 2>/dev/null || echo '')
  RED=$(tput setaf 1 2>/dev/null || echo '')
  GREEN=$(tput setaf 2 2>/dev/null || echo '')
  YELLOW=$(tput setaf 3 2>/dev/null || echo '')
  CYAN=$(tput setaf 6 2>/dev/null || echo '')
  RESET=$(tput sgr0 2>/dev/null || echo '')
else
  BOLD='' RED='' GREEN='' YELLOW='' CYAN='' RESET=''
fi

# ── OS Detection ────────────────────────────────────────────────────────────
detect_os() {
  case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
    darwin)  echo "macos"   ;;
    linux)   echo "linux"   ;;
    mingw*|msys*|cygwin*) echo "windows" ;;
    *)       echo "unknown" ;;
  esac
}

OS=$(detect_os)

# ── Banner ──────────────────────────────────────────────────────────────────
echo ""
echo "${CYAN}${BOLD}⚡ go-ai-agent CLI Launcher${RESET}"
echo "${CYAN}  OS: ${YELLOW}${OS}${RESET}  |  ${CYAN}Shell: ${YELLOW}${SHELL##*/}${RESET}"
echo ""

# ── Preflight: Go ───────────────────────────────────────────────────────────
if ! command -v go &>/dev/null; then
  echo "${RED}${BOLD}✗ Go is not installed.${RESET}"
  echo "  Install from: ${CYAN}https://go.dev/dl/${RESET}"
  exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "${GREEN}✓${RESET} Go ${GO_VERSION}"

# ── Preflight: project root ─────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [ "$(pwd)" != "$SCRIPT_DIR" ]; then
  echo "  ${YELLOW}↳ cd to project root${RESET}"
  cd "$SCRIPT_DIR"
fi

if [ ! -f "go.mod" ]; then
  echo "${RED}${BOLD}✗ go.mod not found — are you in the project root?${RESET}"
  exit 1
fi

echo "${GREEN}✓${RESET} Project root: ${SCRIPT_DIR}"

# ── Run ─────────────────────────────────────────────────────────────────────
echo ""
echo "${BOLD}Starting CLI...${RESET}"
echo ""

if [ "$OS" = "windows" ]; then
  # Windows (Git Bash / MSYS2): ensure terminal supports TUI
  export TERM="${TERM:-xterm-256color}"
fi

go run ./cmd/cli/

# ── Exit ────────────────────────────────────────────────────────────────────
echo ""
echo "${GREEN}CLI exited.${RESET}"
