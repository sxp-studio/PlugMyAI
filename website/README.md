# plug-my-ai Website

Marketing site for plug-my-ai. Live at **https://plugmy.ai**.

## Objective

Pitch plug-my-ai to two audiences:
1. **End users** — install the daemon, connect their AI subscriptions, use them everywhere.
2. **Developers** — add "Plug My AI" support to their apps (web, native, CLI).

## Tech Stack

Plain HTML + CSS + vanilla JS. No framework, no build step.

- **Hosting:** [xmit.co](https://xmit.co) (static file deploy to `plugmy.ai`)
- **Design:** Dark theme matching the daemon dashboard (`#0a0a0a` bg, `#3b82f6` accent)
- **Fonts:** System stack (`-apple-system`, `Segoe UI`, etc.) + monospace for code

## Structure

```
website/
├── index.html     # Landing page (hero, providers, integration demo, install)
├── adopt.html     # Adoption guide (how it works, API docs, native examples, FAQ)
├── style.css      # All styles (shared across both pages)
├── script.js      # Copy-to-clipboard + code tab switching
├── mascot.png     # Plug mascot (transparent PNG, 400x400)
├── deploy.sh      # One-command deploy via xmit
└── README.md      # This file
```

### index.html — Landing Page

| Section | Content |
|---------|---------|
| Hero | Mascot (floating + glow), tagline, CTA buttons, install one-liner |
| How it works | 3-step flow (install → connect → use) |
| Providers | Grid: Claude Code, Claude API, OpenAI, Codex, OpenRouter, Ollama, OpenClaw |
| Integrate | "Plug My AI" button demo + tabbed code (Web/JS and Native/cURL) |
| Install | curl one-liner + Homebrew (coming soon) |
| Demo | Placeholder (TODO) |

### adopt.html — Adoption Guide

Architecture overview, pairing flow walkthrough, chat API docs, OpenAI SDK example, native integration examples (Swift, Python, Rust), best practices, and disclaimers/FAQ.

## Deploy

```sh
cd website
./deploy.sh
```

Runs `xmit plugmy.ai` from the website directory. No build step needed.
