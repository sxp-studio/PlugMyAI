#!/bin/sh
#
# Release script for plug-my-ai
# Usage: ./release.sh <version>          — full pipeline
#        ./release.sh <version> notarize — retry notarization + publish only
#
set -e

VERSION="$1"
STEP="${2:-all}"

if [ -z "$VERSION" ]; then
  echo "Usage: ./release.sh <version> [notarize]"
  echo "  ./release.sh 0.2.0           # full release"
  echo "  ./release.sh 0.2.0 notarize  # retry notarize + publish (after build)"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
TOOLBAR_DIR="$ROOT_DIR/daemon-ui/toolbar/macOS"
WEBSITE_DIR="$ROOT_DIR/website"
IDENTITY="Developer ID Application: SXP Studio (794DNYA4B8)"

echo "═══════════════════════════════════════"
echo "  plug-my-ai release v$VERSION"
echo "═══════════════════════════════════════"
echo ""

# ── Preflight checks ─────────────────────────────────────
echo "Preflight checks..."

command -v go >/dev/null 2>&1 || { echo "Error: go not found"; exit 1; }
command -v gh >/dev/null 2>&1 || { echo "Error: gh CLI not found (brew install gh)"; exit 1; }
command -v xcodebuild >/dev/null 2>&1 || { echo "Error: xcodebuild not found"; exit 1; }
command -v xmit >/dev/null 2>&1 || { echo "Error: xmit not found"; exit 1; }

gh auth status >/dev/null 2>&1 || { echo "Error: gh not authenticated (run: gh auth login)"; exit 1; }

security find-identity -v -p codesigning | grep -q "Developer ID Application" || {
  echo "Error: No 'Developer ID Application' certificate found in keychain"
  exit 1
}

xcrun notarytool history --keychain-profile "notarytool-profile" >/dev/null 2>&1 || {
  echo "Error: notarytool-profile not found (run: xcrun notarytool store-credentials \"notarytool-profile\")"
  exit 1
}

echo "  All checks passed."
echo ""

if [ "$STEP" = "all" ]; then
  # ── Step 1: Build CLI binaries ───────────────────────────
  echo "Step 1/6: Building CLI binaries (all platforms)..."
  make -C "$ROOT_DIR" build-all
  echo ""

  # ── Step 2: Build macOS app + DMG ────────────────────────
  echo "Step 2/6: Building macOS app + DMG..."
  make -C "$TOOLBAR_DIR" clean
  make -C "$TOOLBAR_DIR" bump-version VERSION="$VERSION"
  make -C "$TOOLBAR_DIR" bundle IDENTITY="$IDENTITY"
  make -C "$TOOLBAR_DIR" sign IDENTITY="$IDENTITY"
  make -C "$TOOLBAR_DIR" dmg IDENTITY="$IDENTITY"
  echo ""
fi

# ── Step 3: Notarize ──────────────────────────────────────
DMG="$TOOLBAR_DIR/build/PlugMyAI.dmg"
if [ ! -f "$DMG" ]; then
  echo "Error: DMG not found at $DMG — run full release first"
  exit 1
fi

echo "Step 3/6: Notarizing (this can take 5-15 min)..."
xcrun notarytool submit "$DMG" \
  --keychain-profile "notarytool-profile" \
  --wait
xcrun stapler staple "$DMG"
echo "  ✓ Notarized and stapled"
echo ""

# ── Step 4: Checksums + appcast ───────────────────────────
echo "Step 4/6: Checksums + appcast..."
make -C "$TOOLBAR_DIR" appcast VERSION="$VERSION"
make -C "$TOOLBAR_DIR" checksums
echo ""

# ── Step 5: Commit, GitHub Release, deploy ────────────────
echo "Step 5/6: Publishing..."
git -C "$ROOT_DIR" add -A
git -C "$ROOT_DIR" commit -m "Release v$VERSION" || true
git -C "$ROOT_DIR" push

make -C "$TOOLBAR_DIR" gh-release VERSION="$VERSION"

cd "$WEBSITE_DIR" && xmit plugmy.ai
echo ""

# ── Done ──────────────────────────────────────────────────
echo "═══════════════════════════════════════"
echo "  v$VERSION released!"
echo "═══════════════════════════════════════"
echo ""
echo "  GitHub:  https://github.com/sxp-studio/PlugMyAI/releases/tag/v$VERSION"
echo "  Website: https://plugmy.ai"
echo ""
