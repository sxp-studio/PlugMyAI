# Daemon

Go HTTP server that exposes an OpenAI-compatible API on `localhost:21110`, routing requests to configured AI providers.

## Structure

```
daemon/
├── cmd/plug-my-ai/
│   └── main.go              # Entry point — config, DB, providers, server, tray
├── internal/
│   ├── server/
│   │   ├── server.go        # HTTP server setup, routing, SPA serving
│   │   ├── handlers.go      # All API endpoint handlers
│   │   ├── middleware.go     # CORS + bearer token auth
│   │   └── browser.go       # Cross-platform browser launcher
│   ├── provider/
│   │   ├── provider.go      # Provider interface + registry
│   │   └── claude/
│   │       └── claude.go    # Claude Code CLI provider
│   ├── config/
│   │   └── config.go        # JSON config loading/generation
│   ├── store/
│   │   └── store.go         # SQLite (apps, history, connect requests)
│   ├── tray/
│   │   └── tray.go          # System tray (fyne.io/systray)
│   └── dashboard/
│       ├── embed.go         # go:embed for SPA assets
│       └── dist/            # Built Svelte dashboard (copied at build time)
├── go.mod
└── go.sum
```

## Running

```sh
# From project root
make dev

# Or directly
cd daemon && go build -o ../bin/plug-my-ai ./cmd/plug-my-ai
../bin/plug-my-ai --no-tray

# Create default config without starting
../bin/plug-my-ai init
```

**Flags:**
- `-config <path>` — Config file (default: `~/.plug-my-ai/config.json`)
- `-port <int>` — Override port (default: 21110)
- `-no-tray` — Run without system tray (headless / development)

## API

All endpoints follow OpenAI's API format. Base URL: `http://localhost:21110`.

### Public (no auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/status` | Health check — version, uptime, provider list |
| POST | `/v1/connect` | Start pairing flow — returns request ID + approve URL |
| GET | `/v1/connect/{id}` | Poll pairing status — returns token when approved |
| POST | `/v1/connect/{id}/approve` | Approve pairing (browser-initiated) |
| POST | `/v1/connect/{id}/deny` | Deny pairing |
| GET | `/v1/connect/pending` | List pending pairing requests |
| GET | `/` | Dashboard SPA (admin token injected into HTML) |

### App or Admin auth required

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/models` | List available models (OpenAI format) |
| POST | `/v1/chat/completions` | Chat completion — streaming SSE or JSON |

### Admin auth only

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/history` | Request log (paginated: `?limit=&offset=`) |
| GET | `/v1/history/{id}` | Single history entry |
| GET | `/v1/providers` | List providers + availability |
| GET | `/v1/apps` | List paired apps (tokens redacted) |
| DELETE | `/v1/apps/{id}` | Revoke an app's token |

### Authentication

Tokens are passed as `Authorization: Bearer <token>` or `?token=<token>` (for SSE).

- **Admin token** — `pma_admin_` + 48 hex chars. Generated on first run. Stored in config. Full access.
- **App tokens** — `pma_` + 48 hex chars. Issued via pairing flow. Access to `/v1/models` and `/v1/chat/completions`.

## Pairing Flow

```
Third-party app                          Daemon                          Browser
     │                                     │                               │
     ├─ POST /v1/connect ────────────────▶ │                               │
     │  { app_name, app_url }              │                               │
     │                                     ├── opens approval page ──────▶ │
     ◀── { request_id, approve_url } ─────┤                               │
     │                                     │                               │
     │  (polls)                            │         User clicks Approve   │
     ├─ GET /v1/connect/{id} ────────────▶ │ ◀── POST .../approve ────────┤
     │                                     │                               │
     ◀── { status: "approved", token } ───┤                               │
     │                                     │                               │
     ├─ POST /v1/chat/completions ───────▶ │                               │
     │  Authorization: Bearer pma_xxx      │                               │
```

Connect requests expire after 5 minutes.

## Providers

Providers implement a common interface:

```go
type Provider interface {
    ID() string
    Name() string
    Available() bool
    Models() []Model
    Complete(ctx context.Context, req *ChatCompletionRequest) (<-chan ChatCompletionChunk, error)
}
```

### Claude Code

The default provider. Spawns `claude -p "prompt" --output-format stream-json` as a subprocess and streams the NDJSON output back as OpenAI-compatible chunks.

- **Availability:** Checks `claude` is in PATH via `exec.LookPath`
- **Auth:** Uses the user's existing Claude CLI session (no API key needed)
- **Models:** Exposes `claude` (default) + optional configured model override

### Adding Providers

Providers are plug-and-play. Create a single package that self-registers via `init()` — no changes needed to the core code except one blank import in `main.go`.

See [`internal/provider/CONTRIBUTING.md`](internal/provider/CONTRIBUTING.md) for a step-by-step guide with a full skeleton.

## Storage

SQLite database at `~/.plug-my-ai/data.db` (WAL mode).

| Table | Purpose |
|-------|---------|
| `apps` | Paired applications — name, URL, token, revoked flag |
| `history` | Request log — model, messages, response, tokens, duration |
| `connect_requests` | Pairing requests — status, expiry, generated token |

## Configuration

`~/.plug-my-ai/config.json`:

```json
{
  "port": 21110,
  "admin_token": "pma_admin_<48 hex chars>",
  "data_dir": "~/.plug-my-ai",
  "providers": [
    {
      "type": "claude-code",
      "name": "Claude Code",
      "enabled": true,
      "config": {
        "cli_path": "claude",
        "model": ""
      }
    }
  ]
}
```
