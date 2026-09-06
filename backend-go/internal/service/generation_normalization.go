package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/jsonutil"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/textutil"
	"github.com/google/uuid"
)

func (s *CourseGeneratorService) normalizeGeneratedCourse(request domain.GenerationRequest, generatedCourse domain.Course) (domain.Course, error) {
	if isEmptyGeneratedCourse(generatedCourse) {
		return domain.Course{}, ErrMissingGeneratedCourse
	}

	now := s.now()
	modules := generatedCourse.Modules
	if generatedCourse.ID == uuid.Nil {
		course, err := domain.NewCourseAt(domain.NewCourseParams{
			RequestID:               request.ID,
			ClerkUserID:             request.ClerkUserID,
			Language:                generatedCourse.Language,
			InitialUserPrompt:       textutil.FirstNonBlank(generatedCourse.InitialUserPrompt, request.InitialUserPrompt),
			Title:                   generatedCourse.Title,
			Synopsis:                generatedCourse.Synopsis,
			TargetAudience:          generatedCourse.TargetAudience,
			CurrentLevel:            generatedCourse.CurrentLevel,
			TargetLevel:             generatedCourse.TargetLevel,
			Prerequisites:           generatedCourse.Prerequisites,
			Goals:                   generatedCourse.Goals,
			AcquiredSkills:          generatedCourse.AcquiredSkills,
			FinalProjectTitle:       generatedCourse.FinalProjectTitle,
			FinalProjectDescription: generatedCourse.FinalProjectDescription,
			FinalProjectConstraints: generatedCourse.FinalProjectConstraints,
		}, now)
		if err != nil {
			return domain.Course{}, err
		}
		generatedCourse = course
		generatedCourse.Modules = modules
	}

	if generatedCourse.RequestID == uuid.Nil {
		generatedCourse.RequestID = request.ID
	}
	if generatedCourse.RequestID != request.ID {
		return domain.Course{}, fmt.Errorf("%w: course request id does not match generation request id", domain.ErrInvalidCollection)
	}
	if strings.TrimSpace(generatedCourse.ClerkUserID) == "" {
		generatedCourse.ClerkUserID = request.ClerkUserID
	}
	if generatedCourse.ClerkUserID != request.ClerkUserID {
		return domain.Course{}, fmt.Errorf("%w: course clerk user id does not match generation request", domain.ErrInvalidCollection)
	}
	if strings.TrimSpace(generatedCourse.InitialUserPrompt) == "" {
		generatedCourse.InitialUserPrompt = request.InitialUserPrompt
	}
	if generatedCourse.CreatedAt.IsZero() {
		generatedCourse.CreatedAt = now
	}
	if generatedCourse.UpdatedAt.IsZero() {
		generatedCourse.UpdatedAt = now
	}
	if generatedCourse.Status == "" || generatedCourse.Status == domain.CourseStatusAnalysisPending {
		generatedCourse.Status = domain.CourseStatusAnalysisCompleted
	}
	if err := advanceGeneratedCourseToStructure(&generatedCourse, now); err != nil {
		return domain.Course{}, err
	}
	generatedCourse.Modules = modules
	return generatedCourse, generatedCourse.ValidateCourseOnly()
}

func advanceGeneratedCourseToStructure(course *domain.Course, now time.Time) error {
	switch course.Status {
	case domain.CourseStatusAnalysisCompleted:
		if err := course.MarkArchitectureGenerating(); err != nil {
			return err
		}
		course.UpdatedAt = now
		if err := course.MarkArchitectureGenerated(); err != nil {
			return err
		}
	case domain.CourseStatusArchitectureGenerating:
		if err := course.MarkArchitectureGenerated(); err != nil {
			return err
		}
	case domain.CourseStatusStructureGenerated:
	default:
		return fmt.Errorf("%w: unexpected generated course status %s", domain.ErrInvalidCourseStatus, course.Status)
	}
	course.UpdatedAt = now
	return nil
}

