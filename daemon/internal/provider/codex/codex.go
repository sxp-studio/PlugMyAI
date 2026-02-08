package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"plugmyai/internal/provider"
)

// Config holds config specific to the Codex CLI provider.
type Config struct {
	CLIPath string `json:"cli_path,omitempty"` // defaults to "codex" in PATH
	Model   string `json:"model,omitempty"`
}

func init() {
	provider.RegisterFactory("codex", Factory)
}

// Factory creates a Codex provider from raw JSON config.
func Factory(rawConfig json.RawMessage) (provider.Provider, error) {
	var cfg Config
	if rawConfig != nil {
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("parsing codex config: %w", err)
		}
	}
	return New(cfg.CLIPath, cfg.Model), nil
}

// Provider routes requests through the Codex CLI.
type Provider struct {
	cliPath string
	model   string
}

// cliEvent represents a line of NDJSON output from `codex exec --json`.
type cliEvent struct {
	Type string `json:"type"`
	// For response.output_text.delta
	Delta string `json:"delta,omitempty"`
	// For response.completed
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
	// For error events
	Message string `json:"message,omitempty"`
}

func New(cliPath, model string) *Provider {
	if cliPath == "" {
		cliPath = "codex"
	}
	return &Provider{cliPath: cliPath, model: model}
}

func (p *Provider) ID() string   { return "codex" }
func (p *Provider) Name() string { return "Codex" }

func (p *Provider) Available() bool {
	path, err := exec.LookPath(p.cliPath)
	return err == nil && path != ""
}

func (p *Provider) Models() []provider.Model {
	models := []provider.Model{
		{ID: "codex", Name: "Codex (default)", Provider: "codex"},
	}
	if p.model != "" {
		models = append(models, provider.Model{
			ID:       p.model,
			Name:     p.model,
			Provider: "codex",
		})
	}
	return models
}

func (p *Provider) Complete(ctx context.Context, req *provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
	prompt := buildPrompt(req.Messages)

	args := []string{"exec", prompt, "--json"}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}
	// Full scope: grant full auto-approval for tools; chat scope: use default (suggest mode)
	if req.Scope == "full" {
		args = append(args, "--full-auto")
	}

	cmd := exec.CommandContext(ctx, p.cliPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting codex CLI: %w", err)
	}

	ch := make(chan provider.ChatCompletionChunk, 32)

	go func() {
		defer close(ch)
		defer cmd.Wait()

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var evt cliEvent
			if err := json.Unmarshal([]byte(line), &evt); err != nil {
				continue
			}

			chunk := parseEvent(evt)
			if chunk != nil {
				select {
				case ch <- *chunk:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case ch <- provider.ChatCompletionChunk{Error: err}:
			case <-ctx.Done():
			}
		}
	}()

	return ch, nil
}

func parseEvent(evt cliEvent) *provider.ChatCompletionChunk {
	switch evt.Type {
	case "response.output_text.delta":
		if evt.Delta != "" {
			return &provider.ChatCompletionChunk{Content: evt.Delta}
		}
		return nil

	case "response.completed":
		chunk := &provider.ChatCompletionChunk{
			Done:         true,
			FinishReason: "stop",
		}
		if evt.Usage != nil {
			chunk.Usage = &provider.Usage{
				PromptTokens:     evt.Usage.InputTokens,
				CompletionTokens: evt.Usage.OutputTokens,
				TotalTokens:      evt.Usage.InputTokens + evt.Usage.OutputTokens,
			}
		}
		return chunk

	case "error":
		return &provider.ChatCompletionChunk{
			Error: fmt.Errorf("codex CLI error: %s", evt.Message),
		}

	default:
		return nil
	}
}

// buildPrompt converts OpenAI-style messages into a single prompt string for the CLI.
func buildPrompt(messages []provider.Message) string {
	if len(messages) == 1 {
		return messages[0].Content
	}

	var parts []string
	for _, m := range messages {
		switch m.Role {
		case "system":
			parts = append(parts, fmt.Sprintf("[System]\n%s", m.Content))
		case "user":
			parts = append(parts, m.Content)
		case "assistant":
			parts = append(parts, fmt.Sprintf("[Assistant]\n%s", m.Content))
		default:
			parts = append(parts, m.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}
