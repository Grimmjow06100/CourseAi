package dto

import (
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/pointer"
	"github.com/google/uuid"
)

type StartGenerationRequest struct {
	Prompt string `json:"prompt" binding:"required,max=4000"`
}

type AnalyzeGenerationRequest struct {
	Prompt string `json:"prompt" binding:"required,max=4000"`
}

type GenerateStructureRequest struct {
	Title        string   `json:"title" binding:"required,max=200"`
	Synopsis     string   `json:"synopsis" binding:"required,max=2000"`
	CurrentLevel string   `json:"currentLevel" binding:"required,oneof=beginner intermediate advanced expert unknown"`
	TargetLevel  string   `json:"targetLevel" binding:"required,oneof=beginner intermediate advanced expert unknown"`
	Goals        []string `json:"goals" binding:"required,min=1,max=10,dive,required,max=500"`
	Language     string   `json:"language" binding:"required,oneof=fr en"`
}

type SubmitClarificationsRequest struct {
	Answers  []ClarificationAnswerRequest `json:"answers" binding:"required,min=1,dive"`
	Title    string                       `json:"title" binding:"required,max=200"`
	Synopsis string                       `json:"synopsis" binding:"required,max=2000"`
	Language string                       `json:"language" binding:"required,oneof=fr en"`
}

type ClarificationAnswerRequest struct {
	QuestionID     string   `json:"questionId" binding:"required,oneof=goals currentLevel targetLevel"`
	SelectedValues []string `json:"selectedValues" binding:"required,min=1,max=4,dive,required,max=200"`
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
	RequestID              string                            `json:"requestId"`
	CourseID               *string                           `json:"courseId"`
	PipelineStatus         string                            `json:"pipelineStatus"`
	CourseStatus           *string                           `json:"courseStatus"`
	CurrentStep            *string                           `json:"currentStep"`
	ProgressPercent        int                               `json:"progressPercent"`
	FailureMessage         *string                           `json:"failureMessage"`
	IsOutOfScope           bool                              `json:"isOutOfScope"`
	ErrorMessage           *string                           `json:"errorMessage"`
	WarningMessage         *string                           `json:"warningMessage"`
	SuggestedTitle         *string                           `json:"suggestedTitle"`
	ShortSynopsis          *string                           `json:"shortSynopsis"`
	DetectedCurrentLevel   *string                           `json:"detectedCurrentLevel"`
	DetectedTargetLevel    *string                           `json:"detectedTargetLevel"`
	DetectedGoal           *string                           `json:"detectedGoal"`
	DetectedLanguage       *string                           `json:"detectedLanguage"`
	ClarificationQuestions []ClarificationQuestionResponse   `json:"clarificationQuestions"`
	ActionRequired         *GenerationActionRequiredResponse `json:"actionRequired"`
}

type GenerationActionRequiredResponse struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type GenerationAnalysisResponse struct {
	Request GenerationRequestResponse `json:"request"`
}

type GenerationResultResponse struct {
	Request GenerationRequestResponse `json:"request"`
	Course  CourseResponse            `json:"course"`
}

type GenerationRequestResponse struct {
	ID                        string                          `json:"id"`
	InitialUserPrompt         string                          `json:"initialUserPrompt"`
	PipelineStatus            string                          `json:"pipelineStatus"`
	CurrentStep               *string                         `json:"currentStep"`
	ProgressPercent           int                             `json:"progressPercent"`
	FailureMessage            *string                         `json:"failureMessage"`
	StartedAt                 *time.Time                      `json:"startedAt"`
	CompletedAt               *time.Time                      `json:"completedAt"`
	IsOutOfScope              bool                            `json:"isOutOfScope"`
	ErrorMessage              *string                         `json:"errorMessage"`
	WarningMessage            *string                         `json:"warningMessage"`
	SuggestedTitle            *string                         `json:"suggestedTitle"`
	ShortSynopsis             *string                         `json:"shortSynopsis"`
	DetectedCurrentLevel      *string                         `json:"detectedCurrentLevel"`
	DetectedTargetLevel       *string                         `json:"detectedTargetLevel"`
	DetectedGoal              *string                         `json:"detectedGoal"`
	DetectedLanguage          *string                         `json:"detectedLanguage"`
	ClarificationQuestions    []ClarificationQuestionResponse `json:"clarificationQuestions"`
	ClarificationAnswers      []ClarificationAnswerResponse   `json:"clarificationAnswers"`
	ConfirmedBrief            *GenerationBriefResponse        `json:"confirmedBrief"`
	AnalysisCompletedAt       *time.Time                      `json:"analysisCompletedAt"`
	BriefConfirmedAt          *time.Time                      `json:"briefConfirmedAt"`
	ClarificationsSubmittedAt *time.Time                      `json:"clarificationsSubmittedAt"`
	ClarificationVersion      int                             `json:"clarificationVersion"`
	CreatedAt                 time.Time                       `json:"createdAt"`
	UpdatedAt                 time.Time                       `json:"updatedAt"`
}

type ClarificationQuestionResponse struct {
	ID            string                        `json:"id"`
	Question      string                        `json:"question"`
	Options       []ClarificationOptionResponse `json:"options"`
	AllowMultiple bool                          `json:"allowMultiple"`
}

type ClarificationOptionResponse struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ClarificationAnswerResponse struct {
	QuestionID     string   `json:"questionId"`
	SelectedValues []string `json:"selectedValues"`
}

