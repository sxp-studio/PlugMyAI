package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"plugmyai/internal/provider"
)

// Config holds config specific to the Claude Code CLI provider.
type Config struct {
	CLIPath string `json:"cli_path,omitempty"` // defaults to "claude" in PATH
	Model   string `json:"model,omitempty"`    // defaults to whatever claude uses
}

func init() {
	provider.RegisterFactory("claude-code", Factory)
}

// Factory creates a Claude Code provider from raw JSON config.
func Factory(rawConfig json.RawMessage) (provider.Provider, error) {
	var cfg Config
	if rawConfig != nil {
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("parsing claude-code config: %w", err)
		}
	}
	return New(cfg.CLIPath, cfg.Model), nil
}

// Provider routes requests through the Claude Code CLI.
// No API key needed — uses the user's existing Claude Code OAuth session.
type Provider struct {
	cliPath string
	model   string
}

// cliMessage represents a line of NDJSON output from `claude --output-format stream-json --verbose`.
//
// Key message types from the CLI:
//   - "assistant": full response in message.content[].text
//   - "content_block_delta": incremental text in delta.text (may appear for longer responses)
//   - "result": final summary with result text + usage stats
//   - "error": error description in content
//   - "system": hooks/init info (ignored)
type cliMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"` // used by error messages
	Result  string `json:"result"`  // full text on type=result

	// assistant message envelope
	Message *struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message,omitempty"`

	// content_block_delta incremental text
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta,omitempty"`

	// Usage (top-level on result messages)
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

func New(cliPath, model string) *Provider {
	if cliPath == "" {
		cliPath = "claude"
	}
	return &Provider{cliPath: cliPath, model: model}
}

func (p *Provider) ID() string   { return "claude-code" }
func (p *Provider) Name() string { return "Claude Code" }

func (p *Provider) Available() bool {
	path, err := exec.LookPath(p.cliPath)
	return err == nil && path != ""
}

func (p *Provider) Models() []provider.Model {
	models := []provider.Model{
		{ID: "claude", Name: "Claude (default)", Provider: "claude-code"},
	}
	if p.model != "" {
		models = append(models, provider.Model{
			ID:       p.model,
			Name:     p.model,
			Provider: "claude-code",
		})
	}
	return models
}

func (p *Provider) Complete(ctx context.Context, req *provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
	prompt := buildPrompt(req.Messages)

	args := []string{
		"-p", prompt,
		"--output-format", "stream-json",
		"--verbose",
	}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}
	// Chat scope: disable all tools (LLM-only, no filesystem/shell access)
	if req.Scope == "" || req.Scope == "chat" {
		args = append(args, "--allowedTools", "")
	}

	cmd := exec.CommandContext(ctx, p.cliPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting claude CLI: %w", err)
	}

	ch := make(chan provider.ChatCompletionChunk, 32)

	go func() {
		defer close(ch)
		defer cmd.Wait()

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for long lines

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var msg cliMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				// Skip unparseable lines
				continue
			}

			chunk := parseMessage(msg)
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

func parseMessage(msg cliMessage) *provider.ChatCompletionChunk {
	switch msg.Type {
	case "assistant":
		// Content is nested: message.content[].text
		if msg.Message != nil {
			var text string
			for _, block := range msg.Message.Content {
				if block.Type == "text" {
					text += block.Text
				}
			}
			if text != "" {
				return &provider.ChatCompletionChunk{Content: text}
			}
		}
		return nil

	case "content_block_delta":
		// Incremental text in delta.text (may appear for longer responses)
		if msg.Delta != nil && msg.Delta.Text != "" {
			return &provider.ChatCompletionChunk{Content: msg.Delta.Text}
		}
		return nil

	case "result":
		chunk := &provider.ChatCompletionChunk{
			Done:         true,
			FinishReason: "stop",
		}
		if msg.Usage != nil {
			chunk.Usage = &provider.Usage{
				PromptTokens:     msg.Usage.InputTokens,
				CompletionTokens: msg.Usage.OutputTokens,
				TotalTokens:      msg.Usage.InputTokens + msg.Usage.OutputTokens,
			}
		}
		return chunk

	case "error":
		return &provider.ChatCompletionChunk{
			Error: fmt.Errorf("claude CLI error: %s", msg.Content),
		}

	default:
		// system, content_block_start, content_block_stop, etc. — skip
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
