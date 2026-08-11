package domain

import (
	"fmt"
	"strings"
)

type CourseLanguage string

const (
	CourseLanguageFR CourseLanguage = "fr"
	CourseLanguageEN CourseLanguage = "en"
)

func ParseCourseLanguage(value string) (CourseLanguage, error) {
	language := CourseLanguage(strings.ToLower(strings.TrimSpace(value)))
	if err := language.Validate(); err != nil {
		return "", err
	}
	return language, nil
}

func (l CourseLanguage) Validate() error {
	switch l {
	case CourseLanguageFR, CourseLanguageEN:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidCourseLanguage, l)
	}
}

type Level string

const (
	LevelBeginner     Level = "beginner"
	LevelIntermediate Level = "intermediate"
	LevelAdvanced     Level = "advanced"
	LevelExpert       Level = "expert"
	LevelUnknown      Level = "unknown"
)

func ParseLevel(value string) (Level, error) {
	level := Level(strings.ToLower(strings.TrimSpace(value)))
	if err := level.Validate(); err != nil {
		return "", err
	}
	return level, nil
}

func (l Level) Validate() error {
	switch l {
	case LevelBeginner, LevelIntermediate, LevelAdvanced, LevelExpert, LevelUnknown:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidLevel, l)
	}
}

type LessonType string

const (
	LessonTypeTheory   LessonType = "theory"
	LessonTypePractice LessonType = "practice"
	LessonTypeMixed    LessonType = "mixed"
	LessonTypeQuiz     LessonType = "quiz"
)

func ParseLessonType(value string) (LessonType, error) {
	lessonType := LessonType(strings.ToLower(strings.TrimSpace(value)))
	if err := lessonType.Validate(); err != nil {
		return "", err
	}
	return lessonType, nil
}

func (t LessonType) Validate() error {
	switch t {
	case LessonTypeTheory, LessonTypePractice, LessonTypeMixed, LessonTypeQuiz:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidLessonType, t)
	}
}

type Difficulty string

const (
	DifficultyBeginner     Difficulty = "beginner"
	DifficultyIntermediate Difficulty = "intermediate"
	DifficultyAdvanced     Difficulty = "advanced"
)

func ParseDifficulty(value string) (Difficulty, error) {
	difficulty := Difficulty(strings.ToLower(strings.TrimSpace(value)))
	if err := difficulty.Validate(); err != nil {
		return "", err
	}
	return difficulty, nil
}

func (d Difficulty) Validate() error {
	switch d {
	case DifficultyBeginner, DifficultyIntermediate, DifficultyAdvanced:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidDifficulty, d)
	}
}

type ExerciseType string

const (
	ExerciseTypeGuidedLab     ExerciseType = "guided_lab"
	ExerciseTypeCoding        ExerciseType = "coding"
	ExerciseTypeDebugging     ExerciseType = "debugging"
	ExerciseTypeConfiguration ExerciseType = "configuration"
	ExerciseTypeScenario      ExerciseType = "scenario"
	ExerciseTypeWrittenAnswer ExerciseType = "written_answer"
	ExerciseTypeCommandLine   ExerciseType = "command_line"
	ExerciseTypeMixed         ExerciseType = "mixed"
)

func ParseExerciseType(value string) (ExerciseType, error) {
	exerciseType := ExerciseType(strings.ToLower(strings.TrimSpace(value)))
	if err := exerciseType.Validate(); err != nil {
		return "", err
	}
	return exerciseType, nil
}

func (t ExerciseType) Validate() error {
	switch t {
	case ExerciseTypeGuidedLab,
		ExerciseTypeCoding,
		ExerciseTypeDebugging,
		ExerciseTypeConfiguration,
		ExerciseTypeScenario,
		ExerciseTypeWrittenAnswer,
		ExerciseTypeCommandLine,
		ExerciseTypeMixed:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidExerciseType, t)
	}
}

type QuizType string

const (
	QuizTypeSingleChoice   QuizType = "single_choice"
	QuizTypeMultipleChoice QuizType = "multiple_choice"
	QuizTypeTrueFalse      QuizType = "true_false"
	QuizTypeShortAnswer    QuizType = "short_answer"
	QuizTypeMixed          QuizType = "mixed"
)

func ParseQuizType(value string) (QuizType, error) {
	quizType := QuizType(strings.ToLower(strings.TrimSpace(value)))
	if err := quizType.Validate(); err != nil {
		return "", err
	}
	return quizType, nil
}

func (t QuizType) Validate() error {
	switch t {
	case QuizTypeSingleChoice, QuizTypeMultipleChoice, QuizTypeTrueFalse, QuizTypeShortAnswer, QuizTypeMixed:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidQuizType, t)
	}
}

type QuizQuestionType string

const (
	QuizQuestionTypeSingleChoice   QuizQuestionType = "single_choice"
	QuizQuestionTypeMultipleChoice QuizQuestionType = "multiple_choice"
	QuizQuestionTypeTrueFalse      QuizQuestionType = "true_false"
	QuizQuestionTypeShortAnswer    QuizQuestionType = "short_answer"
)

