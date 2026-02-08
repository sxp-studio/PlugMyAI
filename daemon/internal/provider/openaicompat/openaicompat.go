package openaicompat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"plugmyai/internal/provider"
)

// Config holds config for an OpenAI-compatible API endpoint.
type Config struct {
	BaseURL string `json:"base_url,omitempty"` // defaults to http://localhost:11434/v1
	APIKey  string `json:"api_key,omitempty"`  // optional
}

func init() {
	provider.RegisterFactory("openai-compat", Factory)
}

// Factory creates an OpenAI-compatible provider from raw JSON config.
func Factory(rawConfig json.RawMessage) (provider.Provider, error) {
	var cfg Config
	if rawConfig != nil {
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("parsing openai-compat config: %w", err)
		}
	}
	return New(cfg.BaseURL, cfg.APIKey), nil
}

// Provider routes requests to an OpenAI-compatible HTTP API.
type Provider struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func New(baseURL, apiKey string) *Provider {
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Provider{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *Provider) ID() string   { return "openai-compat" }
func (p *Provider) Name() string { return "OpenAI Compatible" }

func (p *Provider) Available() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return false
	}
	p.setAuth(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// modelsResponse matches the OpenAI GET /models response format.
type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (p *Provider) Models() []provider.Model {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil
	}
	p.setAuth(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var mr modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return nil
	}

	models := make([]provider.Model, 0, len(mr.Data))
	for _, m := range mr.Data {
		models = append(models, provider.Model{
			ID:       m.ID,
			Name:     m.ID,
			Provider: "openai-compat",
		})
	}
	return models
}

// sseChunk matches the OpenAI streaming chunk format.
type sseChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *provider.Usage `json:"usage,omitempty"`
}

func (p *Provider) Complete(ctx context.Context, req *provider.ChatCompletionRequest) (<-chan provider.ChatCompletionChunk, error) {
	// Always stream from the upstream API.
	body := struct {
		Model       string             `json:"model"`
		Messages    []provider.Message `json:"messages"`
		Stream      bool               `json:"stream"`
		Temperature *float64           `json:"temperature,omitempty"`
		MaxTokens   *int               `json:"max_tokens,omitempty"`
	}{
		Model:       req.Model,
		Messages:    req.Messages,
		Stream:      true,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	p.setAuth(httpReq)

	// Use a client without the default timeout for streaming.
	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("upstream API error %d: %s", resp.StatusCode, string(errBody))
	}

	ch := make(chan provider.ChatCompletionChunk, 32)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				select {
				case ch <- provider.ChatCompletionChunk{Done: true, FinishReason: "stop"}:
				case <-ctx.Done():
				}
				return
			}

			var chunk sseChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]
				out := provider.ChatCompletionChunk{Content: choice.Delta.Content}
				if choice.FinishReason != nil {
					out.Done = true
					out.FinishReason = *choice.FinishReason
					out.Usage = chunk.Usage
				}
				select {
				case ch <- out:
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

func (p *Provider) setAuth(req *http.Request) {
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
}
