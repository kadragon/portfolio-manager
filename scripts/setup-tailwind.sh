#!/usr/bin/env bash
set -euo pipefail

# Downloads Tailwind CSS standalone CLI and DaisyUI bundle
# No Node.js required

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="$PROJECT_ROOT/bin"
TAILWIND_DIR="$PROJECT_ROOT/src/portfolio_manager/web/tailwind"

mkdir -p "$BIN_DIR" "$TAILWIND_DIR"

# --- Detect OS/Arch ---
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  darwin) PLATFORM="macos" ;;
  linux)  PLATFORM="linux" ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64)  ARCH_SUFFIX="x64" ;;
  arm64|aarch64) ARCH_SUFFIX="arm64" ;;
  *)             echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

TAILWIND_BIN="$BIN_DIR/tailwindcss"
DAISYUI_CSS="$TAILWIND_DIR/daisyui.css"

# --- Download Tailwind CSS standalone CLI v4 ---
if [ ! -f "$TAILWIND_BIN" ]; then
  TAILWIND_URL="https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-${PLATFORM}-${ARCH_SUFFIX}"
  echo "Downloading Tailwind CSS standalone CLI..."
  curl -sL "$TAILWIND_URL" -o "$TAILWIND_BIN"
  chmod +x "$TAILWIND_BIN"
  echo "Tailwind CSS installed at $TAILWIND_BIN"
else
  echo "Tailwind CSS already installed at $TAILWIND_BIN"
fi

# --- Download DaisyUI CSS bundle ---
if [ ! -f "$DAISYUI_CSS" ]; then
  DAISYUI_URL="https://cdn.jsdelivr.net/npm/daisyui@5/daisyui.css"
  echo "Downloading DaisyUI CSS bundle..."
  curl -sL "$DAISYUI_URL" -o "$DAISYUI_CSS"
  echo "DaisyUI installed at $DAISYUI_CSS"
else
  echo "DaisyUI already installed at $DAISYUI_CSS"
fi

# --- Initial build ---
INPUT_CSS="$TAILWIND_DIR/input.css"
OUTPUT_CSS="$PROJECT_ROOT/src/portfolio_manager/web/static/css/app.css"

if [ -f "$INPUT_CSS" ]; then
  echo "Building CSS..."
  "$TAILWIND_BIN" -i "$INPUT_CSS" -o "$OUTPUT_CSS"
  echo "CSS built at $OUTPUT_CSS"
else
  echo "Warning: $INPUT_CSS not found. Create it first, then run:"
  echo "  $TAILWIND_BIN -i $INPUT_CSS -o $OUTPUT_CSS"
fi

echo "Done! Run 'make css-watch' to start development."
