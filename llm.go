package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// llmProvider is one LLM backend the digest can talk to. Complete returns the
// assistant's raw text for a system+user pair.
type llmProvider interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

// newLLM builds the configured provider. Provider "" or "none" returns
// (nil, nil) — the digest then refuses to call any AI, which is what makes
// "decide how and whether an AI runs" a pure config concern.
func newLLM(cfg llmConfig) (llmProvider, error) {
	timeout := cfg.Timeout.Duration
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	switch cfg.Provider {
	case "", "none":
		return nil, nil
	case "opencode", "claude":
		bin := cfg.CLI
		if bin == "" {
			bin = cfg.Provider
		}
		return newCLI(cfg.Provider, bin, cfg.Model, timeout), nil
	case "deepseek":
		return &openAICompat{
			baseURL:     orDefault(cfg.BaseURL, "https://api.deepseek.com/chat/completions"),
			apiKey:      envKey(cfg, "DEEPSEEK_API_KEY"),
			model:       cfg.Model,
			defModel:    "deepseek-chat",
			maxTokens:   cfg.MaxTokens,
			temperature: cfg.Temperature,
			timeout:     timeout,
		}, nil
	case "openai":
		return &openAICompat{
			baseURL:     orDefault(cfg.BaseURL, "https://api.openai.com/v1/chat/completions"),
			apiKey:      envKey(cfg, "OPENAI_API_KEY"),
			model:       cfg.Model,
			defModel:    "gpt-4o-mini",
			maxTokens:   cfg.MaxTokens,
			temperature: cfg.Temperature,
			timeout:     timeout,
		}, nil
	case "anthropic":
		return &anthropicLLM{
			baseURL:     orDefault(cfg.BaseURL, "https://api.anthropic.com/v1/messages"),
			apiKey:      envKey(cfg, "ANTHROPIC_API_KEY"),
			model:       cfg.Model,
			defModel:    "claude-3-5-haiku-latest",
			maxTokens:   cfg.MaxTokens,
			temperature: cfg.Temperature,
			timeout:     timeout,
		}, nil
	default:
		return nil, fmt.Errorf("provider de LLM desconhecido: %q", cfg.Provider)
	}
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func envKey(cfg llmConfig, fallback string) string {
	name := cfg.APIKeyEnv
	if name == "" {
		name = fallback
	}
	return os.Getenv(name)
}

// cliLLM shells out to a local agent CLI — opencode run / claude -p — the
// zero-extra-setup backends (they reuse whatever auth the CLI already has).
type cliLLM struct {
	bin     string
	prefix  []string
	timeout time.Duration
}

func newCLI(provider, bin, model string, timeout time.Duration) llmProvider {
	prefix := []string{}
	switch provider {
	case "opencode":
		prefix = append(prefix, "run")
		if model != "" {
			prefix = append(prefix, "-m", model)
		}
	case "claude":
		prefix = append(prefix, "-p")
		if model != "" {
			prefix = append(prefix, "--model", model)
		}
	}
	return &cliLLM{bin: bin, prefix: prefix, timeout: timeout}
}

func (l *cliLLM) Complete(ctx context.Context, system, user string) (string, error) {
	prompt := user
	if system != "" {
		prompt = system + "\n\n" + user
	}
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	args := make([]string, 0, len(l.prefix)+1)
	args = append(args, l.prefix...)
	args = append(args, prompt)
	cmd := exec.CommandContext(ctx, l.bin, args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %v (%s)", l.bin, err, strings.TrimSpace(errBuf.String()))
	}
	text := stripANSI(out.String())
	// The CLIs can interleave the answer with spinner/build chatter on stdout
	// on some versions — the final reply is the last non-empty block.
	return strings.TrimSpace(text), nil
}

var ansiRe = regexp.MustCompile(`\x1b\][^\x07]*\x07|\x1b\[[0-9;?]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// openAICompat hits any OpenAI-compatible /chat/completions endpoint —
// deepseek and openai are just two default base URLs for the same client.
type openAICompat struct {
	baseURL     string
	apiKey      string
	model       string
	defModel    string
	maxTokens   int
	temperature float64
	timeout     time.Duration
}

func (p *openAICompat) Complete(ctx context.Context, system, user string) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("%s: chave de API não configurada (env %s)", p.baseURL, "da sua llm.api_key_env")
	}
	model := p.model
	if model == "" {
		model = p.defModel
	}
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": p.temperature,
	}
	if p.maxTokens > 0 {
		payload["max_tokens"] = p.maxTokens
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: status %d: %s", p.baseURL, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("%s: resposta sem conteúdo", p.baseURL)
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

// anthropicLLM is the Anthropic Messages API, which has its own wire shape
// (x-api-key header, required max_tokens, system as a top-level field).
type anthropicLLM struct {
	baseURL     string
	apiKey      string
	model       string
	defModel    string
	maxTokens   int
	temperature float64
	timeout     time.Duration
}

func (p *anthropicLLM) Complete(ctx context.Context, system, user string) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("%s: chave de API não configurada (env %s)", p.baseURL, "da sua llm.api_key_env")
	}
	model := p.model
	if model == "" {
		model = p.defModel
	}
	maxTokens := p.maxTokens
	if maxTokens <= 0 {
		maxTokens = 1024 // required by the API
	}
	payload := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   []map[string]string{{"role": "user", "content": user}},
	}
	if p.temperature != 0 {
		payload["temperature"] = p.temperature
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: status %d: %s", p.baseURL, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	for _, block := range parsed.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return strings.TrimSpace(block.Text), nil
		}
	}
	return "", fmt.Errorf("%s: resposta sem conteúdo", p.baseURL)
}
