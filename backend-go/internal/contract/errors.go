package contract

import "errors"

var (
	ErrCourseNotFound                   = errors.New("course not found")
	ErrGenerationRequestNotFound        = errors.New("generation request not found")
	ErrLessonNotFound                   = errors.New("lesson not found")
	ErrModuleNotFound                   = errors.New("module not found")
	ErrGenerationJobNotFound            = errors.New("generation job not found")
	ErrGenerationJobUnavailable         = errors.New("no generation job is available")
	ErrGenerationJobClaimLost           = errors.New("generation job claim is no longer valid")
	ErrGenerationJobNotCancellable      = errors.New("generation job cannot be cancelled in its current state")
	ErrGenerationJobIdempotencyConflict = errors.New("generation job idempotency key is already used by another operation")
)