type GenerationBriefResponse struct {
	Title        string   `json:"title"`
	Synopsis     string   `json:"synopsis"`
	CurrentLevel string   `json:"currentLevel"`
	TargetLevel  string   `json:"targetLevel"`
	Goals        []string `json:"goals"`
	Language     string   `json:"language"`
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
		LastErrorMessage: publicGenerationFailure(job.LastErrorMessage),
		CreatedAt:        job.CreatedAt,
		UpdatedAt:        job.UpdatedAt,
	}
}

func GenerationStatusFromContract(status contract.GenerationStatus) GenerationStatusResponse {
	response := GenerationStatusResponse{
		RequestID:              status.RequestID.String(),
		CourseID:               pointer.Map(status.CourseID, uuid.UUID.String),
		PipelineStatus:         string(status.PipelineStatus),
		CourseStatus:           pointer.Map(status.CourseStatus, func(value domain.CourseGenerationStatus) string { return string(value) }),
		CurrentStep:            status.CurrentStep,
		ProgressPercent:        status.ProgressPercent,
		FailureMessage:         publicGenerationFailure(status.FailureMessage),
		IsOutOfScope:           status.IsOutOfScope,
		ErrorMessage:           status.ErrorMessage,
		WarningMessage:         status.WarningMessage,
		SuggestedTitle:         status.SuggestedTitle,
		ShortSynopsis:          status.ShortSynopsis,
		DetectedCurrentLevel:   pointer.Map(status.DetectedCurrentLevel, func(value domain.Level) string { return string(value) }),
		DetectedTargetLevel:    pointer.Map(status.DetectedTargetLevel, func(value domain.Level) string { return string(value) }),
		DetectedGoal:           status.DetectedGoal,
		DetectedLanguage:       pointer.Map(status.DetectedLanguage, func(value domain.CourseLanguage) string { return string(value) }),
		ClarificationQuestions: clarificationQuestionsFromDomain(status.ClarificationQuestions),
	}
	if status.ActionRequired != nil {
		response.ActionRequired = &GenerationActionRequiredResponse{Type: status.ActionRequired.Type, URL: status.ActionRequired.URL}
	}
	return response
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
	questions := clarificationQuestionsFromDomain(request.ClarificationQuestions)
	answers := make([]ClarificationAnswerResponse, 0, len(request.ClarificationAnswers))
	for _, answer := range request.ClarificationAnswers {
		answers = append(answers, ClarificationAnswerResponse{QuestionID: answer.QuestionID, SelectedValues: answer.SelectedValues})
	}

	return GenerationRequestResponse{
		ID:                        request.ID.String(),
		InitialUserPrompt:         request.InitialUserPrompt,
		PipelineStatus:            string(request.PipelineStatus),
		CurrentStep:               request.CurrentStep,
		ProgressPercent:           request.ProgressPercent,
		FailureMessage:            publicGenerationFailure(request.FailureMessage),
		StartedAt:                 request.StartedAt,
		CompletedAt:               request.CompletedAt,
		IsOutOfScope:              request.IsOutOfScope,
		ErrorMessage:              request.ErrorMessage,
		WarningMessage:            request.WarningMessage,
		SuggestedTitle:            request.SuggestedTitle,
		ShortSynopsis:             request.ShortSynopsis,
		DetectedCurrentLevel:      pointer.Map(request.DetectedCurrentLevel, func(value domain.Level) string { return string(value) }),
		DetectedTargetLevel:       pointer.Map(request.DetectedTargetLevel, func(value domain.Level) string { return string(value) }),
		DetectedGoal:              request.DetectedGoal,
		DetectedLanguage:          pointer.Map(request.DetectedLanguage, func(value domain.CourseLanguage) string { return string(value) }),
		ClarificationQuestions:    questions,
		ClarificationAnswers:      answers,
		ConfirmedBrief:            generationBriefFromDomain(request.ConfirmedBrief),
		AnalysisCompletedAt:       request.AnalysisCompletedAt,
		BriefConfirmedAt:          request.BriefConfirmedAt,
		ClarificationsSubmittedAt: request.ClarificationsSubmittedAt,
		ClarificationVersion:      request.ClarificationVersion,
		CreatedAt:                 request.CreatedAt,
		UpdatedAt:                 request.UpdatedAt,
	}
}

func publicGenerationFailure(message *string) *string {
	if message == nil {
		return nil
	}
	publicMessage := "generation failed; retry the operation or contact support with the request id"
	return &publicMessage
}

func clarificationQuestionsFromDomain(questions []domain.ClarificationQuestion) []ClarificationQuestionResponse {
	responses := make([]ClarificationQuestionResponse, 0, len(questions))
	for _, question := range questions {
		options := make([]ClarificationOptionResponse, 0, len(question.Options))
		for _, option := range question.Options {
			options = append(options, ClarificationOptionResponse{Value: option.Value, Label: option.Label})
		}
		responses = append(responses, ClarificationQuestionResponse{
			ID:            question.ID,
			Question:      question.Question,
			Options:       options,
			AllowMultiple: question.AllowMultiple,
		})
	}
	return responses
}

func generationBriefFromDomain(brief *domain.GenerationBrief) *GenerationBriefResponse {
	if brief == nil {
		return nil
	}
	return &GenerationBriefResponse{
		Title:        brief.Title,
		Synopsis:     brief.Synopsis,
		CurrentLevel: string(brief.CurrentLevel),
		TargetLevel:  string(brief.TargetLevel),
		Goals:        brief.Goals,
		Language:     string(brief.Language),
	}
}
