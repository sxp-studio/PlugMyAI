package provider

import (
	"context"
	"encoding/json"
	"fmt"
)

// Provider is the interface all AI backends must implement.
type Provider interface {
	// ID returns a unique identifier for this provider (e.g. "claude-code").
	ID() string

	// Name returns a human-readable name (e.g. "Claude Code").
	Name() string

	// Available returns whether this provider is ready to serve requests.
	// For Claude Code, this checks if the CLI is installed and authenticated.
	Available() bool

	// Models returns the list of model IDs this provider supports.
	Models() []Model

	// Complete sends a chat completion request and streams chunks back.
	// The caller must consume the channel until it's closed.
	Complete(ctx context.Context, req *ChatCompletionRequest) (<-chan ChatCompletionChunk, error)
}

type Model struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
	Scope       string    `json:"-"` // "chat" or "full" — set by server, not from JSON body
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionChunk struct {
	Content      string // text delta
	Done         bool   // true when stream is finished
	FinishReason string // "stop", "length", etc. (only set when Done)
	Usage        *Usage // only set when Done
	Error        error  // non-nil if something went wrong
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Registry holds all configured providers.
type Registry struct {
	providers []Provider
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Register(p Provider) {
	r.providers = append(r.providers, p)
}

func (r *Registry) All() []Provider {
	return r.providers
}

func (r *Registry) Available() []Provider {
	var out []Provider
	for _, p := range r.providers {
		if p.Available() {
			out = append(out, p)
		}
	}
	return out
}

func (r *Registry) AllModels() []Model {
	var models []Model
	for _, p := range r.providers {
		if p.Available() {
			models = append(models, p.Models()...)
		}
	}
	return models
}

// FindByID returns the provider with the given ID, or nil if not found.
func (r *Registry) FindByID(id string) Provider {
	for _, p := range r.providers {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// FindProvider returns the first available provider that serves the given model.
func (r *Registry) FindProvider(model string) Provider {
	for _, p := range r.providers {
		if !p.Available() {
			continue
		}
		for _, m := range p.Models() {
			if m.ID == model {
				return p
			}
		}
	}
	// Fallback: if model not found, return first available provider
	avail := r.Available()
	if len(avail) > 0 {
		return avail[0]
	}
	return nil
}

// --- Factory registry for plug-and-play providers ---

// FactoryFunc creates a Provider from raw JSON config.
type FactoryFunc func(rawConfig json.RawMessage) (Provider, error)

var factories = map[string]FactoryFunc{}

// RegisterFactory registers a provider factory under the given type name.
// Typically called from a provider package's init() function.
func RegisterFactory(typeName string, fn FactoryFunc) {
	factories[typeName] = fn
}

// CreateProvider looks up a registered factory and creates a Provider.
func CreateProvider(typeName string, rawConfig json.RawMessage) (Provider, error) {
	fn, ok := factories[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown provider type: %s", typeName)
	}
	return fn(rawConfig)
}
