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

	courseIDs := make([]uuid.UUID, 0, len(courses))
	for _, course := range courses {
		courseIDs = append(courseIDs, course.ID)
	}
	rows, err := l.queries.ListModulesByCourseIDs(ctx, dbsqlc.ListModulesByCourseIDsParams{CourseIds: courseIDs})
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

	modulesByCourseID := make(map[uuid.UUID][]domain.Module, len(courses))
	for _, module := range modules {
		modulesByCourseID[module.CourseID] = append(modulesByCourseID[module.CourseID], module)
	}
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

	moduleIDs := make([]uuid.UUID, 0, len(modules))
	for _, module := range modules {
		moduleIDs = append(moduleIDs, module.ID)
	}
	rows, err := l.queries.ListLessonsByModuleIDs(ctx, dbsqlc.ListLessonsByModuleIDsParams{ModuleIds: moduleIDs})
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

	lessonsByModuleID := make(map[uuid.UUID][]domain.Lesson, len(modules))
	for _, lesson := range lessons {
		lessonsByModuleID[lesson.ModuleID] = append(lessonsByModuleID[lesson.ModuleID], lesson)
	}
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

	lessonIDs := make([]uuid.UUID, 0, len(lessons))
	for _, lesson := range lessons {
		lessonIDs = append(lessonIDs, lesson.ID)
	}
	exerciseRows, err := l.queries.ListLessonExercisesByLessonIDs(ctx, dbsqlc.ListLessonExercisesByLessonIDsParams{LessonIds: lessonIDs})
	if err != nil {
		return nil, err
	}
	quizRows, err := l.queries.ListLessonQuizzesByLessonIDs(ctx, dbsqlc.ListLessonQuizzesByLessonIDsParams{LessonIds: lessonIDs})
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

	exercisesByLessonID := make(map[uuid.UUID][]domain.Exercise, len(lessons))
	for _, exercise := range exercises {
		exercisesByLessonID[exercise.LessonID] = append(exercisesByLessonID[exercise.LessonID], exercise)
	}
	quizzesByLessonID := make(map[uuid.UUID][]domain.Quiz, len(lessons))
	for _, quiz := range quizzes {
		quizzesByLessonID[quiz.LessonID] = append(quizzesByLessonID[quiz.LessonID], quiz)
	}
	for index := range lessons {
		lessons[index].Exercises = exercisesByLessonID[lessons[index].ID]
		lessons[index].Quizzes = quizzesByLessonID[lessons[index].ID]
		if err := lessons[index].Validate(); err != nil {
			return nil, err
		}
	}
	return lessons, nil
}
