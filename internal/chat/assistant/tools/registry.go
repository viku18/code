package tools

import (
	"context"
	"encoding/json"
	//"errors"
	"log/slog"

	"github.com/openai/openai-go/v2"
)

// Registry manages all available tools
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// GetToolDefinitions returns all tool definitions for OpenAI
func (r *Registry) GetToolDefinitions() []openai.ChatCompletionToolUnionParam {
	definitions := make([]openai.ChatCompletionToolUnionParam, 0, len(r.tools))
	for _, tool := range r.tools {
		definitions = append(definitions, tool.Definition())
	}
	return definitions
}

// ExecuteTool executes a tool call by name and arguments
func (r *Registry) ExecuteTool(ctx context.Context, toolName, arguments string) (string, error) {
	tool, exists := r.tools[toolName]
	if !exists {
		return "", ErrToolNotFound
	}

	slog.InfoContext(ctx, "Executing tool", "name", toolName, "args", arguments)
	return tool.Execute(ctx, json.RawMessage(arguments))
}