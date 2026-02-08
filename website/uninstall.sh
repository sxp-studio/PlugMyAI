#!/bin/sh
# plug-my-ai uninstaller
# Usage: curl -fsSL https://plugmy.ai/uninstall.sh | sh
set -e

echo "plug-my-ai uninstaller"
echo "======================"
echo ""

# ── Stop running processes ───────────────────────────────
echo "Stopping processes..."

# Kill the menu bar app (graceful quit first, then force)
osascript -e 'quit app "PlugMyAI"' 2>/dev/null || true
sleep 1

# Force-kill anything still running
pkill -f "PlugMyAI.app" 2>/dev/null && echo "  Stopped PlugMyAI app" || true
pkill -f "plug-my-ai" 2>/dev/null && echo "  Stopped plug-my-ai daemon" || true

# ── Remove LaunchAgent ───────────────────────────────────
PLIST="$HOME/Library/LaunchAgents/ai.plugmy.daemon.plist"
if [ -f "$PLIST" ]; then
  launchctl bootout "gui/$(id -u)" "$PLIST" 2>/dev/null || true
  rm -f "$PLIST"
  echo "  Removed LaunchAgent"
fi

# ── Remove app bundle ────────────────────────────────────
if [ -d "/Applications/PlugMyAI.app" ]; then
  rm -rf "/Applications/PlugMyAI.app"
  echo "  Removed /Applications/PlugMyAI.app"
fi

# Also check user Applications folder
if [ -d "$HOME/Applications/PlugMyAI.app" ]; then
  rm -rf "$HOME/Applications/PlugMyAI.app"
  echo "  Removed ~/Applications/PlugMyAI.app"
fi

# ── Unregister URL scheme ──────────────────────────────
# macOS Launch Services caches the plugmyai:// handler even after app deletion.
# Unregister both common locations so the URL scheme stops triggering a ghost launch.
LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
if [ -x "$LSREGISTER" ]; then
  $LSREGISTER -u "/Applications/PlugMyAI.app" 2>/dev/null || true
  $LSREGISTER -u "$HOME/Applications/PlugMyAI.app" 2>/dev/null || true
  echo "  Unregistered URL scheme from Launch Services"
fi

# ── Remove CLI binary ────────────────────────────────────
if [ -f "$HOME/.plug-my-ai/bin/plug-my-ai" ]; then
  rm -f "$HOME/.plug-my-ai/bin/plug-my-ai"
  echo "  Removed CLI binary"
fi

# ── Remove data directory ────────────────────────────────
if [ -d "$HOME/.plug-my-ai" ]; then
  rm -rf "$HOME/.plug-my-ai"
  echo "  Removed ~/.plug-my-ai (config + data)"
fi

# ── Clean up PATH from shell profile ─────────────────────
for PROFILE in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.config/fish/config.fish"; do
  if [ -f "$PROFILE" ] && grep -q "plug-my-ai" "$PROFILE" 2>/dev/null; then
    sed -i.bak '/plug-my-ai/d' "$PROFILE"
    rm -f "${PROFILE}.bak"
    echo "  Cleaned PATH from $(basename "$PROFILE")"
  fi
done

echo ""
echo "Uninstalled. Thanks for trying plug-my-ai!"
