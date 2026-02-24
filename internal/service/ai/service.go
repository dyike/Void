package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/crosszan/modu/pkg/agent"
	"github.com/crosszan/modu/pkg/coding_agent/tools"
	"github.com/crosszan/modu/pkg/llm"
	_ "github.com/crosszan/modu/pkg/llm/providers/ollama"
	_ "github.com/crosszan/modu/pkg/llm/providers/openai"
	_ "github.com/crosszan/modu/pkg/llm/providers/openai_completions"

	"github.com/dyike/Void/internal/service/settings"
)

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

// buildModel sets environment variables for the provider and returns the llm.Model.
func buildModel(ai settings.AISettings) (*llm.Model, error) {
	modelID := ai.DefaultModel

	switch ai.Provider {
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
		os.Setenv("OPENAI_BASE_URL", baseURL)
		os.Setenv("OPENAI_API_KEY", ai.OpenAICompatibleApiKey)
		return &llm.Model{
			ID:       modelID,
			Provider: llm.Provider(llm.KnownProviderOpenAI),
			Api:      llm.Api(llm.KnownApiOpenAICompletions),
			BaseURL:  baseURL,
		}, nil

	default: // "openai"
		if modelID == "" {
			modelID = "gpt-4o-mini"
		}
		apiKey := ai.OpenAIApiKey
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key is not configured")
		}
		os.Setenv("OPENAI_API_KEY", apiKey)
		return &llm.Model{
			ID:       modelID,
			Provider: llm.Provider(llm.KnownProviderOpenAI),
			Api:      llm.Api(llm.KnownApiOpenAIResponses),
			BaseURL:  "https://api.openai.com/v1",
		}, nil
	}
}

// Ask sends a prompt using the modu Agent and returns the full response.
func (s *Service) Ask(prompt string) (*AskResult, error) {
	cfg, err := s.settings.GetSettings()
	if err != nil {
		log.Printf("[AI] failed to load settings: %v", err)
		return &AskResult{Error: "Failed to load settings: " + err.Error()}, nil
	}
	ai := cfg.AI
	if !ai.Enabled {
		log.Printf("[AI] AI is disabled in settings")
		return &AskResult{Error: "AI is disabled. Please enable it in Settings."}, nil
	}

	model, err := buildModel(ai)
	if err != nil {
		log.Printf("[AI] buildModel error: %v", err)
		return &AskResult{Error: err.Error()}, nil
	}

	log.Printf("[AI] Starting agent with model=%s provider=%s api=%s", model.ID, model.Provider, model.Api)

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
				"- Always respond in the user's language (Chinese if asked in Chinese)\n" +
				"- If you cannot complete a task, say so clearly with the reason\n\n" +
				"# Environment\n" +
				"- Operating system: macOS\n" +
				"- Working directory: " + cwd,
			Model: model,
			Tools: tools.AllTools(cwd),
		},
		GetAPIKey: func(p string) (string, error) {
			return "", nil
		},
	}

	// turnCount tracks EventTypeTurnStart events; cancel context if too many.
	// Use a 90-second hard timeout AND a 10-turn soft limit.
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	turnCount := 0
	a := agent.NewAgent(agentOpts)

	// Subscribe to streaming events and push to Wails frontend
	a.Subscribe(func(e agent.AgentEvent) {
		app := application.Get()
		if app == nil {
			return
		}
		switch e.Type {
		case agent.EventTypeTurnStart:
			turnCount++
			log.Printf("[AI] turn %d", turnCount)
			if turnCount > 10 {
				log.Printf("[AI] turn limit reached, aborting")
				cancel()
			}
		case agent.EventTypeMessageUpdate:
			if e.AssistantMessageEvent != nil && e.AssistantMessageEvent.Delta != "" {
				log.Printf("[AI] delta: %q", e.AssistantMessageEvent.Delta)
				app.Event.Emit("AgentUpdate", e.AssistantMessageEvent.Delta)
			}
		case agent.EventTypeToolExecutionStart:
			argsJSON, _ := json.Marshal(e.Args)
			log.Printf("[AI] tool start: %s args=%s", e.ToolName, string(argsJSON))
			app.Event.Emit("ToolCallStart", e.ToolName, string(argsJSON))
			app.Event.Emit("AgentAction", fmt.Sprintf("Running: %s", e.ToolName))
		case agent.EventTypeToolExecutionEnd:
			resultJSON, _ := json.Marshal(e.Result)
			log.Printf("[AI] tool end: %s isError=%v", e.ToolName, e.IsError)
			app.Event.Emit("ToolCallEnd", e.ToolName, string(resultJSON), e.IsError)
			app.Event.Emit("AgentAction", "")
		case agent.EventTypeAgentEnd:
			log.Printf("[AI] agent finished")
		}
	})

	log.Printf("[AI] Prompting agent: %q", prompt)
	if err := a.Prompt(ctx, prompt); err != nil {
		log.Printf("[AI] Prompt error: %v", err)
		return &AskResult{Error: err.Error()}, nil
	}
	a.WaitForIdle()
	log.Printf("[AI] Agent idle, extracting result")

	state := a.GetState()
	if state.Error != "" {
		log.Printf("[AI] agent state error: %s", state.Error)
		return &AskResult{Error: state.Error}, nil
	}

	// Extract the final text content from the last assistant message
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
			break // Only use the last assistant message
		}
	}

	result := finalContent.String()
	log.Printf("[AI] Final result length: %d chars", len(result))

	if result == "" {
		result = "(No response received. Check the Go console for error details.)"
	}

	return &AskResult{Content: result}, nil
}

// AskWithHistory uses only the last user message (modu manages history internally).
func (s *Service) AskWithHistory(messages []map[string]interface{}) (*AskResult, error) {
	if len(messages) == 0 {
		return &AskResult{Error: "no messages"}, nil
	}
	last, _ := messages[len(messages)-1]["content"].(string)
	if last == "" {
		return &AskResult{Error: "empty message"}, nil
	}
	return s.Ask(last)
}
