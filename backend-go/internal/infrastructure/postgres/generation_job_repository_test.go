package postgres

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestSameGenerationJobOperationUsesSemanticJSONEquality(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	targetID := uuid.New()
	existing := domain.GenerationJob{
		RequestID: requestID,
		Kind:      domain.GenerationJobKindLessonContent,
		TargetID:  &targetID,
		Payload:   json.RawMessage(`{"lessonId":"one","force":true}`),
	}
	candidate := existing
	candidate.Payload = json.RawMessage(`{ "force": true, "lessonId": "one" }`)

	if !sameGenerationJobOperation(existing, candidate) {
		t.Fatal("equivalent JSON objects should describe the same operation")
	}

	candidate.Payload = json.RawMessage(`{"lessonId":"two","force":true}`)
	if sameGenerationJobOperation(existing, candidate) {
		t.Fatal("different payloads should cause an idempotency conflict")
	}
}

func TestGenerationJobFailureUsesTypedErrorCode(t *testing.T) {
	t.Parallel()

	cause := codedJobError{code: "openai_rate_limit", message: "rate limited"}
	code, message, err := generationJobFailure(cause)
	if err != nil {
		t.Fatalf("generation job failure: %v", err)
	}
	if code != cause.code || message != cause.message {
		t.Fatalf("failure = (%q, %q), want (%q, %q)", code, message, cause.code, cause.message)
	}
}

func TestRequireGenerationJobClaimRejectsStaleClaim(t *testing.T) {
	t.Parallel()

	if err := requireGenerationJobClaim(1, nil); err != nil {
		t.Fatalf("valid claim: %v", err)
	}
	if err := requireGenerationJobClaim(0, nil); !errors.Is(err, contract.ErrGenerationJobClaimLost) {
		t.Fatalf("error = %v, want ErrGenerationJobClaimLost", err)
	}
}

func TestValidateQueuedGenerationJobRejectsStartedJob(t *testing.T) {
	t.Parallel()

	now := time.Now()
	job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
		RequestID:      uuid.New(),
		Kind:           domain.GenerationJobKindAnalysis,
		IdempotencyKey: "analysis",
	}, now)
	if err != nil {
		t.Fatalf("new generation job: %v", err)
	}
	job.StartedAt = &now

	if err := validateQueuedGenerationJob(job); !errors.Is(err, domain.ErrInvalidGenerationJobState) {
		t.Fatalf("error = %v, want ErrInvalidGenerationJobState", err)
	}
}

type codedJobError struct {
	code    string
	message string
}

func (e codedJobError) Error() string {
	return e.message
}

func (e codedJobError) ErrorCode() string {
	return e.code
}
