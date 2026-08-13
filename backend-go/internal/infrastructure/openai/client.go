package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

func (g *CourseAIGenerator) callStructuredJSON(ctx context.Context, promptName string, payload any, schemaName string, schema map[string]any) (json.RawMessage, error) {
	if g == nil || g.client == nil {
		return nil, ErrMissingClient
	}
	if g.prompts == nil {
		return nil, ErrMissingPromptStore
	}

	prompt, ok := g.prompts.Get(promptName)
	if !ok || strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("%w: %s", ErrPromptNotFound, promptName)
	}

	inputJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal prompt payload: %w", err)
	}

	format := responses.ResponseFormatTextConfigParamOfJSONSchema(schemaName, schema)
	if format.OfJSONSchema != nil {
		format.OfJSONSchema.Strict = openaisdk.Bool(true)
	}

	params := responses.ResponseNewParams{
		Model:        g.conf.Model,
		Instructions: openaisdk.String(prompt),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openaisdk.String(string(inputJSON)),
		},
		Store: openaisdk.Bool(false),
		Text: responses.ResponseTextConfigParam{
			Format: format,
		},
		MaxOutputTokens: openaisdk.Int(g.conf.MaxOutputTokens),
	}

	response, err := g.client.Responses.New(ctx, params)
	if err != nil {
		return nil, classifyProviderRequestError(err)
	}

	output := strings.TrimSpace(response.OutputText())
	raw, err := extractJSONPayload(output)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidModelOutput, err)
	}

	return raw, nil
}
