// Package jsonutil contains generic helpers for immutable JSON handling.
package jsonutil

import (
	"encoding/json"
	"reflect"
)

// Clone returns a defensive copy of raw and treats empty JSON as absent.
func Clone(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	cloned := make(json.RawMessage, len(raw))
	copy(cloned, raw)
	return cloned
}

// EqualObjects compares JSON values semantically instead of comparing their formatting.
func EqualObjects(left, right json.RawMessage) bool {
	var leftValue any
	var rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}
