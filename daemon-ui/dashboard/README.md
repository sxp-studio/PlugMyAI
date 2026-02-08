# Dashboard

Svelte 5 single-page app that serves as the admin interface for the daemon. Embedded into the Go binary at build time via `go:embed` and served at `http://localhost:21110`.

## Structure

```
dashboard/
├── src/
│   ├── main.js              # Mounts App to #app
│   ├── App.svelte            # Hash router + sidebar layout
│   ├── app.css               # Global styles (dark theme)
│   ├── pages/
│   │   ├── Overview.svelte   # Status summary, providers, recent requests
│   │   ├── Providers.svelte  # Provider list with availability status
│   │   ├── Apps.svelte       # Paired apps — view, revoke tokens
│   │   ├── History.svelte    # Request log with search + pagination
│   │   └── Approve.svelte    # Pairing approval flow (browser-facing)
│   └── lib/
│       ├── api.js            # HTTP client for daemon API
│       └── format.js         # Date, duration, token formatting
├── index.html                # Shell HTML (emoji favicon, mounts #app)
├── package.json
├── vite.config.js
└── svelte.config.js
```

## Development

```sh
cd daemon-ui/dashboard
npm install
npm run dev       # Vite dev server with HMR
npm run build     # Production build to dist/
npm run preview   # Preview production build
```

During development, the Vite dev server proxies API requests. In production, the built `dist/` is copied to `daemon/internal/dashboard/dist/` and embedded in the Go binary.

## Pages

**Overview** — Dashboard home. Shows daemon version, uptime, provider count, connected apps, and a table of the 5 most recent requests. Fetches `/v1/status`, `/v1/providers`, `/v1/apps`, and `/v1/history` concurrently.

**Providers** — Lists all configured AI providers. Displays name, type, availability badge, and model count. Read-only (provider config lives in `config.json`).

**Apps** — Lists paired third-party apps with their name, URL, pairing date, and status. Revoke button removes an app's access via `DELETE /v1/apps/{id}`.

**History** — Paginated request log (20 per page). Searchable by prompt, app name, or model. Expandable rows show full prompt and response. Displays token counts and latency.

**Approve** — Pairing approval page. Reached via `/#/approve?req=<id>` when a third-party app initiates pairing. Shows the requesting app's name, URL, and icon. Approve/Deny buttons. Also lists all pending requests in a table.

## API Client

`lib/api.js` wraps all daemon API calls. Auth token is read from `localStorage.pma_admin_token` — the daemon injects this into the served HTML at runtime so the dashboard authenticates automatically.

## Tech

- **Svelte 5** runes (`$state`, `$effect`, `$derived`) for reactivity
- **Hash-based routing** (`window.location.hash`) — no server-side routing needed
- **Dark theme** — `#0a0a0a` background, `#3b82f6` accent blue, monospace fonts
- **No component library** — all UI is custom CSS
- **Vite 6** with `@sveltejs/vite-plugin-svelte`