func (s *CourseGeneratorService) normalizeGeneratedModules(courseID uuid.UUID, modules []domain.Module) ([]domain.Module, error) {
	if len(modules) == 0 {
		return nil, ErrMissingGeneratedModules
	}

	now := s.now()
	normalizedModules := make([]domain.Module, 0, len(modules))
	for _, module := range modules {
		if module.ID == uuid.Nil {
			newModule, err := domain.NewModuleAt(domain.NewModuleParams{
				CourseID:          courseID,
				Order:             module.Order,
				Title:             module.Title,
				Description:       module.Description,
				KeyLearningPoints: module.KeyLearningPoints,
			}, now)
			if err != nil {
				return nil, err
			}
			module = newModule
		}
		if module.CourseID == uuid.Nil {
			module.CourseID = courseID
		}
		if module.CourseID != courseID {
			return nil, fmt.Errorf("%w: module course id does not match course id", domain.ErrInvalidCollection)
		}
		if module.CreatedAt.IsZero() {
			module.CreatedAt = now
		}
		if module.UpdatedAt.IsZero() {
			module.UpdatedAt = now
		}
		module.Lessons = nil
		if err := module.Validate(); err != nil {
			return nil, err
		}
		normalizedModules = append(normalizedModules, module)
	}
	return normalizedModules, nil
}

func (s *CourseGeneratorService) normalizeGeneratedLessons(moduleID uuid.UUID, lessons []domain.Lesson) ([]domain.Lesson, error) {
	if len(lessons) == 0 {
		return nil, ErrMissingGeneratedLessons
	}

	now := s.now()
	normalizedLessons := make([]domain.Lesson, 0, len(lessons))
	for _, lesson := range lessons {
		if lesson.ID == uuid.Nil {
			newLesson, err := domain.NewLessonAt(domain.NewLessonParams{
				ModuleID:                 moduleID,
				Order:                    lesson.Order,
				Title:                    lesson.Title,
				Type:                     lesson.Type,
				EstimatedDurationMinutes: lesson.EstimatedDurationMinutes,
				LearningGoal:             lesson.LearningGoal,
				RequiresDiagram:          lesson.RequiresDiagram,
				TechnicalKeywords:        lesson.TechnicalKeywords,
			}, now)
			if err != nil {
				return nil, err
			}
			lesson = newLesson
		}
		if lesson.ModuleID == uuid.Nil {
			lesson.ModuleID = moduleID
		}
		if lesson.ModuleID != moduleID {
			return nil, fmt.Errorf("%w: lesson module id does not match module id", domain.ErrInvalidCollection)
		}
		if lesson.CreatedAt.IsZero() {
			lesson.CreatedAt = now
		}
		if lesson.UpdatedAt.IsZero() {
			lesson.UpdatedAt = now
		}
		lesson.ContentMarkdown = nil
		if err := lesson.Validate(); err != nil {
			return nil, err
		}
		normalizedLessons = append(normalizedLessons, lesson)
	}
	return normalizedLessons, nil
}

func (s *CourseGeneratorService) attachGeneratedContent(lesson domain.Lesson, output contract.LessonContentOutput) (domain.Lesson, error) {
	content := strings.TrimSpace(output.ContentMarkdown)
	if content == "" && output.Lesson.ContentMarkdown != nil {
		content = strings.TrimSpace(*output.Lesson.ContentMarkdown)
	}
	if content == "" {
		return domain.Lesson{}, ErrMissingGeneratedContent
	}
	if err := domain.ValidateLessonActivitiesForType(lesson.Type, output.Exercises, output.Quizzes); err != nil {
		return domain.Lesson{}, err
	}
	if err := lesson.AttachContent(content); err != nil {
		return domain.Lesson{}, err
	}
	lesson.RawContentOutput = jsonutil.Clone(output.Raw)
	lesson.Exercises = attachRawOutputToExercises(output.Exercises, output.Raw)
	lesson.Quizzes = attachRawOutputToQuizzes(output.Quizzes, output.Raw)
	lesson.UpdatedAt = s.now()
	return lesson, nil
}

func attachRawOutputToExercises(exercises []domain.Exercise, rawOutput json.RawMessage) []domain.Exercise {
	if len(exercises) == 0 {
		return nil
	}
	withRawOutput := make([]domain.Exercise, 0, len(exercises))
	for _, exercise := range exercises {
		exercise.RawAIOutput = jsonutil.Clone(rawOutput)
		withRawOutput = append(withRawOutput, exercise)
	}
	return withRawOutput
}

func attachRawOutputToQuizzes(quizzes []domain.Quiz, rawOutput json.RawMessage) []domain.Quiz {
	if len(quizzes) == 0 {
		return nil
	}
	withRawOutput := make([]domain.Quiz, 0, len(quizzes))
	for _, quiz := range quizzes {
		quiz.RawAIOutput = jsonutil.Clone(rawOutput)
		withRawOutput = append(withRawOutput, quiz)
	}
	return withRawOutput
}
