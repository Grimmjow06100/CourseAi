package domain

import (
	"fmt"
	"strings"
)

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
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
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		cleaned := normalizeText(value)
		if cleaned != "" {
			normalized = append(normalized, cleaned)
		}
	}
	return normalized
}

func validateProgressPercent(progress int) error {
	if progress < 0 || progress > 100 {
		return ErrInvalidProgress
	}
	return nil
}
