package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/crosszan/modu/pkg/agent"
	"github.com/crosszan/modu/pkg/coding_agent/tools"
	"github.com/crosszan/modu/pkg/llm"
	_ "github.com/crosszan/modu/pkg/llm/providers/deepseek"
	_ "github.com/crosszan/modu/pkg/llm/providers/ollama"
	_ "github.com/crosszan/modu/pkg/llm/providers/openai"
	_ "github.com/crosszan/modu/pkg/llm/providers/openai_chat_completions"
	"github.com/dyike/Void/internal/model"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// AIService handles AI inference.
type AIService struct {
	settings *SettingsService
}

// NewAIService creates a new AI service.
func NewAIService(settingsSvc *SettingsService) *AIService {
	return &AIService{settings: settingsSvc}
}

func buildModel(ai model.AISettings) (*llm.Model, error) {
	modelID := ai.DefaultModel

	switch ai.Provider {
	case "deepseek":
		if modelID == "" {
			modelID = "deepseek-chat"
		}
		if ai.DeepSeekApiKey == "" {
			return nil, fmt.Errorf("DeepSeek API key is not configured")
		}
		os.Setenv("DEEPSEEK_API_KEY", ai.DeepSeekApiKey)
		return &llm.Model{
			ID:       modelID,
			Provider: llm.Provider(llm.KnownProviderDeepSeek),
			Api:      llm.Api(llm.KnownApiDeepSeekChat),
			BaseURL:  "https://api.deepseek.com/v1",
		}, nil

	case "ollama":
		if modelID == "" {
			modelID = "llama3"
		}
		baseURL := ai.OllamaBaseUrl
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		os.Setenv("OLLAMA_BASE_URL", baseURL)
		return &llm.Model{
			ID:       modelID,
			Provider: llm.Provider(llm.KnownProviderOllama),
			Api:      llm.Api(llm.KnownApiOllama),
			BaseURL:  baseURL,
		}, nil

	case "openai-compatible":
		if ai.OpenAICompatibleModel != "" {
			modelID = ai.OpenAICompatibleModel
		}
		if modelID == "" {
			modelID = "gpt-4o-mini"
		}
		baseURL := strings.TrimRight(ai.OpenAICompatibleBaseUrl, "/")
		if baseURL == "" {
			return nil, fmt.Errorf("openai-compatible base URL is not configured")
		}
		apiKey := ai.OpenAICompatibleApiKey
		if apiKey == "" {
			apiKey = "dummy-key" // provide a dummy key since modu requires one
		}
		os.Setenv("OPENAI_API_KEY", apiKey)
		return &llm.Model{
			ID:       modelID,
			Provider: llm.Provider(llm.KnownProviderOpenAI),
			Api:      llm.Api(llm.KnownApiOpenAIChatCompletions),
			BaseURL:  baseURL,
		}, nil

	default: // "openai"
		if modelID == "" {
			modelID = "gpt-4o-mini"
		}
		if ai.OpenAIApiKey == "" {
			return nil, fmt.Errorf("OpenAI API key is not configured")
		}
		os.Setenv("OPENAI_API_KEY", ai.OpenAIApiKey)
		return &llm.Model{
			ID:       modelID,
			Provider: llm.Provider(llm.KnownProviderOpenAI),
			Api:      llm.Api(llm.KnownApiOpenAIResponses),
			BaseURL:  "https://api.openai.com/v1",
		}, nil
	}
}

