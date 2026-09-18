package postgres

import (
	"encoding/json"
	"fmt"

	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/jsonutil"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/pointer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type quizQuestionJSON struct {
	Order      int              `json:"order"`
	Type       string           `json:"type"`
	Question   string           `json:"question"`
	Options    []quizOptionJSON `json:"options"`
	Answer     quizAnswerJSON   `json:"answer"`
	Correction string           `json:"correction"`
}

type quizOptionJSON struct {
	Order int    `json:"order"`
	Text  string `json:"text"`
}

type quizAnswerJSON struct {
	Answer  *string  `json:"answer"`
	Answers []string `json:"answers"`
}

func generationRequestFromSQLC(row dbsqlc.GenerationRequest) (domain.GenerationRequest, error) {
	status, err := domain.ParseGenerationPipelineStatus(string(row.PipelineStatus))
	if err != nil {
		return domain.GenerationRequest{}, err
	}

	request := domain.GenerationRequest{
		GenerationAttempt:         int(row.GenerationAttempt),
		ID:                        row.ID,
		ClerkUserID:               row.ClerkUserID,
		InitialUserPrompt:         row.InitialUserPrompt,
		PipelineStatus:            status,
		CurrentStep:               row.CurrentStep,
		ProgressPercent:           int(row.ProgressPercent),
		FailureMessage:            row.FailureMessage,
		StartedAt:                 row.StartedAt,
		CompletedAt:               row.CompletedAt,
		IsOutOfScope:              row.IsOutOfScope,
		ErrorMessage:              row.ErrorMessage,
		WarningMessage:            row.WarningMessage,
		SuggestedTitle:            row.SuggestedTitle,
		ShortSynopsis:             row.ShortSynopsis,
		DetectedGoal:              row.DetectedGoal,
		RawAnalysisOutput:         rawJSONFromBytes(row.RawAnalysisOutput),
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
		AnalysisCompletedAt:       row.AnalysisCompletedAt,
		BriefConfirmedAt:          row.BriefConfirmedAt,
		ClarificationsSubmittedAt: row.ClarificationsSubmittedAt,
		ClarificationVersion:      int(row.ClarificationVersion),
	}

	if row.DetectedCurrentLevel != nil {
		level, err := domain.ParseLevel(string(*row.DetectedCurrentLevel))
		if err != nil {
			return domain.GenerationRequest{}, err
		}
		request.DetectedCurrentLevel = &level
	}
	if row.DetectedTargetLevel != nil {
		level, err := domain.ParseLevel(string(*row.DetectedTargetLevel))
		if err != nil {
			return domain.GenerationRequest{}, err
		}
		request.DetectedTargetLevel = &level
	}
	if row.DetectedLanguage != nil {
		language, err := domain.ParseCourseLanguage(string(*row.DetectedLanguage))
		if err != nil {
			return domain.GenerationRequest{}, err
		}
		request.DetectedLanguage = &language
	}

	request.ClarificationQuestions, err = clarificationQuestionsFromJSON(row.ClarificationQuestions)
	if err != nil {
		return domain.GenerationRequest{}, err
	}
	request.ClarificationAnswers, err = clarificationAnswersFromJSON(row.ClarificationAnswers)
	if err != nil {
		return domain.GenerationRequest{}, err
	}
	request.ConfirmedBrief, err = confirmedBriefFromSQLC(row)
	if err != nil {
		return domain.GenerationRequest{}, err
	}
	return request, request.Validate()
}

