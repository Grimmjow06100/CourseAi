package postgres

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type GenerationRequestRepository struct {
	queries *dbsqlc.Queries
}

func (r *GenerationRequestRepository) ListCompletionCandidates(ctx context.Context, limit int) ([]uuid.UUID, error) {
	return r.queries.ListGenerationCompletionCandidates(ctx, int32(limit))
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

func (r *GenerationRequestRepository) ListGenerationRequests(ctx context.Context, filters contract.GenerationHistoryFilters) (contract.Page[contract.GenerationSummary], error) {
	pagination := filters.Pagination.Normalize()
	var status *dbsqlc.GenerationPipelineStatus
	if filters.PipelineStatus != nil {
		value := dbsqlc.GenerationPipelineStatus(*filters.PipelineStatus)
		status = &value
	}

	totalItems, err := r.queries.CountGenerationRequestsByOwner(ctx, dbsqlc.CountGenerationRequestsByOwnerParams{
		ClerkUserID:    strings.TrimSpace(filters.ClerkUserID),
		PipelineStatus: status,
	})
	if err != nil {
		return contract.Page[contract.GenerationSummary]{}, err
	}

	rows, err := r.queries.ListGenerationRequestsByOwner(ctx, dbsqlc.ListGenerationRequestsByOwnerParams{
		ClerkUserID:    strings.TrimSpace(filters.ClerkUserID),
		PipelineStatus: status,
		OffsetRows:     int32((pagination.Page - 1) * pagination.PageSize),
		LimitRows:      int32(pagination.PageSize),
	})
	if err != nil {
		return contract.Page[contract.GenerationSummary]{}, err
	}

	items := make([]contract.GenerationSummary, 0, len(rows))
	for _, row := range rows {
		summary := contract.GenerationSummary{
			GenerationAttempt: int(row.GenerationAttempt),
			ContentComplete:   row.ContentComplete,
			RequestID:         row.RequestID,
			InitialUserPrompt: row.InitialUserPrompt,
			Title:             row.Title,
			PipelineStatus:    domain.GenerationPipelineStatus(row.PipelineStatus),
			CurrentStep:       row.CurrentStep,
			ProgressPercent:   int(row.ProgressPercent),
			IsOutOfScope:      row.IsOutOfScope,
			FailureMessage:    row.FailureMessage,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		}
		if row.CourseID.Valid {
			courseID := uuid.UUID(row.CourseID.Bytes)
			summary.CourseID = &courseID
		}
		if row.CourseStatus != nil {
			courseStatus := domain.CourseGenerationStatus(*row.CourseStatus)
			summary.CourseStatus = &courseStatus
		}
		items = append(items, summary)
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(pagination.PageSize)))
	}
	return contract.Page[contract.GenerationSummary]{
		Items:       items,
		Page:        pagination.Page,
		PageSize:    pagination.PageSize,
		TotalItems:  int(totalItems),
		TotalPages:  totalPages,
		HasNext:     pagination.Page < totalPages,
		HasPrevious: pagination.Page > 1,
	}, nil
}

