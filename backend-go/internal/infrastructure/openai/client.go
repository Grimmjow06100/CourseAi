package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/correlation"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/errtrace"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

func (g *CourseAIGenerator) callStructuredJSON(ctx context.Context, promptName string, payload any, schemaName string, schema map[string]any) (json.RawMessage, error) {
	if g == nil || g.client == nil {
		return nil, errtrace.Capture(ErrMissingClient)
	}
	if g.prompts == nil {
		return nil, errtrace.Capture(ErrMissingPromptStore)
	}

	prompt, ok := g.prompts.Get(promptName)
	if !ok || strings.TrimSpace(prompt) == "" {
		return nil, errtrace.Capture(fmt.Errorf("%w: %s", ErrPromptNotFound, promptName))
	}

	inputJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, errtrace.Wrap(err, "marshal prompt payload")
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
		PromptCacheKey:  openaisdk.String("course-ai:" + g.conf.Model + ":" + promptName + ":v1"),
	}

	response, err := g.client.Responses.New(ctx, params)
	if err != nil {
		return nil, errtrace.Capture(classifyProviderRequestError(err))
	}
	logArgs := []any{
		"event", "openai_response_completed",
		"prompt", promptName,
		"schema", schemaName,
		"model", g.conf.Model,
		"response_id", response.ID,
		"input_tokens", response.Usage.InputTokens,
		"cached_input_tokens", response.Usage.InputTokensDetails.CachedTokens,
		"cache_write_tokens", response.Usage.InputTokensDetails.CacheWriteTokens,
		"output_tokens", response.Usage.OutputTokens,
		"reasoning_tokens", response.Usage.OutputTokensDetails.ReasoningTokens,
		"total_tokens", response.Usage.TotalTokens,
	}
	if job, ok := correlation.JobFromContext(ctx); ok {
		logArgs = append(logArgs,
			"job_id", job.JobID,
			"request_id", job.RequestID,
			"job_kind", job.Kind,
			"attempt", job.Attempt,
		)
	}
	slog.InfoContext(ctx, "OpenAI response completed", logArgs...)

	output := strings.TrimSpace(response.OutputText())
	raw, err := extractJSONPayload(output)
	if err != nil {
		return nil, errtrace.Capture(fmt.Errorf("%w: %v", ErrInvalidModelOutput, err))
	}

	return raw, nil
}
