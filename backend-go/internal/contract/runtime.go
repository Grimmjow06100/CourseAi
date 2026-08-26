package contract

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
}

type TransactionalRepositories interface {
	Ownership() OwnershipRepository
	GenerationRequests() GenerationRequestRepository
	GenerationJobs() GenerationJobQueue
	Courses() CourseRepository
	Modules() ModuleRepository
	Lessons() LessonRepository
	Exercises() ExerciseRepository
	Quizzes() QuizRepository
}

type UnitOfWork interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, repositories TransactionalRepositories) error) error
}
