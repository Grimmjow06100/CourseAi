package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func TestCourseAIGeneratorCallsStructuredResponsesAPI(t *testing.T) {
	t.Parallel()

	outputs := map[string]string{
		"analysis prompt":     `{"isOutOfScope":false,"errorMessage":null,"warningMessage":null,"suggestedTitle":"Linux","shortSynopsis":"Course","detectedCurrentLevel":"beginner","detectedTargetLevel":"advanced","detectedGoal":"Administer Linux","detectedLanguage":"en","clarificationQuestions":[]}`,
		"architecture prompt": `{"title":"Linux","synopsis":"Course","targetAudience":"Developers","prerequisites":[],"goals":["Administer"],"acquiredSkills":["Shell"],"modules":[{"order":1,"title":"Basics","description":"Commands","keyLearningPoints":["shell"]}],"finalProject":{"title":"Server","description":"Deploy","constraints":[]}}`,
		"lessons prompt":      `{"moduleOrder":1,"moduleTitle":"Basics","lessons":[{"order":1,"title":"Filesystem","type":"theory","estimatedDuration":20,"learningGoal":"Navigate","requiresDiagram":false,"technicalKeywords":["ls"]}]}`,
		"content prompt":      `{"contentMarkdown":"# Filesystem","exercises":[],"quizzes":[]}`,
	}
	var mutex sync.Mutex
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/responses") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct {
			Instructions string `json:"instructions"`
			Store        bool   `json:"store"`
			Text         struct {
				Format struct {
					Type   string `json:"type"`
					Strict bool   `json:"strict"`
				} `json:"format"`
			} `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		output, ok := outputs[body.Instructions]
		if !ok {
			t.Errorf("unknown instructions: %q", body.Instructions)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body.Store || body.Text.Format.Type != "json_schema" || !body.Text.Format.Strict {
			t.Errorf("structured output request is not strict: %+v", body)
		}
		mutex.Lock()
		requests++
		mutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "resp_test", "object": "response", "created_at": 1, "status": "completed", "model": "test-model",
			"output": []any{map[string]any{
				"id": "msg_test", "type": "message", "status": "completed", "role": "assistant",
				"content": []any{map[string]any{"type": "output_text", "text": output, "annotations": []any{}}},
			}},
		})
	}))
	defer server.Close()

	client := openaisdk.NewClient(
		option.WithAPIKey("test-key"), option.WithBaseURL(server.URL), option.WithMaxRetries(0),
	)
	prompts := testPromptStore{values: map[string]string{
		"analysis": "analysis prompt", "architecture": "architecture prompt",
		"lessons": "lessons prompt", "lesson-content": "content prompt",
	}}
	generator := NewCourseAIGenerator(&client, prompts, Config{Model: "test-model", MaxOutputTokens: 500})
	ctx := context.Background()

	analysis, err := generator.AnalyzePrompt(ctx, contract.AnalysisInput{Prompt: "Linux"})
	if err != nil || analysis.Summary.SuggestedTitle == nil || *analysis.Summary.SuggestedTitle != "Linux" {
		t.Fatalf("AnalyzePrompt() = %+v, %v", analysis, err)
	}
	request := domain.GenerationRequest{ID: uuid.New(), InitialUserPrompt: "Linux"}
	architecture, err := generator.GenerateArchitecture(ctx, contract.ArchitectureInput{
		Request: request, Title: "Linux", Synopsis: "Course", CurrentLevel: domain.LevelBeginner,
		TargetLevel: domain.LevelAdvanced, Goals: []string{"Administer"}, Language: domain.CourseLanguageEN,
	})
	if err != nil || len(architecture.Course.Modules) != 1 {
		t.Fatalf("GenerateArchitecture() = %+v, %v", architecture, err)
	}
	module := architecture.Course.Modules[0]
	module.ID = uuid.New()
	lessons, err := generator.GenerateLessonPlan(ctx, contract.LessonPlanInput{Course: architecture.Course, Module: module})
	if err != nil || len(lessons.Lessons) != 1 {
		t.Fatalf("GenerateLessonPlan() = %+v, %v", lessons, err)
	}
	lesson := lessons.Lessons[0]
	lesson.ID = uuid.New()
	lesson.ModuleID = module.ID
	content, err := generator.GenerateLessonContent(ctx, contract.LessonContentInput{Course: architecture.Course, Module: module, Lesson: lesson})
	if err != nil || content.ContentMarkdown != "# Filesystem" || content.Lesson.ContentMarkdown == nil {
		t.Fatalf("GenerateLessonContent() = %+v, %v", content, err)
	}
	mutex.Lock()
	defer mutex.Unlock()
	if requests != 4 {
		t.Fatalf("OpenAI requests = %d, want 4", requests)
	}
}

func TestCallStructuredJSONValidatesPromptStore(t *testing.T) {
	t.Parallel()

	client := openaisdk.NewClient(option.WithAPIKey("test-key"))
	generator := NewCourseAIGenerator(&client, nil, Config{})
	if _, err := generator.callStructuredJSON(context.Background(), "analysis", map[string]string{}, "schema", analysisSchema()); err != ErrMissingPromptStore {
		t.Fatalf("missing store error = %v", err)
	}
	generator.prompts = testPromptStore{values: map[string]string{}}
	if _, err := generator.callStructuredJSON(context.Background(), "analysis", map[string]string{}, "schema", analysisSchema()); err == nil || !strings.Contains(err.Error(), ErrPromptNotFound.Error()) {
		t.Fatalf("missing prompt error = %v", err)
	}
}

type testPromptStore struct{ values map[string]string }

func (s testPromptStore) Get(name string) (string, bool) {
	value, ok := s.values[name]
	return value, ok
}

func (s testPromptStore) Names() []string {
	names := make([]string, 0, len(s.values))
	for name := range s.values {
		names = append(names, name)
	}
	return names
}
