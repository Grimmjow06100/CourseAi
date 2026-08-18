package postgres

import "github.com/Grimmjow06100/course-ai/backend-go/internal/contract"

var _ contract.UserRepository = (*UserRepository)(nil)
var _ contract.GenerationRequestRepository = (*GenerationRequestRepository)(nil)
var _ contract.GenerationJobQueue = (*GenerationJobRepository)(nil)
var _ contract.CourseRepository = (*CourseRepository)(nil)
var _ contract.ModuleRepository = (*ModuleRepository)(nil)
var _ contract.LessonRepository = (*LessonRepository)(nil)
var _ contract.ExerciseRepository = (*ExerciseRepository)(nil)
var _ contract.QuizRepository = (*QuizRepository)(nil)
var _ contract.TransactionalRepositories = (*Repositories)(nil)
var _ contract.UnitOfWork = (*UnitOfWork)(nil)
