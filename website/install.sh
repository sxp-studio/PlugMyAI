#!/bin/sh
# plug-my-ai installer
# Usage: curl -fsSL https://plugmy.ai/install.sh | sh

set -e

REPO="sxp-studio/PlugMyAI"
INSTALL_DIR="$HOME/.plug-my-ai/bin"
BINARY_NAME="plug-my-ai"

# Detect OS and architecture
detect_platform() {
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"

  case "$OS" in
    linux)  OS="linux" ;;
    darwin) OS="darwin" ;;
    *)
      echo "Error: Unsupported operating system: $OS"
      exit 1
      ;;
  esac

  case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
      echo "Error: Unsupported architecture: $ARCH"
      exit 1
      ;;
  esac

  PLATFORM="${OS}-${ARCH}"
}

# Get latest release tag from GitHub
get_latest_version() {
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  if [ -z "$VERSION" ]; then
    echo "Error: Could not determine latest version"
    exit 1
  fi
}

# Download and install
install() {
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${PLATFORM}"

  echo "plug-my-ai installer"
  echo "===================="
  echo "Version:  ${VERSION}"
  echo "Platform: ${PLATFORM}"
  echo "Install:  ${INSTALL_DIR}/${BINARY_NAME}"
  echo ""

  # Create install directory
  mkdir -p "$INSTALL_DIR"

  # Download binary
  echo "Downloading..."
  curl -fsSL "$DOWNLOAD_URL" -o "${INSTALL_DIR}/${BINARY_NAME}"
  chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

  echo "Downloaded successfully."
  echo ""

  # Add to PATH if not already there
  add_to_path

  # Run init
  "${INSTALL_DIR}/${BINARY_NAME}" init

  echo ""
  echo "Installation complete!"
  echo ""
  echo "Run 'plug-my-ai' to start the daemon."
}

add_to_path() {
  case ":$PATH:" in
    *":$INSTALL_DIR:"*) return ;; # Already in PATH
  esac

  SHELL_NAME="$(basename "$SHELL")"
  PROFILE=""

  case "$SHELL_NAME" in
    zsh)  PROFILE="$HOME/.zshrc" ;;
    bash)
      if [ -f "$HOME/.bash_profile" ]; then
        PROFILE="$HOME/.bash_profile"
      else
        PROFILE="$HOME/.bashrc"
      fi
      ;;
    fish) PROFILE="$HOME/.config/fish/config.fish" ;;
  esac

  if [ -n "$PROFILE" ]; then
    EXPORT_LINE="export PATH=\"${INSTALL_DIR}:\$PATH\""

    if [ "$SHELL_NAME" = "fish" ]; then
      EXPORT_LINE="set -gx PATH ${INSTALL_DIR} \$PATH"
    fi

    # Only add if not already present
    if ! grep -q "plug-my-ai" "$PROFILE" 2>/dev/null; then
      echo "" >> "$PROFILE"
      echo "# plug-my-ai" >> "$PROFILE"
      echo "$EXPORT_LINE" >> "$PROFILE"
      echo "Added ${INSTALL_DIR} to PATH in ${PROFILE}"
      echo "Run 'source ${PROFILE}' or open a new terminal to use plug-my-ai."
    fi
  fi
}

detect_platform
get_latest_version
install
