package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const userColumns = `id, username, password, created_at, updated_at`

const generationRequestColumns = `
	id,
	initial_user_prompt,
	pipeline_status::text,
	current_step,
	progress_percent,
	failure_message,
	started_at,
	completed_at,
	is_out_of_scope,
	error_message,
	warning_message,
	suggested_title,
	short_synopsis,
	detected_current_level::text,
	detected_target_level::text,
	detected_goal,
	detected_language::text,
	clarification_questions,
	created_at,
	updated_at
`

const courseColumns = `
	id,
	request_id,
	language::text,
	status::text,
	initial_user_prompt,
	title,
	synopsis,
	target_audience,
	current_level::text,
	target_level::text,
	prerequisites,
	goals,
	acquired_skills,
	final_project_title,
	final_project_description,
	final_project_constraints,
	created_at,
	updated_at
`

const moduleColumns = `
	id,
	course_id,
	module_order,
	title,
	description,
	key_learning_points,
	created_at,
	updated_at
`

const lessonColumns = `
	id,
	module_id,
	lesson_order,
	title,
	type::text,
	estimated_duration_minutes,
	learning_goal,
	requires_diagram,
	technical_keywords,
	content_markdown,
	created_at,
	updated_at
`

func scanUser(row pgx.Row) (domain.User, error) {
	var user domain.User
	var username string
	if err := row.Scan(
		&user.ID,
		&username,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return domain.User{}, err
	}
	user.Username = domain.Username(username)
	return user, user.Validate()
}

func scanGenerationRequest(row pgx.Row) (domain.GenerationRequest, error) {
	var request domain.GenerationRequest
	var pipelineStatus string
	var currentStep pgtype.Text
	var failureMessage pgtype.Text
	var startedAt pgtype.Timestamp
	var completedAt pgtype.Timestamp
	var errorMessage pgtype.Text
	var warningMessage pgtype.Text
	var suggestedTitle pgtype.Text
	var shortSynopsis pgtype.Text
	var detectedCurrentLevel pgtype.Text
	var detectedTargetLevel pgtype.Text
	var detectedGoal pgtype.Text
	var detectedLanguage pgtype.Text
	var clarificationQuestionsJSON []byte

	if err := row.Scan(
		&request.ID,
		&request.InitialUserPrompt,
		&pipelineStatus,
		&currentStep,
		&request.ProgressPercent,
		&failureMessage,
		&startedAt,
		&completedAt,
		&request.IsOutOfScope,
		&errorMessage,
		&warningMessage,
		&suggestedTitle,
		&shortSynopsis,
		&detectedCurrentLevel,
		&detectedTargetLevel,
		&detectedGoal,
		&detectedLanguage,
		&clarificationQuestionsJSON,
		&request.CreatedAt,
		&request.UpdatedAt,
	); err != nil {
		return domain.GenerationRequest{}, err
	}

	status, err := domain.ParseGenerationPipelineStatus(pipelineStatus)
	if err != nil {
		return domain.GenerationRequest{}, err
	}
	request.PipelineStatus = status
	request.CurrentStep = textPtr(currentStep)
	request.FailureMessage = textPtr(failureMessage)
	request.StartedAt = timestampPtr(startedAt)
	request.CompletedAt = timestampPtr(completedAt)
	request.ErrorMessage = textPtr(errorMessage)
	request.WarningMessage = textPtr(warningMessage)
	request.SuggestedTitle = textPtr(suggestedTitle)
	request.ShortSynopsis = textPtr(shortSynopsis)
	request.DetectedGoal = textPtr(detectedGoal)

	if detectedCurrentLevel.Valid {
		level, err := domain.ParseLevel(detectedCurrentLevel.String)
		if err != nil {
			return domain.GenerationRequest{}, err
		}
		request.DetectedCurrentLevel = &level
	}
	if detectedTargetLevel.Valid {
		level, err := domain.ParseLevel(detectedTargetLevel.String)
		if err != nil {
			return domain.GenerationRequest{}, err
		}
		request.DetectedTargetLevel = &level
	}
	if detectedLanguage.Valid {
		language, err := domain.ParseCourseLanguage(detectedLanguage.String)
		if err != nil {
			return domain.GenerationRequest{}, err
		}
		request.DetectedLanguage = &language
	}

	questions, err := clarificationQuestionsFromJSON(clarificationQuestionsJSON)
	if err != nil {
		return domain.GenerationRequest{}, err
	}
	request.ClarificationQuestions = questions

	return request, request.Validate()
}

