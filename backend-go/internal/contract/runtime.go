package contract

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
}

type TransactionalRepositories interface {
	Users() UserRepository
	GenerationRequests() GenerationRequestRepository
	Courses() CourseRepository
	Modules() ModuleRepository
	Lessons() LessonRepository
}

type UnitOfWork interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, repositories TransactionalRepositories) error) error
}