// Ask sends a prompt using the modu Agent and returns the full response.
func (s *AIService) Ask(prompt string) (*model.AskResult, error) {
	cfg, err := s.settings.GetSettings()
	if err != nil {
		return &model.AskResult{Error: "Failed to load settings: " + err.Error()}, nil
	}
	ai := cfg.AI
	if !ai.Enabled {
		return &model.AskResult{Error: "AI is disabled. Please enable it in Settings."}, nil
	}

	m, err := buildModel(ai)
	if err != nil {
		log.Printf("[AI] buildModel error: %v", err)
		return &model.AskResult{Error: err.Error()}, nil
	}

	log.Printf("[AI] Starting agent with model=%s provider=%s api=%s", m.ID, m.Provider, m.Api)

	cwd, _ := os.UserHomeDir()
	agentOpts := agent.AgentOptions{
		InitialState: &agent.AgentState{
			SystemPrompt: "You are Void, an intelligent macOS desktop assistant.\n" +
				"You have access to tools for reading files, writing files, editing files, " +
				"running bash commands, searching code (grep), finding files, and listing directories.\n\n" +
				"# Guidelines\n" +
				"- Use the most appropriate tool for each task (prefer read/grep/find over bash when possible)\n" +
				"- NEVER call the same tool with the same arguments twice — use the result from the first call\n" +
				"- Once you have the information needed, respond immediately without additional tool calls\n" +
				"- Read files before modifying them to understand existing content\n" +
				"- Use edit for precise modifications instead of rewriting entire files\n" +
				"- If a command or tool fails, try ONE alternative approach, then explain what happened\n" +
				"- NEVER retry the same failed command — explain the error to the user instead\n" +
				"- Keep responses concise and in Markdown format\n" +
				"- Always respond in the user's language; use Simplified Chinese (简体中文) when the user writes in Chinese\n" +
				"- If you cannot complete a task, say so clearly with the reason\n\n" +
				"# Environment\n" +
				"- Operating system: macOS\n" +
				"- Working directory: " + cwd,
			Model: m,
			Tools: tools.AllTools(cwd),
		},
		GetAPIKey: func(p string) (string, error) { return "", nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	turnCount := 0
	var seq int64
	a := agent.NewAgent(agentOpts)
	nextSeq := func() int64 { return atomic.AddInt64(&seq, 1) }

	a.Subscribe(func(e agent.AgentEvent) {
		app := application.Get()
		if app == nil {
			return
		}
		switch e.Type {
		case agent.EventTypeTurnStart:
			turnCount++
			if turnCount > 10 {
				cancel()
			}
			app.Event.Emit("AIEvent", model.AIEvent{Type: "turn", Turn: turnCount, Seq: nextSeq(), Timestamp: time.Now().UnixMilli()})
		case agent.EventTypeMessageUpdate:
			if e.AssistantMessageEvent != nil && e.AssistantMessageEvent.Delta != "" {
				app.Event.Emit("AIEvent", model.AIEvent{Type: "delta", Content: e.AssistantMessageEvent.Delta, Seq: nextSeq(), Timestamp: time.Now().UnixMilli()})
			}
		case agent.EventTypeToolExecutionStart:
			argsJSON, _ := json.Marshal(e.Args)
			app.Event.Emit("AIEvent", model.AIEvent{Type: "tool_start", Name: e.ToolName, Args: string(argsJSON), Seq: nextSeq(), Timestamp: time.Now().UnixMilli()})
		case agent.EventTypeToolExecutionEnd:
			resultJSON, _ := json.Marshal(e.Result)
			app.Event.Emit("AIEvent", model.AIEvent{Type: "tool_end", Name: e.ToolName, Result: string(resultJSON), IsError: e.IsError, Seq: nextSeq(), Timestamp: time.Now().UnixMilli()})
		case agent.EventTypeAgentEnd:
			app.Event.Emit("AIEvent", model.AIEvent{Type: "done", Seq: nextSeq(), Timestamp: time.Now().UnixMilli()})
		}
	})

	if err := a.Prompt(ctx, prompt); err != nil {
		return &model.AskResult{Error: err.Error()}, nil
	}
	a.WaitForIdle()

	state := a.GetState()
	if state.Error != "" {
		return &model.AskResult{Error: state.Error}, nil
	}

	var finalContent strings.Builder
	for i := len(state.Messages) - 1; i >= 0; i-- {
		if msg, ok := state.Messages[i].(llm.AssistantMessage); ok {
			for _, block := range msg.Content {
				switch t := block.(type) {
				case llm.TextContent:
					finalContent.WriteString(t.Text)
				case *llm.TextContent:
					if t != nil {
						finalContent.WriteString(t.Text)
					}
				}
			}
			break
		}
	}

	result := finalContent.String()
	if result == "" {
		result = "(No response received. Check the Go console for error details.)"
	}
	return &model.AskResult{Content: result}, nil
}

// AskWithHistory uses only the last user message.
func (s *AIService) AskWithHistory(messages []map[string]interface{}) (*model.AskResult, error) {
	if len(messages) == 0 {
		return &model.AskResult{Error: "no messages"}, nil
	}
	last, _ := messages[len(messages)-1]["content"].(string)
	if last == "" {
		return &model.AskResult{Error: "empty message"}, nil
	}
	return s.Ask(last)
}
