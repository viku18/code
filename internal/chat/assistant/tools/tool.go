package tools

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/openai/openai-go/v2"
)

// Tool defines the interface for all assistant tools
type Tool interface {
	Name() string
	Definition() openai.ChatCompletionToolUnionParam
	Execute(ctx context.Context, arguments json.RawMessage) (string, error)
}

// Errors
var (
	ErrToolNotFound = errors.New("tool not found")
)
