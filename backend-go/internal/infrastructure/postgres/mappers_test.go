package postgres

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

func TestQuizQuestionsJSONUsesFrontendShape(t *testing.T) {
	t.Parallel()

	answer := "Dockerfile"
	questions := []domain.QuizQuestion{
		{
			Order:    1,
			Type:     domain.QuizQuestionTypeSingleChoice,
			Question: "Quel fichier decrit une image Docker ?",
			Options: []domain.QuizOption{
				{Order: 1, Text: "docker-compose.yml"},
				{Order: 2, Text: "Dockerfile"},
			},
			Answer:     domain.QuizAnswer{Answer: &answer},
			Correction: "Un Dockerfile decrit les instructions de construction d'une image.",
		},
	}

	raw, err := quizQuestionsJSON(questions)
	if err != nil {
		t.Fatalf("marshal quiz questions: %v", err)
	}
	if !strings.Contains(raw, `"order"`) || strings.Contains(raw, `"Order"`) {
		t.Fatalf("expected frontend JSON shape, got %s", raw)
	}

	decodedQuestions, err := quizQuestionsFromJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unmarshal quiz questions: %v", err)
	}
	if len(decodedQuestions) != 1 {
		t.Fatalf("expected one decoded question, got %d", len(decodedQuestions))
	}
	if decodedQuestions[0].Type != domain.QuizQuestionTypeSingleChoice {
		t.Fatalf("unexpected question type: %s", decodedQuestions[0].Type)
	}
	if decodedQuestions[0].Answer.Answer == nil || *decodedQuestions[0].Answer.Answer != answer {
		t.Fatalf("unexpected answer: %#v", decodedQuestions[0].Answer.Answer)
	}
	if len(decodedQuestions[0].Answer.Answers) != 0 {
		t.Fatalf("expected empty multiple answers, got %#v", decodedQuestions[0].Answer.Answers)
	}
}

func TestExercisePayloadJSONRoundTrip(t *testing.T) {
	t.Parallel()

	payload := domain.ExercisePayload{
		"tasks":          []string{"Lister les conteneurs"},
		"resources":      []string{"docker ps"},
		"starterCode":    nil,
		"expectedOutput": "Une liste de conteneurs",
		"hints":          []string{"Utilise la CLI Docker."},
	}

	raw, err := exercisePayloadJSON(payload)
	if err != nil {
		t.Fatalf("marshal exercise payload: %v", err)
	}

	decodedPayload, err := exercisePayloadFromJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unmarshal exercise payload: %v", err)
	}
	if decodedPayload["expectedOutput"] != "Une liste de conteneurs" {
		t.Fatalf("unexpected payload: %#v", decodedPayload)
	}
}

func TestRawJSONValueRoundTrip(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"source":"openai","step":"lesson_content"}`)

	value := rawJSONValue(raw)
	rawValue, ok := value.(string)
	if !ok {
		t.Fatalf("expected raw JSON value to be a string, got %T", value)
	}
	if rawValue != string(raw) {
		t.Fatalf("unexpected raw JSON value: %s", rawValue)
	}

	decodedRaw := rawJSONFromBytes([]byte(rawValue))
	if string(decodedRaw) != string(raw) {
		t.Fatalf("unexpected decoded raw JSON: %s", string(decodedRaw))
	}
	if rawJSONValue(nil) != nil {
		t.Fatal("expected nil raw JSON to map to nil database value")
	}
	if rawJSONFromBytes(nil) != nil {
		t.Fatal("expected nil database value to map to nil raw JSON")
	}
}