func (r *GenerationRequestRepository) FindGenerationRequestByID(ctx context.Context, id uuid.UUID) (domain.GenerationRequest, error) {
	row, err := r.queries.GetGenerationRequestByID(ctx, id)
	if err != nil {
		return domain.GenerationRequest{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}
	return generationRequestFromSQLC(row)
}

func (r *GenerationRequestRepository) GetGenerationAdmissionUsage(ctx context.Context, clerkUserID string, createdSince time.Time) (contract.GenerationAdmissionUsage, error) {
	row, err := r.queries.GetGenerationAdmissionUsage(ctx, dbsqlc.GetGenerationAdmissionUsageParams{
		ClerkUserID:  strings.TrimSpace(clerkUserID),
		CreatedSince: createdSince,
	})
	if err != nil {
		return contract.GenerationAdmissionUsage{}, err
	}
	return contract.GenerationAdmissionUsage{
		ActiveRequests: row.ActiveRequests,
		DailyRequests:  row.DailyRequests,
		PendingJobs:    row.PendingJobs,
	}, nil
}

func (r *GenerationRequestRepository) DeleteGenerationRequest(ctx context.Context, id uuid.UUID) error {
	rows, err := r.queries.DeleteGenerationRequestByID(ctx, id)
	if err != nil {
		return err
	}
	if rows != 1 {
		return contract.ErrGenerationRequestNotFound
	}
	return nil
}

func (r *GenerationRequestRepository) FindGenerationRequestForUpdate(ctx context.Context, id uuid.UUID) (domain.GenerationRequest, error) {
	row, err := r.queries.GetGenerationRequestForUpdate(ctx, id)
	if err != nil {
		return domain.GenerationRequest{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}
	return generationRequestFromSQLC(row)
}

func (r *GenerationRequestRepository) FindGenerationRequestByCourseID(ctx context.Context, courseID uuid.UUID) (domain.GenerationRequest, error) {
	row, err := r.queries.GetGenerationRequestByCourseID(ctx, courseID)
	if err != nil {
		return domain.GenerationRequest{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}
	return generationRequestFromSQLC(row)
}

func (r *GenerationRequestRepository) FindGenerationStatusByID(ctx context.Context, id uuid.UUID) (contract.GenerationStatus, error) {
	row, err := r.queries.GetGenerationStatusByID(ctx, id)
	if err != nil {
		return contract.GenerationStatus{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}

	status := contract.GenerationStatus{
		GenerationAttempt: int(row.GenerationAttempt),
		ContentComplete:   row.ContentComplete,
		RequestID:         row.RequestID,
		PipelineStatus:    domain.GenerationPipelineStatus(row.PipelineStatus),
		CurrentStep:       row.CurrentStep,
		ProgressPercent:   int(row.ProgressPercent),
		FailureMessage:    row.FailureMessage,
		IsOutOfScope:      row.IsOutOfScope,
		ErrorMessage:      row.ErrorMessage,
		WarningMessage:    row.WarningMessage,
		SuggestedTitle:    row.SuggestedTitle,
		ShortSynopsis:     row.ShortSynopsis,
		DetectedGoal:      row.DetectedGoal,
	}
	if row.DetectedCurrentLevel != nil {
		level, parseErr := domain.ParseLevel(string(*row.DetectedCurrentLevel))
		if parseErr != nil {
			return contract.GenerationStatus{}, parseErr
		}
		status.DetectedCurrentLevel = &level
	}
	if row.DetectedTargetLevel != nil {
		level, parseErr := domain.ParseLevel(string(*row.DetectedTargetLevel))
		if parseErr != nil {
			return contract.GenerationStatus{}, parseErr
		}
		status.DetectedTargetLevel = &level
	}
	if row.DetectedLanguage != nil {
		language, parseErr := domain.ParseCourseLanguage(string(*row.DetectedLanguage))
		if parseErr != nil {
			return contract.GenerationStatus{}, parseErr
		}
		status.DetectedLanguage = &language
	}
	status.ClarificationQuestions, err = clarificationQuestionsFromJSON(row.ClarificationQuestions)
	if err != nil {
		return contract.GenerationStatus{}, err
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
	answers, err := clarificationAnswersJSON(request.ClarificationAnswers)
	if err != nil {
		return dbsqlc.CreateGenerationRequestParams{}, err
	}
	confirmedGoals, err := confirmedGoalsJSON(request.ConfirmedBrief)
	if err != nil {
		return dbsqlc.CreateGenerationRequestParams{}, err
	}
	return dbsqlc.CreateGenerationRequestParams{
		ID:                        request.ID,
		ClerkUserID:               request.ClerkUserID,
		InitialUserPrompt:         request.InitialUserPrompt,
		PipelineStatus:            dbsqlc.GenerationPipelineStatus(request.PipelineStatus),
		CurrentStep:               request.CurrentStep,
		ProgressPercent:           int32(request.ProgressPercent),
		FailureMessage:            request.FailureMessage,
		StartedAt:                 request.StartedAt,
		CompletedAt:               request.CompletedAt,
		IsOutOfScope:              request.IsOutOfScope,
		ErrorMessage:              request.ErrorMessage,
		WarningMessage:            request.WarningMessage,
		SuggestedTitle:            request.SuggestedTitle,
		ShortSynopsis:             request.ShortSynopsis,
		DetectedCurrentLevel:      sqlcLevelPtr(request.DetectedCurrentLevel),
		DetectedTargetLevel:       sqlcLevelPtr(request.DetectedTargetLevel),
		DetectedGoal:              request.DetectedGoal,
		DetectedLanguage:          sqlcLanguagePtr(request.DetectedLanguage),
		ClarificationQuestions:    json.RawMessage(questions),
		RawAnalysisOutput:         rawJSONFromBytes(request.RawAnalysisOutput),
		AnalysisCompletedAt:       request.AnalysisCompletedAt,
		ClarificationAnswers:      json.RawMessage(answers),
		ConfirmedTitle:            briefString(request.ConfirmedBrief, func(brief domain.GenerationBrief) string { return brief.Title }),
		ConfirmedSynopsis:         briefString(request.ConfirmedBrief, func(brief domain.GenerationBrief) string { return brief.Synopsis }),
		ConfirmedCurrentLevel:     briefLevel(request.ConfirmedBrief, func(brief domain.GenerationBrief) domain.Level { return brief.CurrentLevel }),
		ConfirmedTargetLevel:      briefLevel(request.ConfirmedBrief, func(brief domain.GenerationBrief) domain.Level { return brief.TargetLevel }),
		ConfirmedGoals:            json.RawMessage(confirmedGoals),
		ConfirmedLanguage:         briefLanguage(request.ConfirmedBrief),
		BriefConfirmedAt:          request.BriefConfirmedAt,
		ClarificationsSubmittedAt: request.ClarificationsSubmittedAt,
		ClarificationVersion:      int32(request.ClarificationVersion),
		GenerationAttempt:         int32(request.GenerationAttempt),
		CreatedAt:                 request.CreatedAt,
		UpdatedAt:                 request.UpdatedAt,
	}, nil
}

func updateGenerationRequestParams(request domain.GenerationRequest) (dbsqlc.UpdateGenerationRequestParams, error) {
	questions, err := clarificationQuestionsJSON(request.ClarificationQuestions)
	if err != nil {
		return dbsqlc.UpdateGenerationRequestParams{}, err
	}
	answers, err := clarificationAnswersJSON(request.ClarificationAnswers)
	if err != nil {
		return dbsqlc.UpdateGenerationRequestParams{}, err
	}
	confirmedGoals, err := confirmedGoalsJSON(request.ConfirmedBrief)
	if err != nil {
		return dbsqlc.UpdateGenerationRequestParams{}, err
	}
	return dbsqlc.UpdateGenerationRequestParams{
		ClerkUserID:               request.ClerkUserID,
		InitialUserPrompt:         request.InitialUserPrompt,
		PipelineStatus:            dbsqlc.GenerationPipelineStatus(request.PipelineStatus),
		CurrentStep:               request.CurrentStep,
		ProgressPercent:           int32(request.ProgressPercent),
		FailureMessage:            request.FailureMessage,
		StartedAt:                 request.StartedAt,
		CompletedAt:               request.CompletedAt,
		IsOutOfScope:              request.IsOutOfScope,
		ErrorMessage:              request.ErrorMessage,
		WarningMessage:            request.WarningMessage,
		SuggestedTitle:            request.SuggestedTitle,
		ShortSynopsis:             request.ShortSynopsis,
		DetectedCurrentLevel:      sqlcLevelPtr(request.DetectedCurrentLevel),
		DetectedTargetLevel:       sqlcLevelPtr(request.DetectedTargetLevel),
		DetectedGoal:              request.DetectedGoal,
		DetectedLanguage:          sqlcLanguagePtr(request.DetectedLanguage),
		ClarificationQuestions:    json.RawMessage(questions),
		RawAnalysisOutput:         rawJSONFromBytes(request.RawAnalysisOutput),
		AnalysisCompletedAt:       request.AnalysisCompletedAt,
		ClarificationAnswers:      json.RawMessage(answers),
		ConfirmedTitle:            briefString(request.ConfirmedBrief, func(brief domain.GenerationBrief) string { return brief.Title }),
		ConfirmedSynopsis:         briefString(request.ConfirmedBrief, func(brief domain.GenerationBrief) string { return brief.Synopsis }),
		ConfirmedCurrentLevel:     briefLevel(request.ConfirmedBrief, func(brief domain.GenerationBrief) domain.Level { return brief.CurrentLevel }),
		ConfirmedTargetLevel:      briefLevel(request.ConfirmedBrief, func(brief domain.GenerationBrief) domain.Level { return brief.TargetLevel }),
		ConfirmedGoals:            json.RawMessage(confirmedGoals),
		ConfirmedLanguage:         briefLanguage(request.ConfirmedBrief),
		BriefConfirmedAt:          request.BriefConfirmedAt,
		ClarificationsSubmittedAt: request.ClarificationsSubmittedAt,
		ClarificationVersion:      int32(request.ClarificationVersion),
		GenerationAttempt:         int32(request.GenerationAttempt),
		UpdatedAt:                 request.UpdatedAt,
		ID:                        request.ID,
	}, nil
}

func confirmedGoalsJSON(brief *domain.GenerationBrief) (string, error) {
	if brief == nil {
		return stringSliceJSON(nil)
	}
	return stringSliceJSON(brief.Goals)
}

func briefString(brief *domain.GenerationBrief, selectValue func(domain.GenerationBrief) string) *string {
	if brief == nil {
		return nil
	}
	value := selectValue(*brief)
	return &value
}

func briefLevel(brief *domain.GenerationBrief, selectValue func(domain.GenerationBrief) domain.Level) *dbsqlc.Level {
	if brief == nil {
		return nil
	}
	value := dbsqlc.Level(selectValue(*brief))
	return &value
}

func briefLanguage(brief *domain.GenerationBrief) *dbsqlc.CourseLanguage {
	if brief == nil {
		return nil
	}
	value := dbsqlc.CourseLanguage(brief.Language)
	return &value
}
