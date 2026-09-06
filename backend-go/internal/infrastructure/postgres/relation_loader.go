package postgres

import (
	"context"

	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type relationLoader struct {
	queries *dbsqlc.Queries
}

func (l relationLoader) hydrateCourses(ctx context.Context, courses []domain.Course) ([]domain.Course, error) {
	if len(courses) == 0 {
		return courses, nil
	}

	courseIDs := collectUUIDs(courses, func(course domain.Course) uuid.UUID { return course.ID })
	rows, err := l.queries.ListModulesByCourseIDs(ctx, courseIDs)
	if err != nil {
		return nil, err
	}
	modules, err := modulesFromSQLC(rows)
	if err != nil {
		return nil, err
	}
	modules, err = l.hydrateModules(ctx, modules)
	if err != nil {
		return nil, err
	}

	modulesByCourseID := groupByUUID(modules, func(module domain.Module) uuid.UUID { return module.CourseID })
	for index := range courses {
		courses[index].Modules = modulesByCourseID[courses[index].ID]
		if err := courses[index].Validate(); err != nil {
			return nil, err
		}
	}
	return courses, nil
}

func (l relationLoader) hydrateModules(ctx context.Context, modules []domain.Module) ([]domain.Module, error) {
	if len(modules) == 0 {
		return modules, nil
	}

	moduleIDs := collectUUIDs(modules, func(module domain.Module) uuid.UUID { return module.ID })
	rows, err := l.queries.ListLessonsByModuleIDs(ctx, moduleIDs)
	if err != nil {
		return nil, err
	}
	lessons, err := lessonsFromSQLC(rows)
	if err != nil {
		return nil, err
	}
	lessons, err = l.hydrateLessons(ctx, lessons)
	if err != nil {
		return nil, err
	}

	lessonsByModuleID := groupByUUID(lessons, func(lesson domain.Lesson) uuid.UUID { return lesson.ModuleID })
	for index := range modules {
		modules[index].Lessons = lessonsByModuleID[modules[index].ID]
		if err := modules[index].Validate(); err != nil {
			return nil, err
		}
	}
	return modules, nil
}

func (l relationLoader) hydrateLessons(ctx context.Context, lessons []domain.Lesson) ([]domain.Lesson, error) {
	if len(lessons) == 0 {
		return lessons, nil
	}

	lessonIDs := collectUUIDs(lessons, func(lesson domain.Lesson) uuid.UUID { return lesson.ID })
	exerciseRows, err := l.queries.ListLessonExercisesByLessonIDs(ctx, lessonIDs)
	if err != nil {
		return nil, err
	}
	quizRows, err := l.queries.ListLessonQuizzesByLessonIDs(ctx, lessonIDs)
	if err != nil {
		return nil, err
	}
	exercises, err := exercisesFromSQLC(exerciseRows)
	if err != nil {
		return nil, err
	}
	quizzes, err := quizzesFromSQLC(quizRows)
	if err != nil {
		return nil, err
	}

	exercisesByLessonID := groupByUUID(exercises, func(exercise domain.Exercise) uuid.UUID { return exercise.LessonID })
	quizzesByLessonID := groupByUUID(quizzes, func(quiz domain.Quiz) uuid.UUID { return quiz.LessonID })
	for index := range lessons {
		lessons[index].Exercises = exercisesByLessonID[lessons[index].ID]
		lessons[index].Quizzes = quizzesByLessonID[lessons[index].ID]
		if err := lessons[index].Validate(); err != nil {
			return nil, err
		}
	}
	return lessons, nil
}

func collectUUIDs[Entity any](entities []Entity, id func(Entity) uuid.UUID) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(entities))
	for _, entity := range entities {
		ids = append(ids, id(entity))
	}
	return ids
}

func groupByUUID[Entity any](entities []Entity, parentID func(Entity) uuid.UUID) map[uuid.UUID][]Entity {
	grouped := make(map[uuid.UUID][]Entity)
	for _, entity := range entities {
		key := parentID(entity)
		grouped[key] = append(grouped[key], entity)
	}
	return grouped
}
