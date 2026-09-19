package postgres

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/errtrace"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repositories exposes PostgreSQL implementations behind the application contract.
type Repositories struct {
	tracking           *GenerationTrackingRepository
	ownership          *OwnershipRepository
	generationRequests *GenerationRequestRepository
	generationJobs     *GenerationJobRepository
	courses            *CourseRepository
	modules            *ModuleRepository
	lessons            *LessonRepository
	exercises          *ExerciseRepository
	quizzes            *QuizRepository
}

func NewRepositories(db DBTX) *Repositories {
	return &Repositories{
		tracking:           NewGenerationTrackingRepository(db),
		ownership:          NewOwnershipRepository(db),
		generationRequests: NewGenerationRequestRepository(db),
		generationJobs:     NewGenerationJobRepository(db),
		courses:            NewCourseRepository(db),
		modules:            NewModuleRepository(db),
		lessons:            NewLessonRepository(db),
		exercises:          NewExerciseRepository(db),
		quizzes:            NewQuizRepository(db),
	}
}

func (r *Repositories) Ownership() contract.OwnershipRepository {
	return r.ownership
}

func (r *Repositories) GenerationRequests() contract.GenerationRequestRepository {
	return r.generationRequests
}

func (r *Repositories) GenerationJobs() contract.GenerationJobQueue {
	return r.generationJobs
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

func (r *Repositories) Exercises() contract.ExerciseRepository {
	return r.exercises
}

func (r *Repositories) Quizzes() contract.QuizRepository {
	return r.quizzes
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
		return errtrace.Wrap(err, "begin PostgreSQL transaction")
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := fn(ctx, NewRepositories(tx)); err != nil {
		return errtrace.Capture(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return errtrace.Wrap(err, "commit PostgreSQL transaction")
	}
	committed = true
	return nil
}

func (r *Repositories) Tracking() contract.GenerationTrackingRepository { return r.tracking }