func scanCourse(row pgx.Row) (domain.Course, error) {
	var course domain.Course
	var language string
	var status string
	var targetAudience pgtype.Text
	var currentLevel string
	var targetLevel string
	var prerequisitesJSON []byte
	var goalsJSON []byte
	var acquiredSkillsJSON []byte
	var finalProjectTitle pgtype.Text
	var finalProjectDescription pgtype.Text
	var finalProjectConstraintsJSON []byte

	if err := row.Scan(
		&course.ID,
		&course.RequestID,
		&language,
		&status,
		&course.InitialUserPrompt,
		&course.Title,
		&course.Synopsis,
		&targetAudience,
		&currentLevel,
		&targetLevel,
		&prerequisitesJSON,
		&goalsJSON,
		&acquiredSkillsJSON,
		&finalProjectTitle,
		&finalProjectDescription,
		&finalProjectConstraintsJSON,
		&course.CreatedAt,
		&course.UpdatedAt,
	); err != nil {
		return domain.Course{}, err
	}

	parsedLanguage, err := domain.ParseCourseLanguage(language)
	if err != nil {
		return domain.Course{}, err
	}
	parsedStatus, err := domain.ParseCourseGenerationStatus(status)
	if err != nil {
		return domain.Course{}, err
	}
	parsedCurrentLevel, err := domain.ParseLevel(currentLevel)
	if err != nil {
		return domain.Course{}, err
	}
	parsedTargetLevel, err := domain.ParseLevel(targetLevel)
	if err != nil {
		return domain.Course{}, err
	}

	course.Language = parsedLanguage
	course.Status = parsedStatus
	course.TargetAudience = textPtr(targetAudience)
	course.CurrentLevel = parsedCurrentLevel
	course.TargetLevel = parsedTargetLevel
	course.FinalProjectTitle = textPtr(finalProjectTitle)
	course.FinalProjectDescription = textPtr(finalProjectDescription)

	if course.Prerequisites, err = stringSliceFromJSON(prerequisitesJSON); err != nil {
		return domain.Course{}, err
	}
	if course.Goals, err = stringSliceFromJSON(goalsJSON); err != nil {
		return domain.Course{}, err
	}
	if course.AcquiredSkills, err = stringSliceFromJSON(acquiredSkillsJSON); err != nil {
		return domain.Course{}, err
	}
	if course.FinalProjectConstraints, err = stringSliceFromJSON(finalProjectConstraintsJSON); err != nil {
		return domain.Course{}, err
	}

	return course, nil
}

func scanModule(row pgx.Row) (domain.Module, error) {
	var module domain.Module
	var keyLearningPointsJSON []byte

	if err := row.Scan(
		&module.ID,
		&module.CourseID,
		&module.Order,
		&module.Title,
		&module.Description,
		&keyLearningPointsJSON,
		&module.CreatedAt,
		&module.UpdatedAt,
	); err != nil {
		return domain.Module{}, err
	}

	points, err := stringSliceFromJSON(keyLearningPointsJSON)
	if err != nil {
		return domain.Module{}, err
	}
	module.KeyLearningPoints = points

	return module, nil
}

func scanLesson(row pgx.Row) (domain.Lesson, error) {
	var lesson domain.Lesson
	var lessonType string
	var technicalKeywordsJSON []byte
	var contentMarkdown pgtype.Text

	if err := row.Scan(
		&lesson.ID,
		&lesson.ModuleID,
		&lesson.Order,
		&lesson.Title,
		&lessonType,
		&lesson.EstimatedDurationMinutes,
		&lesson.LearningGoal,
		&lesson.RequiresDiagram,
		&technicalKeywordsJSON,
		&contentMarkdown,
		&lesson.CreatedAt,
		&lesson.UpdatedAt,
	); err != nil {
		return domain.Lesson{}, err
	}

	parsedType, err := domain.ParseLessonType(lessonType)
	if err != nil {
		return domain.Lesson{}, err
	}
	lesson.Type = parsedType
	lesson.ContentMarkdown = textPtr(contentMarkdown)

	keywords, err := stringSliceFromJSON(technicalKeywordsJSON)
	if err != nil {
		return domain.Lesson{}, err
	}
	lesson.TechnicalKeywords = keywords

	return lesson, lesson.Validate()
}

func scanCourses(rows pgx.Rows) ([]domain.Course, error) {
	defer rows.Close()
	courses := make([]domain.Course, 0)
	for rows.Next() {
		course, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		courses = append(courses, course)
	}
	return courses, rows.Err()
}

func scanModules(rows pgx.Rows) ([]domain.Module, error) {
	defer rows.Close()
	modules := make([]domain.Module, 0)
	for rows.Next() {
		module, err := scanModule(rows)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}
	return modules, rows.Err()
}

func scanLessons(rows pgx.Rows) ([]domain.Lesson, error) {
	defer rows.Close()
	lessons := make([]domain.Lesson, 0)
	for rows.Next() {
		lesson, err := scanLesson(rows)
		if err != nil {
			return nil, err
		}
		lessons = append(lessons, lesson)
	}
	return lessons, rows.Err()
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

func textValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func timeValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}

func timestampPtr(value pgtype.Timestamp) *time.Time {
	if !value.Valid {
		return nil
	}
	timestamp := value.Time
	return &timestamp
}

func levelValue(value *domain.Level) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func languageValue(value *domain.CourseLanguage) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func uuidValue(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}
