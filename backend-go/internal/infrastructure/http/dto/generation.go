package dto

import (
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

type StartGenerationRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

type GenerationStartedResponse struct {
	RequestID string `json:"requestId"`
	Status    string `json:"status"`
	StatusURL string `json:"statusUrl"`
	ResultURL string `json:"resultUrl"`
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
		RequestID: started.RequestID.String(),
		Status:    string(started.Status),
		StatusURL: started.StatusURL,
		ResultURL: started.ResultURL,
	}
}

func GenerationStatusFromContract(status contract.GenerationStatus) GenerationStatusResponse {
	var courseID *string
	if status.CourseID != nil {
		value := status.CourseID.String()
		courseID = &value
	}

	var courseStatus *string
	if status.CourseStatus != nil {
		value := string(*status.CourseStatus)
		courseStatus = &value
	}

	return GenerationStatusResponse{
		RequestID:       status.RequestID.String(),
		CourseID:        courseID,
		PipelineStatus:  string(status.PipelineStatus),
		CourseStatus:    courseStatus,
		CurrentStep:     status.CurrentStep,
		ProgressPercent: status.ProgressPercent,
		FailureMessage:  status.FailureMessage,
	}
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
		DetectedCurrentLevel:   levelPtr(request.DetectedCurrentLevel),
		DetectedTargetLevel:    levelPtr(request.DetectedTargetLevel),
		DetectedGoal:           request.DetectedGoal,
		DetectedLanguage:       languagePtr(request.DetectedLanguage),
		ClarificationQuestions: questions,
		CreatedAt:              request.CreatedAt,
		UpdatedAt:              request.UpdatedAt,
	}
}

func levelPtr(level *domain.Level) *string {
	if level == nil {
		return nil
	}
	value := string(*level)
	return &value
}

func languagePtr(language *domain.CourseLanguage) *string {
	if language == nil {
		return nil
	}
	value := string(*language)
	return &value
}
