// Package textutil provides context-free text normalization helpers.
package textutil

import "strings"

// CollapseWhitespace trims value and replaces whitespace runs with one space.
func CollapseWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// NormalizeNonBlank collapses whitespace and removes blank entries.
// It returns an initialized empty slice for an empty input so domain collections
// remain JSON arrays instead of null values.
func NormalizeNonBlank(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if value = CollapseWhitespace(value); value != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

// TrimNonBlank trims entries and removes blank values while preserving their order.
func TrimNonBlank(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

// SplitNonBlank splits value and removes blank, whitespace-only parts.
func SplitNonBlank(value, separator string) []string {
	return TrimNonBlank(strings.Split(value, separator))
}

// TrimmedPointer returns nil for blank text or a pointer to its trimmed value.
func TrimmedPointer(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

// TrimmedPointerFrom normalizes an optional string and preserves nil or blank as nil.
func TrimmedPointerFrom(value *string) *string {
	if value == nil {
		return nil
	}
	return TrimmedPointer(*value)
}

// FirstNonBlank returns the first value containing non-whitespace characters.
func FirstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// ValueOr returns a trimmed optional value or fallback when it is nil or blank.
func ValueOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	if trimmed := strings.TrimSpace(*value); trimmed != "" {
		return trimmed
	}
	return fallback
}
