#!/bin/sh
#
# Release script for plug-my-ai
# Usage: ./release.sh 0.2.0
#
set -e

VERSION="$1"
if [ -z "$VERSION" ]; then
  echo "Usage: ./release.sh <version>"
  echo "Example: ./release.sh 0.2.0"
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

# Check gh is authenticated
gh auth status >/dev/null 2>&1 || { echo "Error: gh not authenticated (run: gh auth login)"; exit 1; }

# Check Developer ID signing identity
security find-identity -v -p codesigning | grep -q "Developer ID Application" || {
  echo "Error: No 'Developer ID Application' certificate found in keychain"
  exit 1
}

# Check notarytool profile
xcrun notarytool history --keychain-profile "notarytool-profile" >/dev/null 2>&1 || {
  echo "Error: notarytool-profile not found (run: xcrun notarytool store-credentials \"notarytool-profile\")"
  exit 1
}

# Check working tree is clean
if [ -n "$(git -C "$ROOT_DIR" status --porcelain)" ]; then
  echo "Error: working tree is dirty — commit or stash changes first"
  exit 1
fi

echo "  All checks passed."
echo ""

# ── Step 1: Build CLI binaries ───────────────────────────
echo "Step 1/6: Building CLI binaries (all platforms)..."
make -C "$ROOT_DIR" build-all
echo ""

# ── Step 2: Build macOS app + DMG + notarize ─────────────
echo "Step 2/6: Building macOS app, DMG, signing, notarizing..."
make -C "$TOOLBAR_DIR" release VERSION="$VERSION" IDENTITY="$IDENTITY"
echo ""

# ── Step 3: Commit checksums ─────────────────────────────
echo "Step 3/6: Committing checksums..."
git -C "$ROOT_DIR" add website/releases/SHA256SUMS
git -C "$ROOT_DIR" commit -m "checksums for v$VERSION"
git -C "$ROOT_DIR" push
echo ""

# ── Step 4: Create GitHub Release ────────────────────────
echo "Step 4/6: Creating GitHub Release..."
make -C "$TOOLBAR_DIR" gh-release VERSION="$VERSION"
echo ""

# ── Step 5: Deploy website ───────────────────────────────
echo "Step 5/6: Deploying website..."
cd "$WEBSITE_DIR" && xmit plugmy.ai
echo ""

# ── Step 6: Tag ──────────────────────────────────────────
echo "Step 6/6: Done."
echo ""
echo "═══════════════════════════════════════"
echo "  v$VERSION released!"
echo "═══════════════════════════════════════"
echo ""
echo "  GitHub:  https://github.com/sxp-studio/PlugMyAI/releases/tag/v$VERSION"
echo "  Website: https://plugmy.ai"
echo ""
