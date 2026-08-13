package jsonutil

import (
	"encoding/json"
	"testing"
)

func TestCloneReturnsIndependentJSON(t *testing.T) {
	original := json.RawMessage(`{"course":"linux"}`)
	cloned := Clone(original)
	cloned[2] = 'X'

	if string(original) != `{"course":"linux"}` {
		t.Fatal("changing the clone must not mutate the original JSON")
	}
	if Clone(nil) != nil {
		t.Fatal("cloning nil must return nil")
	}
	if Clone(json.RawMessage{}) != nil {
		t.Fatal("cloning empty JSON must return nil")
	}
}

func TestEqualObjectsIgnoresFormattingAndPropertyOrder(t *testing.T) {
	left := json.RawMessage(`{"course":"linux","level":1}`)
	right := json.RawMessage(`{ "level": 1, "course": "linux" }`)
	if !EqualObjects(left, right) {
		t.Fatal("semantically identical JSON objects must be equal")
	}
	if EqualObjects(left, json.RawMessage(`{"course":"go"}`)) {
		t.Fatal("different JSON objects must not be equal")
	}
	if EqualObjects(left, json.RawMessage(`invalid`)) {
		t.Fatal("invalid JSON must not compare as equal")
	}
}
