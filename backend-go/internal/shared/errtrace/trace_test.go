package errtrace

import (
	"errors"
	"strings"
	"testing"
)

func TestCapturePreservesCauseAndStack(t *testing.T) {
	cause := errors.New("boom")
	err := Capture(cause)
	if !errors.Is(err, cause) {
		t.Fatal("captured error must preserve its cause")
	}
	if stack := Stack(err); !strings.Contains(stack, "TestCapturePreservesCauseAndStack") {
		t.Fatalf("stack does not contain capture call site: %s", stack)
	}
	if Capture(err) != err {
		t.Fatal("capture must not replace an existing trace")
	}
}
