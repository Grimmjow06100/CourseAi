package postgres

import (
	"context"
	"encoding/json"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type GenerationRequestRepository struct {
	queries *dbsqlc.Queries
}

func NewGenerationRequestRepository(db DBTX) *GenerationRequestRepository {
	return &GenerationRequestRepository{queries: dbsqlc.New(db)}
}

func (r *GenerationRequestRepository) SaveGenerationRequest(ctx context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	if err := request.Validate(); err != nil {
		return domain.GenerationRequest{}, err
	}

	params, err := createGenerationRequestParams(request)
	if err != nil {
		return domain.GenerationRequest{}, err
	}
	row, err := r.queries.CreateGenerationRequest(ctx, params)
	if err != nil {
		return domain.GenerationRequest{}, err
	}
	return generationRequestFromSQLC(row)
}

func (r *GenerationRequestRepository) UpdateGenerationRequest(ctx context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	if err := request.Validate(); err != nil {
		return domain.GenerationRequest{}, err
	}

	params, err := updateGenerationRequestParams(request)
	if err != nil {
		return domain.GenerationRequest{}, err
	}
	row, err := r.queries.UpdateGenerationRequest(ctx, params)
	if err != nil {
		return domain.GenerationRequest{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}
	return generationRequestFromSQLC(row)
}

func (r *GenerationRequestRepository) FindGenerationRequestByID(ctx context.Context, id uuid.UUID) (domain.GenerationRequest, error) {
	row, err := r.queries.GetGenerationRequestByID(ctx, dbsqlc.GetGenerationRequestByIDParams{ID: id})
	if err != nil {
		return domain.GenerationRequest{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}
	return generationRequestFromSQLC(row)
}

func (r *GenerationRequestRepository) FindGenerationRequestByCourseID(ctx context.Context, courseID uuid.UUID) (domain.GenerationRequest, error) {
	row, err := r.queries.GetGenerationRequestByCourseID(ctx, dbsqlc.GetGenerationRequestByCourseIDParams{CourseID: courseID})
	if err != nil {
		return domain.GenerationRequest{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}
	return generationRequestFromSQLC(row)
}

func (r *GenerationRequestRepository) FindGenerationStatusByID(ctx context.Context, id uuid.UUID) (contract.GenerationStatus, error) {
	row, err := r.queries.GetGenerationStatusByID(ctx, dbsqlc.GetGenerationStatusByIDParams{ID: id})
	if err != nil {
		return contract.GenerationStatus{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}

	status := contract.GenerationStatus{
		RequestID:       row.RequestID,
		PipelineStatus:  domain.GenerationPipelineStatus(row.PipelineStatus),
		CurrentStep:     row.CurrentStep,
		ProgressPercent: int(row.ProgressPercent),
		FailureMessage:  row.FailureMessage,
	}
	if row.CourseID.Valid {
		courseID := uuid.UUID(row.CourseID.Bytes)
		status.CourseID = &courseID
	}
	if row.CourseStatus != nil {
		courseStatus := domain.CourseGenerationStatus(*row.CourseStatus)
		status.CourseStatus = &courseStatus
	}
	return status, nil
}

func createGenerationRequestParams(request domain.GenerationRequest) (dbsqlc.CreateGenerationRequestParams, error) {
	questions, err := clarificationQuestionsJSON(request.ClarificationQuestions)
	if err != nil {
		return dbsqlc.CreateGenerationRequestParams{}, err
	}
	return dbsqlc.CreateGenerationRequestParams{
		ID:                     request.ID,
		InitialUserPrompt:      request.InitialUserPrompt,
		PipelineStatus:         dbsqlc.GenerationPipelineStatus(request.PipelineStatus),
		CurrentStep:            request.CurrentStep,
		ProgressPercent:        int32(request.ProgressPercent),
		FailureMessage:         request.FailureMessage,
		StartedAt:              request.StartedAt,
		CompletedAt:            request.CompletedAt,
		IsOutOfScope:           request.IsOutOfScope,
		ErrorMessage:           request.ErrorMessage,
		WarningMessage:         request.WarningMessage,
		SuggestedTitle:         request.SuggestedTitle,
		ShortSynopsis:          request.ShortSynopsis,
		DetectedCurrentLevel:   sqlcLevelPtr(request.DetectedCurrentLevel),
		DetectedTargetLevel:    sqlcLevelPtr(request.DetectedTargetLevel),
		DetectedGoal:           request.DetectedGoal,
		DetectedLanguage:       sqlcLanguagePtr(request.DetectedLanguage),
		ClarificationQuestions: json.RawMessage(questions),
		RawAnalysisOutput:      rawJSONFromBytes(request.RawAnalysisOutput),
		CreatedAt:              request.CreatedAt,
		UpdatedAt:              request.UpdatedAt,
	}, nil
}

func updateGenerationRequestParams(request domain.GenerationRequest) (dbsqlc.UpdateGenerationRequestParams, error) {
	questions, err := clarificationQuestionsJSON(request.ClarificationQuestions)
	if err != nil {
		return dbsqlc.UpdateGenerationRequestParams{}, err
	}
	return dbsqlc.UpdateGenerationRequestParams{
		InitialUserPrompt:      request.InitialUserPrompt,
		PipelineStatus:         dbsqlc.GenerationPipelineStatus(request.PipelineStatus),
		CurrentStep:            request.CurrentStep,
		ProgressPercent:        int32(request.ProgressPercent),
		FailureMessage:         request.FailureMessage,
		StartedAt:              request.StartedAt,
		CompletedAt:            request.CompletedAt,
		IsOutOfScope:           request.IsOutOfScope,
		ErrorMessage:           request.ErrorMessage,
		WarningMessage:         request.WarningMessage,
		SuggestedTitle:         request.SuggestedTitle,
		ShortSynopsis:          request.ShortSynopsis,
		DetectedCurrentLevel:   sqlcLevelPtr(request.DetectedCurrentLevel),
		DetectedTargetLevel:    sqlcLevelPtr(request.DetectedTargetLevel),
		DetectedGoal:           request.DetectedGoal,
		DetectedLanguage:       sqlcLanguagePtr(request.DetectedLanguage),
		ClarificationQuestions: json.RawMessage(questions),
		RawAnalysisOutput:      rawJSONFromBytes(request.RawAnalysisOutput),
		UpdatedAt:              request.UpdatedAt,
		ID:                     request.ID,
	}, nil
}
