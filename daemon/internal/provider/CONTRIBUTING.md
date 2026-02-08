# Writing a Provider

Add a new AI backend to plug-my-ai by creating a single package. No changes needed outside your package except one blank import.

## Steps

### 1. Create the package

```
daemon/internal/provider/<name>/<name>.go
```

### 2. Implement the `Provider` interface

```go
package mybackend

import (
    "context"
    "encoding/json"
    "fmt"

    "plugmyai/internal/provider"
)

// Config holds provider-specific settings parsed from the user's config.json.
type Config struct {
    APIKey string `json:"api_key"`
    Model  string `json:"model,omitempty"`
}

type Provider struct {
    apiKey string
    model  string
}

func New(apiKey, model string) *Provider {
    return &Provider{apiKey: apiKey, model: model}
}

func (p *Provider) ID() string        { return "mybackend" }
func (p *Provider) Name() string      { return "My Backend" }
func (p *Provider) Available() bool   { return p.apiKey != "" }
func (p *Provider) Models() []provider.Model {
    return []provider.Model{{ID: p.model, Name: p.model, Provider: "mybackend"}}
}

func (p *Provider) Complete(ctx context.Context, req *provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
    ch := make(chan provider.ChatCompletionChunk, 32)
    go func() {
        defer close(ch)
        // Call your backend, send chunks to ch.
        // Set Done: true and FinishReason on the final chunk.
    }()
    return ch, nil
}
```

### 3. Add the factory function and self-register via `init()`

```go
func init() {
    provider.RegisterFactory("mybackend", Factory)
}

func Factory(rawConfig json.RawMessage) (provider.Provider, error) {
    var cfg Config
    if rawConfig != nil {
        if err := json.Unmarshal(rawConfig, &cfg); err != nil {
            return nil, fmt.Errorf("parsing mybackend config: %w", err)
        }
    }
    return New(cfg.APIKey, cfg.Model), nil
}
```

### 4. Add a blank import in `main.go`

In `daemon/cmd/plug-my-ai/main.go`, add your package to the blank import block:

```go
// Provider self-registration via init().
// Add new providers here as blank imports.
_ "plugmyai/internal/provider/claude"
_ "plugmyai/internal/provider/mybackend"
```

### 5. Add a config entry

Users enable your provider in `~/.plug-my-ai/config.json`:

```json
{
  "providers": [
    {
      "type": "mybackend",
      "name": "My Backend",
      "enabled": true,
      "config": {
        "api_key": "sk-...",
        "model": "my-model-v1"
      }
    }
  ]
}
```

## Checklist

- [ ] Package at `daemon/internal/provider/<name>/`
- [ ] Implements all 5 methods of `provider.Provider`
- [ ] `Factory()` parses `json.RawMessage` into your own config struct
- [ ] `init()` calls `provider.RegisterFactory("<name>", Factory)`
- [ ] Blank import added to `main.go`
- [ ] Streaming: send chunks to the channel, close it when done
- [ ] Set `Done: true` and `FinishReason` on the final chunk
- [ ] Populate `Usage` on the final chunk if your backend provides token counts
