package dto

import (
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestGenerationDTOConversions(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	jobID := uuid.New()
	parentID := uuid.New()
	targetID := uuid.New()
	courseID := uuid.New()
	courseStatus := domain.CourseStatusLessonsGenerated
	level := domain.LevelIntermediate
	language := domain.CourseLanguageFR
	title := "Linux administration"
	synopsis := "Learn Linux progressively"
	goal := "Administer Linux"
	now := time.Unix(100, 0).UTC()

	started := GenerationStartedFromContract(contract.GenerationStarted{
		JobID: jobID, RequestID: requestID, Status: domain.PipelineStatusQueued,
		JobStatus: domain.GenerationJobStatusQueued, StatusURL: "/status", JobStatusURL: "/job", ResultURL: "/result",
	})
	if started.JobID != jobID.String() || started.RequestID != requestID.String() || started.JobStatusURL != "/job" {
		t.Fatalf("unexpected started response: %+v", started)
	}

	job := GenerationJobFromDomain(domain.GenerationJob{
		ID: jobID, RequestID: requestID, ParentJobID: &parentID, TargetID: &targetID,
		Kind: domain.GenerationJobKindLessonContent, Status: domain.GenerationJobStatusRunning, CreatedAt: now,
	})
	if job.ParentJobID == nil || *job.ParentJobID != parentID.String() || job.TargetID == nil || *job.TargetID != targetID.String() {
		t.Fatalf("unexpected job response: %+v", job)
	}

	status := GenerationStatusFromContract(contract.GenerationStatus{
		RequestID: requestID, CourseID: &courseID, PipelineStatus: domain.PipelineStatusRunning,
		CourseStatus: &courseStatus, ProgressPercent: 60, SuggestedTitle: &title, ShortSynopsis: &synopsis,
		DetectedCurrentLevel: &level, DetectedGoal: &goal, DetectedLanguage: &language,
	})
	if status.CourseID == nil || *status.CourseID != courseID.String() || status.CourseStatus == nil || *status.CourseStatus != string(courseStatus) ||
		status.SuggestedTitle == nil || *status.SuggestedTitle != title || status.DetectedCurrentLevel == nil || *status.DetectedCurrentLevel != "intermediate" ||
		status.DetectedLanguage == nil || *status.DetectedLanguage != "fr" {
		t.Fatalf("unexpected status response: %+v", status)
	}

	request := domain.GenerationRequest{
		ID: requestID, PipelineStatus: domain.PipelineStatusRunning,
		DetectedCurrentLevel: &level, DetectedLanguage: &language,
		ClarificationQuestions: []domain.ClarificationQuestion{{
			ID: "goals", Question: "Goal?", AllowMultiple: true,
			Options: []domain.ClarificationOption{{Value: "Learn", Label: "Learn"}, {Value: "Build", Label: "Build"}},
		}},
	}
	requestResponse := GenerationRequestFromDomain(request)
	if requestResponse.DetectedCurrentLevel == nil || *requestResponse.DetectedCurrentLevel != "intermediate" ||
		requestResponse.DetectedLanguage == nil || *requestResponse.DetectedLanguage != "fr" || len(requestResponse.ClarificationQuestions) != 1 {
		t.Fatalf("unexpected request response: %+v", requestResponse)
	}
}

func TestGenerationResultWrapper(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	request := domain.GenerationRequest{ID: requestID}
	course := domain.Course{ID: uuid.New(), RequestID: requestID}
	result := GenerationResultFromContract(contract.GenerationResult{Request: request, Course: course})
	if result.Request.ID != requestID.String() || result.Course.ID != course.ID.String() {
		t.Fatalf("unexpected wrapper response: %+v", result)
	}
}
