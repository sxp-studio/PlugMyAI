# plug-my-ai — Roadmap

## Up Next

### Security
- [ ] Run AI agents in [Apple containers](https://github.com/apple/container) (sandbox filesystem/network access)

### Providers
- [ ] Ollama auto-detection and pass-through
- [ ] LMStudio support
- [ ] Direct Anthropic API key provider
- [ ] Direct OpenAI API key provider
- [ ] OpenRouter BYOK support

### Dashboard
- [ ] Edit app settings after pairing (scope, providers)
- [ ] Connected apps count in menu bar
- [ ] Start at Login toggle

### More Endpoints
- [ ] POST /v1/embeddings
- [ ] POST /v1/images/generations
- [ ] POST /v1/audio/speech

### Features
- [ ] Token budgeting per app (daily/monthly limits)
- [ ] Rate limiting per app
- [ ] Multiple simultaneous Claude Code sessions
- [ ] Request caching (same prompt = cached response)

### Distribution
- [ ] Homebrew tap
- [ ] Linux systemd service + tray support
- [ ] Windows support

### Reach
- [ ] Tunnel support (expose daemon to iPhone / remote devices)
- [ ] Browser extension (auto-detect daemon, streamline pairing)