func generationJobFromSQLC(row dbsqlc.GenerationJob) (domain.GenerationJob, error) {
	kind, err := domain.ParseGenerationJobKind(string(row.Kind))
	if err != nil {
		return domain.GenerationJob{}, err
	}
	status, err := domain.ParseGenerationJobStatus(string(row.Status))
	if err != nil {
		return domain.GenerationJob{}, err
	}

	job := domain.GenerationJob{
		GenerationAttempt: int(row.GenerationAttempt),
		ID:                row.ID,
		RequestID:         row.RequestID,
		ParentJobID:       uuidPointerFromPGType(row.ParentJobID),
		Kind:              kind,
		Status:            status,
		TargetID:          uuidPointerFromPGType(row.TargetID),
		IdempotencyKey:    row.IdempotencyKey,
		Payload:           rawJSONFromBytes(row.Payload),
		Priority:          int(row.Priority),
		AttemptCount:      int(row.AttemptCount),
		MaxAttempts:       int(row.MaxAttempts),
		AvailableAt:       row.AvailableAt,
		LockedBy:          pointer.Clone(row.LockedBy),
		LockedUntil:       pointer.Clone(row.LockedUntil),
		StartedAt:         pointer.Clone(row.StartedAt),
		CompletedAt:       pointer.Clone(row.CompletedAt),
		LastErrorCode:     pointer.Clone(row.LastErrorCode),
		LastErrorMessage:  pointer.Clone(row.LastErrorMessage),
		FailureHandledAt:  pointer.Clone(row.FailureHandledAt),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
	return job, job.Validate()
}

func courseFromSQLC(row dbsqlc.Course) (domain.Course, error) {
	language, err := domain.ParseCourseLanguage(string(row.Language))
	if err != nil {
		return domain.Course{}, err
	}
	status, err := domain.ParseCourseGenerationStatus(string(row.Status))
	if err != nil {
		return domain.Course{}, err
	}
	currentLevel, err := domain.ParseLevel(string(row.CurrentLevel))
	if err != nil {
		return domain.Course{}, err
	}
	targetLevel, err := domain.ParseLevel(string(row.TargetLevel))
	if err != nil {
		return domain.Course{}, err
	}

	course := domain.Course{
		ID:                      row.ID,
		RequestID:               row.RequestID,
		ClerkUserID:             row.ClerkUserID,
		Language:                language,
		Status:                  status,
		InitialUserPrompt:       row.InitialUserPrompt,
		Title:                   row.Title,
		Synopsis:                row.Synopsis,
		TargetAudience:          row.TargetAudience,
		CurrentLevel:            currentLevel,
		TargetLevel:             targetLevel,
		FinalProjectTitle:       row.FinalProjectTitle,
		FinalProjectDescription: row.FinalProjectDescription,
		RawArchitectureOutput:   rawJSONFromBytes(row.RawArchitectureOutput),
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
	}

	if course.Prerequisites, err = stringSliceFromJSON(row.Prerequisites); err != nil {
		return domain.Course{}, err
	}
	if course.Goals, err = stringSliceFromJSON(row.Goals); err != nil {
		return domain.Course{}, err
	}
	if course.AcquiredSkills, err = stringSliceFromJSON(row.AcquiredSkills); err != nil {
		return domain.Course{}, err
	}
	if course.FinalProjectConstraints, err = stringSliceFromJSON(row.FinalProjectConstraints); err != nil {
		return domain.Course{}, err
	}
	return course, course.ValidateCourseOnly()
}

func moduleFromSQLC(row dbsqlc.Module) (domain.Module, error) {
	points, err := stringSliceFromJSON(row.KeyLearningPoints)
	if err != nil {
		return domain.Module{}, err
	}
	return domain.Module{
		ID:                   row.ID,
		CourseID:             row.CourseID,
		Order:                int(row.ModuleOrder),
		Title:                row.Title,
		Description:          row.Description,
		KeyLearningPoints:    points,
		RawLessonsPlanOutput: rawJSONFromBytes(row.RawLessonsPlanOutput),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}, nil
}

func lessonFromSQLC(row dbsqlc.Lesson) (domain.Lesson, error) {
	lessonType, err := domain.ParseLessonType(string(row.Type))
	if err != nil {
		return domain.Lesson{}, err
	}
	keywords, err := stringSliceFromJSON(row.TechnicalKeywords)
	if err != nil {
		return domain.Lesson{}, err
	}
	lesson := domain.Lesson{
		ID:                       row.ID,
		ModuleID:                 row.ModuleID,
		Order:                    int(row.LessonOrder),
		Title:                    row.Title,
		Type:                     lessonType,
		EstimatedDurationMinutes: int(row.EstimatedDurationMinutes),
		LearningGoal:             row.LearningGoal,
		RequiresDiagram:          row.RequiresDiagram,
		TechnicalKeywords:        keywords,
		ContentMarkdown:          row.ContentMarkdown,
		RawContentOutput:         rawJSONFromBytes(row.RawContentOutput),
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
	return lesson, lesson.Validate()
}

func exerciseFromSQLC(row dbsqlc.LessonExercise) (domain.Exercise, error) {
	exerciseType, err := domain.ParseExerciseType(string(row.Type))
	if err != nil {
		return domain.Exercise{}, err
	}
	difficulty, err := domain.ParseDifficulty(string(row.Difficulty))
	if err != nil {
		return domain.Exercise{}, err
	}
	payload, err := exercisePayloadFromJSON(row.Payload)
	if err != nil {
		return domain.Exercise{}, err
	}
	exercise := domain.Exercise{
		ID:                   row.ID,
		LessonID:             row.LessonID,
		Type:                 exerciseType,
		Difficulty:           difficulty,
		Title:                row.Title,
		Objective:            row.Objective,
		InstructionsMarkdown: row.InstructionsMarkdown,
		ContentMarkdown:      row.ContentMarkdown,
		CorrectionMarkdown:   row.CorrectionMarkdown,
		Payload:              payload,
		RawAIOutput:          rawJSONFromBytes(row.RawAiOutput),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
	return exercise, exercise.Validate()
}

func quizFromSQLC(row dbsqlc.LessonQuiz) (domain.Quiz, error) {
	quizType, err := domain.ParseQuizType(string(row.Type))
	if err != nil {
		return domain.Quiz{}, err
	}
	difficulty, err := domain.ParseDifficulty(string(row.Difficulty))
	if err != nil {
		return domain.Quiz{}, err
	}
	questions, err := quizQuestionsFromJSON(row.Questions)
	if err != nil {
		return domain.Quiz{}, err
	}
	quiz := domain.Quiz{
		ID:          row.ID,
		LessonID:    row.LessonID,
		Type:        quizType,
		Difficulty:  difficulty,
		Title:       row.Title,
		Objective:   row.Objective,
		Questions:   questions,
		RawAIOutput: rawJSONFromBytes(row.RawAiOutput),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	return quiz, quiz.Validate()
}

func coursesFromSQLC(rows []dbsqlc.Course) ([]domain.Course, error) {
	courses := make([]domain.Course, 0, len(rows))
	for _, row := range rows {
		course, err := courseFromSQLC(row)
		if err != nil {
			return nil, err
		}
		courses = append(courses, course)
	}
	return courses, nil
}

func generationJobsFromSQLC(rows []dbsqlc.GenerationJob) ([]domain.GenerationJob, error) {
	jobs := make([]domain.GenerationJob, 0, len(rows))
	for _, row := range rows {
		job, err := generationJobFromSQLC(row)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func modulesFromSQLC(rows []dbsqlc.Module) ([]domain.Module, error) {
	modules := make([]domain.Module, 0, len(rows))
	for _, row := range rows {
		module, err := moduleFromSQLC(row)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}
	return modules, nil
}

func lessonsFromSQLC(rows []dbsqlc.Lesson) ([]domain.Lesson, error) {
	lessons := make([]domain.Lesson, 0, len(rows))
	for _, row := range rows {
		lesson, err := lessonFromSQLC(row)
		if err != nil {
			return nil, err
		}
		lessons = append(lessons, lesson)
	}
	return lessons, nil
}

func exercisesFromSQLC(rows []dbsqlc.LessonExercise) ([]domain.Exercise, error) {
	exercises := make([]domain.Exercise, 0, len(rows))
	for _, row := range rows {
		exercise, err := exerciseFromSQLC(row)
		if err != nil {
			return nil, err
		}
		exercises = append(exercises, exercise)
	}
	return exercises, nil
}

func quizzesFromSQLC(rows []dbsqlc.LessonQuiz) ([]domain.Quiz, error) {
	quizzes := make([]domain.Quiz, 0, len(rows))
	for _, row := range rows {
		quiz, err := quizFromSQLC(row)
		if err != nil {
			return nil, err
		}
		quizzes = append(quizzes, quiz)
	}
	return quizzes, nil
}

func sqlcLevelPtr(value *domain.Level) *dbsqlc.Level {
	if value == nil {
		return nil
	}
	converted := dbsqlc.Level(*value)
	return &converted
}

func sqlcLanguagePtr(value *domain.CourseLanguage) *dbsqlc.CourseLanguage {
	if value == nil {
		return nil
	}
	converted := dbsqlc.CourseLanguage(*value)
	return &converted
}

func sqlcStatusPtr(value *domain.CourseGenerationStatus) *dbsqlc.CourseGenerationStatus {
	if value == nil {
		return nil
	}
	converted := dbsqlc.CourseGenerationStatus(*value)
	return &converted
}

func stringSliceJSON(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal string slice: %w", err)
	}
	return string(data), nil
}

func stringSliceFromJSON(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	values := make([]string, 0)
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("unmarshal string slice: %w", err)
	}
	return values, nil
}

func clarificationQuestionsJSON(values []domain.ClarificationQuestion) (string, error) {
	if values == nil {
		values = []domain.ClarificationQuestion{}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal clarification questions: %w", err)
	}
	return string(data), nil
}

func clarificationQuestionsFromJSON(data []byte) ([]domain.ClarificationQuestion, error) {
	if len(data) == 0 {
		return nil, nil
	}
	values := make([]domain.ClarificationQuestion, 0)
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("unmarshal clarification questions: %w", err)
	}
	return values, nil
}

func clarificationAnswersJSON(values []domain.ClarificationAnswer) (string, error) {
	if values == nil {
		values = []domain.ClarificationAnswer{}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal clarification answers: %w", err)
	}
	return string(data), nil
}

func clarificationAnswersFromJSON(data []byte) ([]domain.ClarificationAnswer, error) {
	if len(data) == 0 {
		return nil, nil
	}
	values := make([]domain.ClarificationAnswer, 0)
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("unmarshal clarification answers: %w", err)
	}
	return values, nil
}

func confirmedBriefFromSQLC(row dbsqlc.GenerationRequest) (*domain.GenerationBrief, error) {
	if row.ConfirmedTitle == nil && row.ConfirmedSynopsis == nil && row.ConfirmedCurrentLevel == nil && row.ConfirmedTargetLevel == nil && row.ConfirmedLanguage == nil {
		return nil, nil
	}
	if row.ConfirmedTitle == nil || row.ConfirmedSynopsis == nil || row.ConfirmedCurrentLevel == nil || row.ConfirmedTargetLevel == nil || row.ConfirmedLanguage == nil {
		return nil, domain.ErrGenerationBriefIncomplete
	}
	goals, err := stringSliceFromJSON(row.ConfirmedGoals)
	if err != nil {
		return nil, err
	}
	currentLevel, err := domain.ParseLevel(string(*row.ConfirmedCurrentLevel))
	if err != nil {
		return nil, err
	}
	targetLevel, err := domain.ParseLevel(string(*row.ConfirmedTargetLevel))
	if err != nil {
		return nil, err
	}
	language, err := domain.ParseCourseLanguage(string(*row.ConfirmedLanguage))
	if err != nil {
		return nil, err
	}
	brief := domain.GenerationBrief{
		Title:        *row.ConfirmedTitle,
		Synopsis:     *row.ConfirmedSynopsis,
		CurrentLevel: currentLevel,
		TargetLevel:  targetLevel,
		Goals:        goals,
		Language:     language,
	}
	if err := brief.Validate(); err != nil {
		return nil, err
	}
	return &brief, nil
}

func exercisePayloadJSON(value domain.ExercisePayload) (string, error) {
	if value == nil {
		value = domain.ExercisePayload{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal exercise payload: %w", err)
	}
	return string(data), nil
}

func exercisePayloadFromJSON(data []byte) (domain.ExercisePayload, error) {
	if len(data) == 0 {
		return domain.ExercisePayload{}, nil
	}
	payload := domain.ExercisePayload{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal exercise payload: %w", err)
	}
	return payload, nil
}

func quizQuestionsJSON(values []domain.QuizQuestion) (string, error) {
	if values == nil {
		values = []domain.QuizQuestion{}
	}
	data, err := json.Marshal(quizQuestionsToJSON(values))
	if err != nil {
		return "", fmt.Errorf("marshal quiz questions: %w", err)
	}
	return string(data), nil
}

func quizQuestionsFromJSON(data []byte) ([]domain.QuizQuestion, error) {
	if len(data) == 0 {
		return nil, nil
	}
	values := make([]quizQuestionJSON, 0)
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("unmarshal quiz questions: %w", err)
	}
	return quizQuestionsFromJSONPayload(values)
}

func quizQuestionsToJSON(values []domain.QuizQuestion) []quizQuestionJSON {
	questions := make([]quizQuestionJSON, 0, len(values))
	for _, value := range values {
		options := make([]quizOptionJSON, 0, len(value.Options))
		for _, option := range value.Options {
			options = append(options, quizOptionJSON{Order: option.Order, Text: option.Text})
		}
		answers := value.Answer.Answers
		if answers == nil {
			answers = []string{}
		}
		questions = append(questions, quizQuestionJSON{
			Order:      value.Order,
			Type:       string(value.Type),
			Question:   value.Question,
			Options:    options,
			Answer:     quizAnswerJSON{Answer: value.Answer.Answer, Answers: answers},
			Correction: value.Correction,
		})
	}
	return questions
}

func quizQuestionsFromJSONPayload(values []quizQuestionJSON) ([]domain.QuizQuestion, error) {
	questions := make([]domain.QuizQuestion, 0, len(values))
	for index, value := range values {
		questionType, err := domain.ParseQuizQuestionType(value.Type)
		if err != nil {
			return nil, fmt.Errorf("quiz question %d type: %w", index+1, err)
		}

		options := make([]domain.QuizOption, 0, len(value.Options))
		for _, option := range value.Options {
			options = append(options, domain.QuizOption{Order: option.Order, Text: option.Text})
		}
		questions = append(questions, domain.QuizQuestion{
			Order:      value.Order,
			Type:       questionType,
			Question:   value.Question,
			Options:    options,
			Answer:     domain.QuizAnswer{Answer: value.Answer.Answer, Answers: value.Answer.Answers},
			Correction: value.Correction,
		})
	}
	return questions, nil
}

func rawJSONValue(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}

func rawJSONFromBytes(data []byte) json.RawMessage {
	if len(data) == 0 {
		return nil
	}
	return jsonutil.Clone(json.RawMessage(data))
}

func uuidPointerFromPGType(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	return pointer.To(uuid.UUID(value.Bytes))
}
