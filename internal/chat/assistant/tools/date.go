package tools

import (
	"context"
	"time"
	"encoding/json"

	"github.com/openai/openai-go/v2"
)

type DateTool struct{}

func NewDateTool() *DateTool {
	return &DateTool{}
}

func (d *DateTool) Name() string {
	return "get_today_date"
}

func (d *DateTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        d.Name(),
		Description: openai.String("Get today's date and time in RFC3339 format"),
	})
}

func (d *DateTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}