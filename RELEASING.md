# Releasing

## One command

```sh
./release.sh 0.2.0
```

This runs the full pipeline:

1. **Build CLI binaries** — cross-compiled for darwin/linux, arm64/amd64
2. **Build macOS app** — Swift app + universal Go binary → signed DMG → Apple notarization
3. **Checksums** — generates `website/releases/SHA256SUMS`, commits and pushes
4. **GitHub Release** — uploads DMG + CLI binaries to `v0.2.0` release
5. **Deploy website** — publishes appcast + site to plugmy.ai via xmit

## Prerequisites

- Go (`/opt/homebrew/bin/go`)
- Xcode + command line tools
- `gh` CLI, authenticated (`gh auth login`)
- `xmit` CLI (for website deploy)
- Apple Developer signing identity (for notarization)
- Clean git working tree

## What goes where

| Artifact | Location | In git? |
|----------|----------|---------|
| DMG (download) | GitHub Releases | No |
| CLI binaries | GitHub Releases | No |
| SHA256SUMS | `website/releases/SHA256SUMS` | Yes |
| Sparkle appcast | `website/appcast.xml` (via xmit) | No |
| DMG (for appcast generation) | `website/releases/*.dmg` | No (gitignored) |
| install.sh | `website/install.sh` (via xmit) | Yes |

## Manual steps (if needed)

Individual Makefile targets can be run separately:

```sh
make build-all                                            # CLI binaries only
make -C daemon-ui/toolbar/macOS dmg VERSION=0.2.0         # DMG only (skip notarize)
make -C daemon-ui/toolbar/macOS gh-release VERSION=0.2.0  # GitHub upload only
make -C daemon-ui/toolbar/macOS checksums                 # regenerate SHA256SUMS
cd website && ./deploy.sh                                 # website deploy only
```

## Updating the website version

When bumping versions, update these in `website/index.html`:
- Nav version badge (`v0.x`)
- Download button URL (GitHub Release tag)
- Verify section (`shasum` command filename)
