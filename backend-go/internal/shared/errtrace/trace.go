package errtrace

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

const maxFrames = 64

type tracedError struct {
	err error
	pcs []uintptr
}

func (e *tracedError) Error() string         { return e.err.Error() }
func (e *tracedError) Unwrap() error         { return e.err }
func (e *tracedError) stackTrace() []uintptr { return e.pcs }

type stackTracer interface {
	stackTrace() []uintptr
}

// Capture attaches a call stack once and preserves any stack already present in the error chain.
func Capture(err error) error {
	if err == nil {
		return nil
	}
	var existing stackTracer
	if errors.As(err, &existing) {
		return err
	}
	pcs := make([]uintptr, maxFrames)
	count := runtime.Callers(2, pcs)
	return &tracedError{err: err, pcs: pcs[:count]}
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return Capture(fmt.Errorf("%s: %w", strings.TrimSpace(message), err))
}

func Stack(err error) string {
	var traced stackTracer
	if !errors.As(err, &traced) {
		return ""
	}
	frames := runtime.CallersFrames(traced.stackTrace())
	var builder strings.Builder
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&builder, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return strings.TrimSpace(builder.String())
}
