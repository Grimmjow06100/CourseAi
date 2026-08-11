package openai

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractJSONPayloadAcceptsRawJSON(t *testing.T) {
	raw, err := extractJSONPayload(`{"title":"Formation Linux"}`)
	if err != nil {
		t.Fatalf("expected raw JSON to be accepted: %v", err)
	}

	assertDecodedTitle(t, raw, "Formation Linux")
}

func TestExtractJSONPayloadAcceptsFencedJSON(t *testing.T) {
	output := "```json\n{\"title\":\"Formation Linux\"}\n```"
	raw, err := extractJSONPayload(output)
	if err != nil {
		t.Fatalf("expected fenced JSON to be extracted: %v", err)
	}

	assertDecodedTitle(t, raw, "Formation Linux")
}

func TestExtractJSONPayloadAcceptsPrefixedJSON(t *testing.T) {
	output := `Voici le JSON demandé:
{"title":"Formation {Linux}","items":["shell","filesystem"]}
Fin.`
	raw, err := extractJSONPayload(output)
	if err != nil {
		t.Fatalf("expected embedded JSON to be extracted: %v", err)
	}

	assertDecodedTitle(t, raw, "Formation {Linux}")
}

func TestExtractJSONPayloadRejectsInvalidOutput(t *testing.T) {
	_, err := extractJSONPayload("```json\n{\"title\":\n```")
	if err == nil {
		t.Fatal("expected invalid JSON output to fail")
	}
	if !strings.Contains(err.Error(), "output is not valid json") {
		t.Fatalf("expected invalid json error, got: %v", err)
	}
}

func assertDecodedTitle(t *testing.T, raw json.RawMessage, expectedTitle string) {
	t.Helper()

	if !json.Valid(raw) {
		t.Fatalf("expected valid JSON, got: %s", raw)
	}

	var decoded struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("expected decodable JSON: %v", err)
	}
	if decoded.Title != expectedTitle {
		t.Fatalf("expected title %q, got %q", expectedTitle, decoded.Title)
	}
}
