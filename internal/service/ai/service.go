package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dyike/Void/internal/service/settings"
)

// ChatMessage is a single turn in the conversation.
type ChatMessage struct {
	Role    string `json:"role"` // "user" | "assistant" | "system"
	Content string `json:"content"`
}

// AskResult holds the completed AI response.
type AskResult struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// Service handles AI inference.
type Service struct {
	settings *settings.Service
}

// NewService creates a new AI service.
func NewService(settingsSvc *settings.Service) *Service {
	return &Service{settings: settingsSvc}
}

// Ask sends a prompt and returns the complete response.
func (s *Service) Ask(prompt string) (*AskResult, error) {
	cfg, err := s.settings.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}
	ai := cfg.AI
	if !ai.Enabled {
		return nil, fmt.Errorf("AI is disabled in settings")
	}

	switch ai.Provider {
	case "ollama":
		return s.askOllama(prompt, ai)
	case "openai-compatible":
		return s.askOpenAICompat(prompt, ai)
	default: // "openai"
		return s.askOpenAI(prompt, ai)
	}
}

// AskWithHistory sends a conversation and returns the complete response.
func (s *Service) AskWithHistory(messages []ChatMessage) (*AskResult, error) {
	cfg, err := s.settings.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}
	ai := cfg.AI
	if !ai.Enabled {
		return nil, fmt.Errorf("AI is disabled in settings")
	}

	switch ai.Provider {
	case "ollama":
		return s.ollamaChat(messages, ai)
	case "openai-compatible":
		return s.openAICompatChat(messages, ai)
	default:
		return s.openAIChat(messages, ai)
	}
}

// ─── OpenAI ───────────────────────────────────────────────────────────────

func (s *Service) askOpenAI(prompt string, ai settings.AISettings) (*AskResult, error) {
	return s.openAIChat([]ChatMessage{{Role: "user", Content: prompt}}, ai)
}

func (s *Service) openAIChat(messages []ChatMessage, ai settings.AISettings) (*AskResult, error) {
	model := ai.DefaultModel
	if model == "" {
		model = "gpt-4o-mini"
	}

	reqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}

	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ai.OpenAIApiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return parseOpenAIResponse(resp)
}

// ─── OpenAI-Compatible ────────────────────────────────────────────────────

func (s *Service) askOpenAICompat(prompt string, ai settings.AISettings) (*AskResult, error) {
	return s.openAICompatChat([]ChatMessage{{Role: "user", Content: prompt}}, ai)
}

func (s *Service) openAICompatChat(messages []ChatMessage, ai settings.AISettings) (*AskResult, error) {
	baseURL := ai.OpenAICompatibleBaseUrl
	if baseURL == "" {
		return nil, fmt.Errorf("openai-compatible base URL not configured")
	}
	model := ai.OpenAICompatibleModel
	if model == "" {
		model = ai.DefaultModel
	}

	reqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}

	data, _ := json.Marshal(reqBody)
	url := strings.TrimRight(baseURL, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if ai.OpenAICompatibleApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+ai.OpenAICompatibleApiKey)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return parseOpenAIResponse(resp)
}

// ─── Ollama ───────────────────────────────────────────────────────────────

func (s *Service) askOllama(prompt string, ai settings.AISettings) (*AskResult, error) {
	return s.ollamaChat([]ChatMessage{{Role: "user", Content: prompt}}, ai)
}

func (s *Service) ollamaChat(messages []ChatMessage, ai settings.AISettings) (*AskResult, error) {
	baseURL := ai.OllamaBaseUrl
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := ai.DefaultModel
	if model == "" {
		model = "llama3"
	}

	reqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   true,
	}

	data, _ := json.Marshal(reqBody)
	url := strings.TrimRight(baseURL, "/") + "/api/chat"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error %d: %s", resp.StatusCode, string(body))
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		sb.WriteString(chunk.Message.Content)
		if chunk.Done {
			break
		}
	}

	return &AskResult{Content: sb.String()}, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func parseOpenAIResponse(resp *http.Response) (*AskResult, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if result.Error.Message != "" {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &AskResult{Content: result.Choices[0].Message.Content}, nil
}
