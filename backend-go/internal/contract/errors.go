package contract

import "errors"

var (
	ErrCourseNotFound            = errors.New("course not found")
	ErrGenerationRequestNotFound = errors.New("generation request not found")
	ErrLessonNotFound            = errors.New("lesson not found")
	ErrModuleNotFound            = errors.New("module not found")
)
