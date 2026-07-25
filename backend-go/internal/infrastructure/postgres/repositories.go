package postgres

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repositories exposes PostgreSQL implementations behind the application contract.
type Repositories struct {
	users              *UserRepository
	generationRequests *GenerationRequestRepository
	courses            *CourseRepository
	modules            *ModuleRepository
	lessons            *LessonRepository
}

func NewRepositories(db DBTX) *Repositories {
	return &Repositories{
		users:              NewUserRepository(db),
		generationRequests: NewGenerationRequestRepository(db),
		courses:            NewCourseRepository(db),
		modules:            NewModuleRepository(db),
		lessons:            NewLessonRepository(db),
	}
}

func (r *Repositories) Users() contract.UserRepository {
	return r.users
}

func (r *Repositories) GenerationRequests() contract.GenerationRequestRepository {
	return r.generationRequests
}

func (r *Repositories) Courses() contract.CourseRepository {
	return r.courses
}

func (r *Repositories) Modules() contract.ModuleRepository {
	return r.modules
}

func (r *Repositories) Lessons() contract.LessonRepository {
	return r.lessons
}

// UnitOfWork runs repository operations inside a single PostgreSQL transaction.
type UnitOfWork struct {
	pool *pgxpool.Pool
}

func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

func (u *UnitOfWork) WithinTx(ctx context.Context, fn func(ctx context.Context, repositories contract.TransactionalRepositories) error) error {
	tx, err := u.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := fn(ctx, NewRepositories(tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}
