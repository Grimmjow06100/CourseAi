package postgres

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

func TestSQLCEntityMappers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 13, 10, 0, 0, 0, time.UTC)
	requestID := uuid.New()
	courseID := uuid.New()
	moduleID := uuid.New()
	lessonID := uuid.New()
	answer := "Linux"

	request, err := generationRequestFromSQLC(dbsqlc.GenerationRequest{
		ID: requestID, ClerkUserID: "user_test", InitialUserPrompt: "Linux", PipelineStatus: dbsqlc.GenerationPipelineStatus("queued"),
		ClarificationQuestions: json.RawMessage(`[]`), CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || request.ID != requestID {
		t.Fatalf("generationRequestFromSQLC() = %+v, %v", request, err)
	}

	job, err := generationJobFromSQLC(dbsqlc.GenerationJob{
		ID: uuid.New(), RequestID: requestID, Kind: dbsqlc.GenerationJobKind("analysis"),
		Status: dbsqlc.GenerationJobStatus("queued"), IdempotencyKey: "analysis", Payload: json.RawMessage(`{}`),
		MaxAttempts: 3, AvailableAt: now, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || job.Kind != domain.GenerationJobKindAnalysis {
		t.Fatalf("generationJobFromSQLC() = %+v, %v", job, err)
	}

	courseRow := dbsqlc.Course{
		ID: courseID, RequestID: requestID, ClerkUserID: "user_test", Language: dbsqlc.CourseLanguage("fr"),
		Status: dbsqlc.CourseGenerationStatus("analysis_completed"), InitialUserPrompt: "Linux",
		Title: "Linux", Synopsis: "Course", CurrentLevel: dbsqlc.Level("beginner"), TargetLevel: dbsqlc.Level("advanced"),
		Prerequisites: json.RawMessage(`[]`), Goals: json.RawMessage(`["admin"]`), AcquiredSkills: json.RawMessage(`[]`),
		FinalProjectConstraints: json.RawMessage(`[]`), CreatedAt: now, UpdatedAt: now,
	}
	course, err := courseFromSQLC(courseRow)
	if err != nil || course.ID != courseID || !reflect.DeepEqual(course.Goals, []string{"admin"}) {
		t.Fatalf("courseFromSQLC() = %+v, %v", course, err)
	}

	module, err := moduleFromSQLC(dbsqlc.Module{
		ID: moduleID, CourseID: courseID, ModuleOrder: 1, Title: "Basics", Description: "Description",
		KeyLearningPoints: json.RawMessage(`["shell"]`), CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || module.ID != moduleID || !reflect.DeepEqual(module.KeyLearningPoints, []string{"shell"}) {
		t.Fatalf("moduleFromSQLC() = %+v, %v", module, err)
	}

	lesson, err := lessonFromSQLC(dbsqlc.Lesson{
		ID: lessonID, ModuleID: moduleID, LessonOrder: 1, Title: "Commands", Type: dbsqlc.LessonType("theory"),
		EstimatedDurationMinutes: 20, LearningGoal: "Navigate", TechnicalKeywords: json.RawMessage(`["ls"]`),
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || lesson.ID != lessonID || lesson.Type != domain.LessonTypeTheory {
		t.Fatalf("lessonFromSQLC() = %+v, %v", lesson, err)
	}

	exercise, err := exerciseFromSQLC(dbsqlc.LessonExercise{
		ID: uuid.New(), LessonID: lessonID, Type: dbsqlc.ExerciseType("command_line"), Difficulty: dbsqlc.ActivityDifficulty("beginner"),
		Title: "Exercise", Objective: "Run", InstructionsMarkdown: "Run it", ContentMarkdown: "ls", CorrectionMarkdown: "files",
		Payload: json.RawMessage(`{}`), CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || exercise.Type != domain.ExerciseTypeCommandLine {
		t.Fatalf("exerciseFromSQLC() = %+v, %v", exercise, err)
	}

	questions, err := quizQuestionsJSON([]domain.QuizQuestion{{
		Order: 1, Type: domain.QuizQuestionTypeShortAnswer, Question: "System?",
		Answer: domain.QuizAnswer{Answer: &answer}, Correction: "Linux",
	}})
	if err != nil {
		t.Fatalf("quiz questions JSON: %v", err)
	}
	quiz, err := quizFromSQLC(dbsqlc.LessonQuiz{
		ID: uuid.New(), LessonID: lessonID, Type: dbsqlc.QuizType("short_answer"), Difficulty: dbsqlc.ActivityDifficulty("beginner"),
		Title: "Quiz", Objective: "Check", Questions: json.RawMessage(questions), CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || len(quiz.Questions) != 1 {
		t.Fatalf("quizFromSQLC() = %+v, %v", quiz, err)
	}

	if courses, err := coursesFromSQLC([]dbsqlc.Course{courseRow}); err != nil || len(courses) != 1 {
		t.Fatalf("coursesFromSQLC() = %+v, %v", courses, err)
	}
}

func TestMapperHelpersAndInvalidJSON(t *testing.T) {
	t.Parallel()

	level := domain.LevelAdvanced
	language := domain.CourseLanguageEN
	status := domain.CourseStatusCompleted
	if sqlcLevelPtr(&level) == nil || sqlcLanguagePtr(&language) == nil || sqlcStatusPtr(&status) == nil {
		t.Fatal("enum pointer conversion returned nil")
	}
	if sqlcLevelPtr(nil) != nil || sqlcLanguagePtr(nil) != nil || sqlcStatusPtr(nil) != nil {
		t.Fatal("nil enum pointer conversion returned a value")
	}

	values, err := stringSliceJSON(nil)
	if err != nil || values != "[]" {
		t.Fatalf("stringSliceJSON(nil) = %q, %v", values, err)
	}
	if decoded, err := stringSliceFromJSON(nil); err != nil || decoded != nil {
		t.Fatalf("stringSliceFromJSON(nil) = %#v, %v", decoded, err)
	}
	if _, err := stringSliceFromJSON([]byte(`{"invalid":true}`)); err == nil {
		t.Fatal("expected invalid string slice JSON error")
	}
	questionsJSON, err := clarificationQuestionsJSON(nil)
	if err != nil || questionsJSON != "[]" {
		t.Fatalf("clarificationQuestionsJSON(nil) = %q, %v", questionsJSON, err)
	}
	if questions, err := clarificationQuestionsFromJSON(nil); err != nil || questions != nil {
		t.Fatalf("clarificationQuestionsFromJSON(nil) = %#v, %v", questions, err)
	}
	if _, err := clarificationQuestionsFromJSON([]byte(`{}`)); err == nil {
		t.Fatal("expected invalid clarification JSON error")
	}
	if payload, err := exercisePayloadFromJSON(nil); err != nil || payload == nil {
		t.Fatalf("exercisePayloadFromJSON(nil) = %#v, %v", payload, err)
	}
	if _, err := exercisePayloadFromJSON([]byte(`[]`)); err == nil {
		t.Fatal("expected invalid exercise payload JSON error")
	}
	if _, err := quizQuestionsFromJSON([]byte(`{"invalid":true}`)); err == nil {
		t.Fatal("expected invalid quiz questions JSON error")
	}
	if _, err := quizQuestionsFromJSONPayload([]quizQuestionJSON{{Type: "essay"}}); !errors.Is(err, domain.ErrInvalidQuizQuestionType) {
		t.Fatalf("invalid quiz question type error = %v", err)
	}

	id := uuid.New()
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	if got := uuidPointerFromPGType(pgID); got == nil || *got != id {
		t.Fatalf("uuidPointerFromPGType() = %v", got)
	}
	if uuidPointerFromPGType(pgtype.UUID{}) != nil {
		t.Fatal("invalid PostgreSQL UUID should map to nil")
	}
}

func TestCoursePersistenceParams(t *testing.T) {
	t.Parallel()

	course := domain.Course{
		ID: uuid.New(), RequestID: uuid.New(), Language: domain.CourseLanguageFR,
		Status: domain.CourseStatusAnalysisCompleted, InitialUserPrompt: "Linux", Title: "Linux", Synopsis: "Course",
		CurrentLevel: domain.LevelBeginner, TargetLevel: domain.LevelAdvanced,
		Prerequisites: []string{"shell"}, Goals: []string{"admin"}, AcquiredSkills: []string{"bash"}, FinalProjectConstraints: []string{"secure"},
		RawArchitectureOutput: json.RawMessage(`{"title":"Linux"}`), UpdatedAt: time.Now(), CreatedAt: time.Now(),
	}
	created, err := createCourseParams(course)
	if err != nil {
		t.Fatalf("createCourseParams() error = %v", err)
	}
	updated, err := updateCourseParams(course)
	if err != nil {
		t.Fatalf("updateCourseParams() error = %v", err)
	}
	if created.ID != course.ID || updated.ID != course.ID || string(created.Goals) != `["admin"]` || string(updated.RawArchitectureOutput) != `{"title":"Linux"}` {
		t.Fatalf("unexpected persistence params: create=%+v update=%+v", created, updated)
	}
}
