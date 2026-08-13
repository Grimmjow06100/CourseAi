package openai

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/textutil"
)

func extractJSONPayload(output string) (json.RawMessage, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, errors.New("empty model output")
	}

	if json.Valid([]byte(output)) {
		return json.RawMessage(output), nil
	}

	if fenced, ok := extractFencedJSON(output); ok && json.Valid([]byte(fenced)) {
		return json.RawMessage(fenced), nil
	}

	if balanced, ok := extractBalancedJSON(output); ok && json.Valid([]byte(balanced)) {
		return json.RawMessage(balanced), nil
	}

	return nil, fmt.Errorf("output is not valid json: %s", outputPreview(output))
}

func extractFencedJSON(output string) (string, bool) {
	start := strings.Index(output, "```")
	if start == -1 {
		return "", false
	}

	content := output[start+3:]
	lineBreak := strings.IndexAny(content, "\r\n")
	if lineBreak == -1 {
		return "", false
	}
	content = content[lineBreak+1:]

	end := strings.LastIndex(content, "```")
	if end == -1 {
		return "", false
	}

	return strings.TrimSpace(content[:end]), true
}

func extractBalancedJSON(output string) (string, bool) {
	start := -1
	for i := 0; i < len(output); i++ {
		if output[i] == '{' || output[i] == '[' {
			start = i
			break
		}
	}
	if start == -1 {
		return "", false
	}

	stack := make([]byte, 0, 8)
	inString := false
	escaped := false

	for i := start; i < len(output); i++ {
		char := output[i]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}

		switch char {
		case '"':
			inString = true
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}', ']':
			if len(stack) == 0 || stack[len(stack)-1] != char {
				return "", false
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return strings.TrimSpace(output[start : i+1]), true
			}
		}
	}

	return "", false
}

func outputPreview(value string) string {
	value = textutil.CollapseWhitespace(value)
	const maxPreviewLength = 240
	if len(value) <= maxPreviewLength {
		return value
	}
	return value[:maxPreviewLength] + "..."
}
