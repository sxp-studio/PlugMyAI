# macOS Menu Bar App

Native Swift app that lives in the macOS menu bar. Manages the Go daemon lifecycle (as a subprocess or via launchd) and provides quick access to the dashboard.

## Structure

```
toolbar/macOS/
├── PlugMyAI/
│   ├── AppDelegate.swift           # App entry point — wires everything together
│   ├── StatusBarController.swift   # Menu bar icon + dropdown menu
│   ├── DaemonManager.swift         # Daemon subprocess lifecycle (start/stop/restart)
│   ├── LaunchAgentManager.swift    # ~/Library/LaunchAgents plist management
│   ├── StatusPoller.swift          # Polls GET /v1/status every 5s
│   ├── DaemonStatus.swift          # Status response model
│   ├── UpdaterManager.swift        # Sparkle 2 auto-update delegate
│   └── Resources/
│       ├── Info.plist              # Bundle metadata + Sparkle keys
│       ├── PlugMyAI.entitlements   # Sandbox disabled (needs subprocess, fs access)
│       └── Assets.xcassets         # App icon + menu bar icon
├── project.yml                     # XcodeGen project spec
├── Makefile                        # Build, sign, notarize, release
└── AUTO-UPDATE.md                  # Sparkle update system docs
```

## Building

```sh
# From this directory
make bundle                    # Generate project + build app + bundle daemon

# Or from project root
make macos-app
```

**Prerequisites:** Xcode, [XcodeGen](https://github.com/yonaskolb/XcodeGen) (`brew install xcodegen`), Go.

The build pipeline:
1. `xcodegen generate` — generates `.xcodeproj` from `project.yml`
2. `xcodebuild` — compiles Swift app (resolves Sparkle via SPM)
3. Go daemon built as universal binary (arm64 + amd64 via `lipo`)
4. Binary copied into `.app/Contents/Resources/`

## How It Works

### Menu Bar

The app creates an `NSStatusItem` with a brain icon. The dropdown menu shows:

- **Status** — `● Running — N provider(s)` or `○ Daemon unreachable`
- **Open Dashboard** (Cmd+D) — opens `http://localhost:21110` in browser
- **Check for Updates...** — triggers Sparkle update check
- **Start at Login** — toggles LaunchAgent (checkmark when active)
- **Quit** (Cmd+Q)

### Daemon Lifecycle

On launch, the app checks if the daemon is already reachable at `localhost:21110`:

- **Reachable** — Monitors only (daemon is managed externally or by launchd)
- **Unreachable + LaunchAgent installed** — Re-bootstraps launchd (handles post-update relaunch)
- **Unreachable + no LaunchAgent** — Launches daemon as a subprocess with auto-restart on crash (2s delay)

On quit, the app only terminates the daemon if it was launched as a subprocess. LaunchAgent-managed daemons keep running.

### Start at Login (LaunchAgent)

Toggling "Start at Login" creates or removes `~/Library/LaunchAgents/ai.plugmy.daemon.plist`. When installed, launchd runs the daemon at login with `KeepAlive: true` so it auto-restarts on crash.

### Auto-Update

Uses Sparkle 2 to check `https://plugmy.ai/appcast.xml` for updates. Before installing an update, the `UpdaterManager` delegate stops the daemon subprocess and bootouts the LaunchAgent (without deleting the plist). After the update replaces the `.app` and relaunches, the daemon manager detects the preserved plist and re-bootstraps launchd with the new binary path.

See [AUTO-UPDATE.md](AUTO-UPDATE.md) for the full update lifecycle and release process.

## Distribution

```sh
# Sign, create DMG, notarize, and generate appcast
make release VERSION=0.2.0 IDENTITY="Developer ID Application: ..."
```

The `release` target runs the full pipeline: version bump, build, code sign, DMG creation, Apple notarization, and appcast generation. See [AUTO-UPDATE.md](AUTO-UPDATE.md) for details.

## Configuration

| Setting | Value |
|---------|-------|
| Bundle ID | `ai.plugmy.app` |
| Deployment target | macOS 13.0 |
| Sandbox | Disabled (needs subprocess + file system access) |
| Hardened runtime | Enabled |
| LSUIElement | true (no dock icon) |
| Default port | 21110 |
