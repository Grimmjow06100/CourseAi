package jobs

import (
	"errors"
	"testing"
)

func TestPermanentJobErrorMetadata(t *testing.T) {
	t.Parallel()

	cause := errors.New("invalid command")
	err := permanentJobError{code: "invalid_command", err: cause}
	if err.Error() != cause.Error() || err.ErrorCode() != "invalid_command" || err.Retryable() || !errors.Is(err, cause) {
		t.Fatalf("unexpected permanent job error behavior: %+v", err)
	}
}
