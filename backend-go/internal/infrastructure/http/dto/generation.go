package dto

import (
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/pointer"
	"github.com/google/uuid"
)

type StartGenerationRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

type AnalyzeGenerationRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

type GenerateStructureRequest struct {
	Title        string   `json:"title" binding:"required"`
	Synopsis     string   `json:"synopsis" binding:"required"`
	CurrentLevel string   `json:"currentLevel" binding:"required"`
	TargetLevel  string   `json:"targetLevel" binding:"required"`
	Goals        []string `json:"goals" binding:"required"`
	Language     string   `json:"language" binding:"required"`
}

type GenerationStartedResponse struct {
	JobID        string `json:"jobId"`
	RequestID    string `json:"requestId"`
	Status       string `json:"status"`
	JobStatus    string `json:"jobStatus"`
	StatusURL    string `json:"statusUrl"`
	JobStatusURL string `json:"jobStatusUrl"`
	ResultURL    string `json:"resultUrl"`
}

type GenerationJobResponse struct {
	ID               string     `json:"id"`
	RequestID        string     `json:"requestId"`
	ParentJobID      *string    `json:"parentJobId"`
	Kind             string     `json:"kind"`
	Status           string     `json:"status"`
	TargetID         *string    `json:"targetId"`
	Priority         int        `json:"priority"`
	AttemptCount     int        `json:"attemptCount"`
	MaxAttempts      int        `json:"maxAttempts"`
	AvailableAt      time.Time  `json:"availableAt"`
	LockedUntil      *time.Time `json:"lockedUntil"`
	StartedAt        *time.Time `json:"startedAt"`
	CompletedAt      *time.Time `json:"completedAt"`
	LastErrorCode    *string    `json:"lastErrorCode"`
	LastErrorMessage *string    `json:"lastErrorMessage"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type GenerationStatusResponse struct {
	RequestID       string  `json:"requestId"`
	CourseID        *string `json:"courseId"`
	PipelineStatus  string  `json:"pipelineStatus"`
	CourseStatus    *string `json:"courseStatus"`
	CurrentStep     *string `json:"currentStep"`
	ProgressPercent int     `json:"progressPercent"`
	FailureMessage  *string `json:"failureMessage"`
}

type GenerationAnalysisResponse struct {
	Request GenerationRequestResponse `json:"request"`
}

type GenerationResultResponse struct {
	Request GenerationRequestResponse `json:"request"`
	Course  CourseResponse            `json:"course"`
}

type GenerationRequestResponse struct {
	ID                     string                          `json:"id"`
	InitialUserPrompt      string                          `json:"initialUserPrompt"`
	PipelineStatus         string                          `json:"pipelineStatus"`
	CurrentStep            *string                         `json:"currentStep"`
	ProgressPercent        int                             `json:"progressPercent"`
	FailureMessage         *string                         `json:"failureMessage"`
	StartedAt              *time.Time                      `json:"startedAt"`
	CompletedAt            *time.Time                      `json:"completedAt"`
	IsOutOfScope           bool                            `json:"isOutOfScope"`
	ErrorMessage           *string                         `json:"errorMessage"`
	WarningMessage         *string                         `json:"warningMessage"`
	SuggestedTitle         *string                         `json:"suggestedTitle"`
	ShortSynopsis          *string                         `json:"shortSynopsis"`
	DetectedCurrentLevel   *string                         `json:"detectedCurrentLevel"`
	DetectedTargetLevel    *string                         `json:"detectedTargetLevel"`
	DetectedGoal           *string                         `json:"detectedGoal"`
	DetectedLanguage       *string                         `json:"detectedLanguage"`
	ClarificationQuestions []ClarificationQuestionResponse `json:"clarificationQuestions"`
	CreatedAt              time.Time                       `json:"createdAt"`
	UpdatedAt              time.Time                       `json:"updatedAt"`
}

type ClarificationQuestionResponse struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

func GenerationStartedFromContract(started contract.GenerationStarted) GenerationStartedResponse {
	return GenerationStartedResponse{
		JobID:        started.JobID.String(),
		RequestID:    started.RequestID.String(),
		Status:       string(started.Status),
		JobStatus:    string(started.JobStatus),
		StatusURL:    started.StatusURL,
		JobStatusURL: started.JobStatusURL,
		ResultURL:    started.ResultURL,
	}
}

func GenerationJobFromDomain(job domain.GenerationJob) GenerationJobResponse {
	return GenerationJobResponse{
		ID:               job.ID.String(),
		RequestID:        job.RequestID.String(),
		ParentJobID:      pointer.Map(job.ParentJobID, uuid.UUID.String),
		Kind:             string(job.Kind),
		Status:           string(job.Status),
		TargetID:         pointer.Map(job.TargetID, uuid.UUID.String),
		Priority:         job.Priority,
		AttemptCount:     job.AttemptCount,
		MaxAttempts:      job.MaxAttempts,
		AvailableAt:      job.AvailableAt,
		LockedUntil:      job.LockedUntil,
		StartedAt:        job.StartedAt,
		CompletedAt:      job.CompletedAt,
		LastErrorCode:    job.LastErrorCode,
		LastErrorMessage: job.LastErrorMessage,
		CreatedAt:        job.CreatedAt,
		UpdatedAt:        job.UpdatedAt,
	}
}

func GenerationStatusFromContract(status contract.GenerationStatus) GenerationStatusResponse {
	return GenerationStatusResponse{
		RequestID:       status.RequestID.String(),
		CourseID:        pointer.Map(status.CourseID, uuid.UUID.String),
		PipelineStatus:  string(status.PipelineStatus),
		CourseStatus:    pointer.Map(status.CourseStatus, func(value domain.CourseGenerationStatus) string { return string(value) }),
		CurrentStep:     status.CurrentStep,
		ProgressPercent: status.ProgressPercent,
		FailureMessage:  status.FailureMessage,
	}
}

func GenerationAnalysisFromContract(result contract.GenerationAnalysisResult) GenerationAnalysisResponse {
	return GenerationAnalysisResponse{Request: GenerationRequestFromDomain(result.Request)}
}

func GenerationResultFromContract(result contract.GenerationResult) GenerationResultResponse {
	return GenerationResultResponse{
		Request: GenerationRequestFromDomain(result.Request),
		Course:  CourseFromDomain(result.Course),
	}
}

func GenerationRequestFromDomain(request domain.GenerationRequest) GenerationRequestResponse {
	questions := make([]ClarificationQuestionResponse, 0, len(request.ClarificationQuestions))
	for _, question := range request.ClarificationQuestions {
		questions = append(questions, ClarificationQuestionResponse{
			ID:       question.ID,
			Question: question.Question,
			Options:  question.Options,
		})
	}

	return GenerationRequestResponse{
		ID:                     request.ID.String(),
		InitialUserPrompt:      request.InitialUserPrompt,
		PipelineStatus:         string(request.PipelineStatus),
		CurrentStep:            request.CurrentStep,
		ProgressPercent:        request.ProgressPercent,
		FailureMessage:         request.FailureMessage,
		StartedAt:              request.StartedAt,
		CompletedAt:            request.CompletedAt,
		IsOutOfScope:           request.IsOutOfScope,
		ErrorMessage:           request.ErrorMessage,
		WarningMessage:         request.WarningMessage,
		SuggestedTitle:         request.SuggestedTitle,
		ShortSynopsis:          request.ShortSynopsis,
		DetectedCurrentLevel:   pointer.Map(request.DetectedCurrentLevel, func(value domain.Level) string { return string(value) }),
		DetectedTargetLevel:    pointer.Map(request.DetectedTargetLevel, func(value domain.Level) string { return string(value) }),
		DetectedGoal:           request.DetectedGoal,
		DetectedLanguage:       pointer.Map(request.DetectedLanguage, func(value domain.CourseLanguage) string { return string(value) }),
		ClarificationQuestions: questions,
		CreatedAt:              request.CreatedAt,
		UpdatedAt:              request.UpdatedAt,
	}
}
