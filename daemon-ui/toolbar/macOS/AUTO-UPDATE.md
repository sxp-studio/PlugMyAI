# Auto-Update (Sparkle 2)

The macOS menu bar app uses [Sparkle 2](https://sparkle-project.org/) to check for and install updates from a static appcast hosted on plugmy.ai.

## How It Works

1. Sparkle periodically fetches `https://plugmy.ai/appcast.xml`
2. If a newer version exists, it shows a native update dialog
3. User clicks Install → Sparkle downloads the DMG
4. Before replacing the app, our `UpdaterManager` delegate:
   - Stops the daemon subprocess (if running)
   - Bootouts the LaunchAgent (if installed) without deleting the plist
5. Sparkle replaces the `.app` bundle and relaunches
6. On relaunch, `DaemonManager.start()` detects the LaunchAgent plist still exists but daemon is unreachable → re-bootstraps launchd with the new binary path

## Signing Keys

Sparkle uses EdDSA (ed25519) to verify update authenticity.

- **Public key** — stored in `Info.plist` under `SUPublicEDKey`, safe to commit
- **Private key** — stored in the macOS Keychain (placed there by `generate_keys`), never a file on disk

If you need to regenerate keys (new machine, lost Keychain):

```sh
# From the macOS toolbar directory
build/SourcePackages/artifacts/sparkle/Sparkle/bin/generate_keys
```

Then update `SUPublicEDKey` in `PlugMyAI/Resources/Info.plist` and re-sign all future releases with the new key. Existing installs won't be able to verify updates signed with the new key — users would need to re-download manually once.

## Releasing an Update

### One command

```sh
make release VERSION=0.2.0 IDENTITY="Developer ID Application: Your Name (TEAMID)"
```

This runs: `bump-version` → `bundle` → `sign` → `dmg` → `notarize` → `appcast`.

### Step by step

```sh
# 1. Bump version in Info.plist + Go daemon
make bump-version VERSION=0.2.0

# 2. Build app + universal Go binary
make bundle IDENTITY="Developer ID Application: ..."

# 3. Code sign for distribution
make sign IDENTITY="Developer ID Application: ..."

# 4. Create DMG
make dmg

# 5. Notarize with Apple
make notarize

# 6. Copy DMG to website/releases/ and regenerate appcast.xml
make appcast VERSION=0.2.0
```

### Deploy

Copy `website/releases/PlugMyAI-0.2.0.dmg` and `website/appcast.xml` to plugmy.ai's static hosting. Older app versions will detect the update on their next check.

## Files Involved

| File | Role |
|------|------|
| `UpdaterManager.swift` | Sparkle delegate — pre-update daemon shutdown hooks |
| `Info.plist` | `SUFeedURL`, `SUPublicEDKey`, `SUEnableAutomaticChecks` |
| `AppDelegate.swift` | Wires `UpdaterManager` with daemon/launchd closures |
| `StatusBarController.swift` | "Check for Updates…" menu item |
| `LaunchAgentManager.swift` | `bootoutOnly()` / `bootstrapIfInstalled()` for update lifecycle |
| `DaemonManager.swift` | Post-update detection and launchd re-bootstrap |
| `Makefile` | `bump-version`, `appcast`, `release` targets |
| `project.yml` | Sparkle SPM dependency |