func ParseQuizQuestionType(value string) (QuizQuestionType, error) {
	questionType := QuizQuestionType(strings.ToLower(strings.TrimSpace(value)))
	if err := questionType.Validate(); err != nil {
		return "", err
	}
	return questionType, nil
}

func (t QuizQuestionType) Validate() error {
	switch t {
	case QuizQuestionTypeSingleChoice,
		QuizQuestionTypeMultipleChoice,
		QuizQuestionTypeTrueFalse,
		QuizQuestionTypeShortAnswer:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidQuizQuestionType, t)
	}
}

type CourseGenerationStatus string

const (
	CourseStatusAnalysisPending        CourseGenerationStatus = "analysis_pending"
	CourseStatusNeedsClarification     CourseGenerationStatus = "needs_clarification"
	CourseStatusAnalysisCompleted      CourseGenerationStatus = "analysis_completed"
	CourseStatusArchitectureGenerating CourseGenerationStatus = "architecture_generating"
	CourseStatusStructureGenerated     CourseGenerationStatus = "structure_generated"
	CourseStatusLessonsGenerating      CourseGenerationStatus = "lessons_generating"
	CourseStatusLessonsGenerated       CourseGenerationStatus = "lessons_generated"
	CourseStatusContentGenerating      CourseGenerationStatus = "content_generating"
	CourseStatusCompleted              CourseGenerationStatus = "completed"
	CourseStatusFailed                 CourseGenerationStatus = "failed"
)

func ParseCourseGenerationStatus(value string) (CourseGenerationStatus, error) {
	status := CourseGenerationStatus(strings.ToLower(strings.TrimSpace(value)))
	if err := status.Validate(); err != nil {
		return "", err
	}
	return status, nil
}

func (s CourseGenerationStatus) Validate() error {
	switch s {
	case CourseStatusAnalysisPending,
		CourseStatusNeedsClarification,
		CourseStatusAnalysisCompleted,
		CourseStatusArchitectureGenerating,
		CourseStatusStructureGenerated,
		CourseStatusLessonsGenerating,
		CourseStatusLessonsGenerated,
		CourseStatusContentGenerating,
		CourseStatusCompleted,
		CourseStatusFailed:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidCourseStatus, s)
	}
}

func (s CourseGenerationStatus) IsTerminal() bool {
	return s == CourseStatusCompleted || s == CourseStatusFailed
}

func (s CourseGenerationStatus) CanTransitionTo(next CourseGenerationStatus) bool {
	if s == next {
		return true
	}
	if s.IsTerminal() {
		return false
	}

	allowedTransitions := map[CourseGenerationStatus][]CourseGenerationStatus{
		CourseStatusAnalysisPending: {
			CourseStatusNeedsClarification,
			CourseStatusAnalysisCompleted,
			CourseStatusFailed,
		},
		CourseStatusNeedsClarification: {
			CourseStatusAnalysisPending,
			CourseStatusFailed,
		},
		CourseStatusAnalysisCompleted: {
			CourseStatusArchitectureGenerating,
			CourseStatusFailed,
		},
		CourseStatusArchitectureGenerating: {
			CourseStatusStructureGenerated,
			CourseStatusFailed,
		},
		CourseStatusStructureGenerated: {
			CourseStatusLessonsGenerating,
			CourseStatusFailed,
		},
		CourseStatusLessonsGenerating: {
			CourseStatusLessonsGenerated,
			CourseStatusFailed,
		},
		CourseStatusLessonsGenerated: {
			CourseStatusContentGenerating,
			CourseStatusFailed,
		},
		CourseStatusContentGenerating: {
			CourseStatusCompleted,
			CourseStatusFailed,
		},
	}

	for _, allowed := range allowedTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

type GenerationPipelineStatus string

const (
	PipelineStatusQueued    GenerationPipelineStatus = "queued"
	PipelineStatusRunning   GenerationPipelineStatus = "running"
	PipelineStatusCompleted GenerationPipelineStatus = "completed"
	PipelineStatusFailed    GenerationPipelineStatus = "failed"
)

func ParseGenerationPipelineStatus(value string) (GenerationPipelineStatus, error) {
	status := GenerationPipelineStatus(strings.ToLower(strings.TrimSpace(value)))
	if err := status.Validate(); err != nil {
		return "", err
	}
	return status, nil
}

func (s GenerationPipelineStatus) Validate() error {
	switch s {
	case PipelineStatusQueued, PipelineStatusRunning, PipelineStatusCompleted, PipelineStatusFailed:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidGenerationStatus, s)
	}
}

func (s GenerationPipelineStatus) IsTerminal() bool {
	return s == PipelineStatusCompleted || s == PipelineStatusFailed
}

func (s GenerationPipelineStatus) CanTransitionTo(next GenerationPipelineStatus) bool {
	if s == next {
		return true
	}
	if s.IsTerminal() {
		return false
	}

	switch s {
	case PipelineStatusQueued:
		return next == PipelineStatusRunning || next == PipelineStatusFailed
	case PipelineStatusRunning:
		return next == PipelineStatusCompleted || next == PipelineStatusFailed
	default:
		return false
	}
}
