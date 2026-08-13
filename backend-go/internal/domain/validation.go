package domain

import (
	"fmt"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/textutil"
)

func normalizeText(value string) string {
	return textutil.CollapseWhitespace(value)
}

func normalizeMarkdown(value string) string {
	return strings.TrimSpace(value)
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	normalized := normalizeText(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func requireNotBlank(fieldName string, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: %s", ErrBlankField, fieldName)
	}
	return nil
}

func validatePositiveInt(fieldName string, value int) error {
	if value <= 0 {
		return fmt.Errorf("%w: %s", ErrInvalidOrder, fieldName)
	}
	return nil
}

func normalizeStringSlice(values []string) []string {
	return textutil.NormalizeNonBlank(values)
}

func validateProgressPercent(progress int) error {
	if progress < 0 || progress > 100 {
		return ErrInvalidProgress
	}
	return nil
}
