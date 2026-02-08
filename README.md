# plug-my-ai

Local AI proxy daemon that lets you route your AI subscriptions to any app via an OpenAI-compatible API on `localhost:21110`.

Install the daemon, pair third-party apps through a browser approval flow, and they can use your Claude, OpenAI, Ollama (etc.) subscriptions without needing their own API keys.

**Free and open source (MIT).**

## Install

**macOS app** (menu bar, auto-updates):

Download from [Releases](https://github.com/sxp-studio/plug-my-ai/releases/latest)

**CLI** (any platform):

```sh
curl -fsSL https://plugmy.ai/install.sh | sh
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Your Machine                                           │
│                                                         │
│  ┌──────────────┐    localhost:21110    ┌─────────────┐ │
│  │  3rd-party   │ ──── /v1/chat ─────▶ │   plug-my-ai│ │
│  │  app (any)   │ ◀── streaming SSE ── │   daemon    │──┼──▶ Claude CLI
│  └──────────────┘                      │             │──┼──▶ OpenAI API
│                                        │             │──┼──▶ Ollama
│  ┌──────────────┐                      │  ┌────────┐ │ │
│  │   Browser     │ ◀── dashboard ───── │  │ SPA UI │ │ │
│  └──────────────┘                      │  └────────┘ │ │
│                                        └─────────────┘ │
│  ┌──────────────┐                                       │
│  │ macOS menu   │ ── manages lifecycle, status polling  │
│  │ bar app      │                                       │
│  └──────────────┘                                       │
└─────────────────────────────────────────────────────────┘
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Daemon | Go 1.23 — net/http, SQLite (modernc.org/sqlite), systray (fyne.io/systray) |
| Dashboard | Svelte 5 + Vite 6 — embedded into Go binary via `go:embed` |
| macOS app | Swift 5.9 — NSStatusItem menu bar app, Sparkle 2 auto-updates |
| Website | Static HTML/CSS/JS — hosted on plugmy.ai via xmit |
| Build | GNU Make — root Makefile orchestrates all subsystems |

## Project Structure

```
plug-my-ai/
├── daemon/                 # Go daemon (API server + providers)
├── daemon-ui/
│   ├── dashboard/          # Svelte 5 SPA (admin dashboard)
│   └── toolbar/macOS/      # Native macOS menu bar app
├── website/                # Marketing site (plugmy.ai)
├── Makefile                # Top-level build orchestration
└── install.sh              # User install script
```

## Quick Start

```sh
# Build everything (dashboard + daemon)
make build

# Run in development mode (no system tray)
make dev

# Build macOS menu bar app (.app bundle with embedded daemon)
make macos-app
```

The daemon runs on `http://localhost:21110`. Open it in a browser to access the dashboard.

## Key Concepts

**Providers** — Backend AI services (Claude Code, OpenAI, Ollama, etc.). The daemon routes requests to the right provider based on the model name. See [daemon/README.md](daemon/README.md).

**Pairing** — Apps register via `POST /v1/connect`, which opens a browser approval page. Once approved, the app receives a bearer token (`pma_xxx`). See [daemon/README.md](daemon/README.md).

**Dashboard** — Embedded web UI for managing providers, viewing paired apps, and browsing request history. See [daemon-ui/dashboard/README.md](daemon-ui/dashboard/README.md).

**macOS App** — Native menu bar wrapper that manages the daemon lifecycle (subprocess or launchd), with Sparkle auto-updates. See [daemon-ui/toolbar/macOS/README.md](daemon-ui/toolbar/macOS/README.md).

## Build Targets

| Target | Description |
|--------|-------------|
| `make build` | Build dashboard + daemon binary to `bin/` |
| `make dashboard` | Build Svelte UI and copy to embed directory |
| `make daemon` | Compile Go binary (requires dashboard built first) |
| `make daemon-only` | Rebuild daemon without rebuilding dashboard |
| `make dev` | Build and run with `--no-tray` |
| `make macos-app` | Build macOS .app bundle (dashboard + universal binary + Swift app) |
| `make build-all` | Cross-compile for darwin/linux arm64/amd64 |
| `make clean` | Remove all build artifacts |

## Configuration

- **Config file:** `~/.plug-my-ai/config.json`
- **Database:** `~/.plug-my-ai/data.db` (SQLite)
- **Default port:** 21110

## Documentation

- [Daemon](daemon/README.md) — API server, providers, auth, storage
- [Dashboard](daemon-ui/dashboard/README.md) — Svelte 5 admin UI
- [macOS App](daemon-ui/toolbar/macOS/README.md) — Menu bar app and daemon management
- [Auto-Update](daemon-ui/toolbar/macOS/AUTO-UPDATE.md) — Sparkle 2 update system and release process
- [Website](website/README.md) — Marketing site (plugmy.ai)
- [Releasing](RELEASING.md) — How to cut a release
