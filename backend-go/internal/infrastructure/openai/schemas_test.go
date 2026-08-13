package openai

import (
	"encoding/json"
	"testing"
)

func TestStructuredOutputSchemasAreStrictJSONObjects(t *testing.T) {
	t.Parallel()

	schemas := map[string]map[string]any{
		"analysis":       analysisSchema(),
		"architecture":   architectureSchema(),
		"lessons":        lessonsSchema(),
		"lesson content": lessonContentSchema(),
	}
	for name, schema := range schemas {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if schema["type"] != "object" || schema["additionalProperties"] != false {
				t.Fatalf("schema is not a strict object: %#v", schema)
			}
			if _, ok := schema["required"].([]string); !ok {
				t.Fatalf("schema required field has unexpected type: %T", schema["required"])
			}
			if _, err := json.Marshal(schema); err != nil {
				t.Fatalf("schema is not JSON serializable: %v", err)
			}
		})
	}
}

func TestSchemaHelpers(t *testing.T) {
	t.Parallel()

	if arraySchema(stringSchema())["type"] != "array" || boolSchema()["type"] != "boolean" || integerSchema()["type"] != "integer" {
		t.Fatal("primitive schema helper returned the wrong type")
	}
	if nullableStringSchema()["type"] == nil || difficultySchema()["enum"] == nil || enumSchema([]string{"a"})["type"] != "string" {
		t.Fatal("nullable or enum schema helper is malformed")
	}
}
