package assistant

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/acai-travel/tech-challenge/internal/chat/assistant/tools"
	"github.com/openai/openai-go/v2"
)

type Assistant struct {
	cli          openai.Client
	toolRegistry *tools.Registry
}

func New() *Assistant {
	// Initialize tool registry
	registry := tools.NewRegistry()
	
	// Register all tools
	weatherAPIKey := os.Getenv("WEATHER_API_KEY")
	if weatherAPIKey != "" {
		registry.Register(tools.NewWeatherTool(weatherAPIKey))
		slog.Info("Weather tool registered successfully")
	} else {
		slog.Warn("WEATHER_API_KEY not set, weather functionality will be limited")
	}
	
	// Register stock tool if API key is available
	stockAPIKey := os.Getenv("STOCK_API_KEY")
	if stockAPIKey != "" {
		registry.Register(tools.NewStockTool(stockAPIKey))
		slog.Info("Stock tool registered successfully")
	} else {
		slog.Warn("STOCK_API_KEY not set, stock functionality will be unavailable")
	}
	
	registry.Register(tools.NewDateTool())
	registry.Register(tools.NewHolidaysTool())

	return &Assistant{
		cli:          openai.NewClient(),
		toolRegistry: registry,
	}
}

func (a *Assistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "An empty conversation", nil
	}

	slog.InfoContext(ctx, "Generating title for conversation", "conversation_id", conv.ID)

	msgs := make([]openai.ChatCompletionMessageParamUnion, len(conv.Messages)+1)
	msgs[0] = openai.AssistantMessage(`Generate a concise, descriptive title that summarizes the user's question or topic. 
	The title should be 2-5 words maximum, no more than 80 characters, and should NOT answer the question. 
	Focus on extracting the main subject matter only. 
	
	Examples:
	- "What is the weather in Barcelona?" → "Weather in Barcelona"
	- "How do I bake chocolate chip cookies?" → "Chocolate Chip Cookie Recipe"
	- "Tell me about the history of ancient Rome" → "Ancient Roman History"
	- "What are the best restaurants in Paris?" → "Paris Restaurant Recommendations"`)
		
	for i, m := range conv.Messages {
		msgs[i+1] = openai.UserMessage(m.Content)
	}

	resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4Turbo,
		Messages: msgs,
	})

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return "", errors.New("empty response from OpenAI for title generation")
	}

	title := resp.Choices[0].Message.Content
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.Trim(title, " \t\r\n-\"'")

	if len(title) > 80 {
		title = title[:80]
	}

	return title, nil
}

func (a *Assistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "", errors.New("conversation has no messages")
	}

	slog.InfoContext(ctx, "Generating reply for conversation", "conversation_id", conv.ID)

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful, concise AI assistant. Provide accurate, safe, and clear responses."),
	}

	for _, m := range conv.Messages {
		switch m.Role {
		case model.RoleUser:
			msgs = append(msgs, openai.UserMessage(m.Content))
		case model.RoleAssistant:
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		}
	}
	
	const maxToolIterations = 15
	for i := 0; i < maxToolIterations; i++ {
		resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4_1,
			Messages: msgs,
			Tools:    a.toolRegistry.GetToolDefinitions(),
		})

		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", errors.New("no choices returned by OpenAI")
		}
		
		message := resp.Choices[0].Message
		if len(message.ToolCalls) == 0 {
			return message.Content, nil
		}
		

		// Add the assistant's message with tool calls
		msgs = append(msgs, message.ToParam())
		
		// Process each tool call - ACCESS FIELDS DIRECTLY FROM THE STRUCT
		for _, call := range message.ToolCalls {
			// Since openai.ChatCompletionMessageToolCallUnion is a struct, access fields directly
			toolName := call.Function.Name
			arguments := call.Function.Arguments
			callID := call.ID
			
			result, err := a.toolRegistry.ExecuteTool(ctx, toolName, arguments)
			if err != nil {
				result = "Error: " + err.Error()
			}
			// Append the tool result message
			msgs = append(msgs, openai.ToolMessage(result, callID))
		}
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}