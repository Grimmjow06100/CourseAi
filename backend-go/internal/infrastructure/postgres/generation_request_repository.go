package postgres

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type GenerationRequestRepository struct {
	db DBTX
}

func NewGenerationRequestRepository(db DBTX) *GenerationRequestRepository {
	return &GenerationRequestRepository{db: db}
}

func (r *GenerationRequestRepository) SaveGenerationRequest(ctx context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	if err := request.Validate(); err != nil {
		return domain.GenerationRequest{}, err
	}

	questionsJSON, err := clarificationQuestionsJSON(request.ClarificationQuestions)
	if err != nil {
		return domain.GenerationRequest{}, err
	}

	savedRequest, err := scanGenerationRequest(r.db.QueryRow(ctx, `
		INSERT INTO generation_requests (
			id,
			initial_user_prompt,
			pipeline_status,
			current_step,
			progress_percent,
			failure_message,
			started_at,
			completed_at,
			is_out_of_scope,
			error_message,
			warning_message,
			suggested_title,
			short_synopsis,
			detected_current_level,
			detected_target_level,
			detected_goal,
			detected_language,
			clarification_questions,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18::jsonb, $19, $20)
		RETURNING `+generationRequestColumns+`
	`,
		request.ID,
		request.InitialUserPrompt,
		string(request.PipelineStatus),
		textValue(request.CurrentStep),
		request.ProgressPercent,
		textValue(request.FailureMessage),
		timeValue(request.StartedAt),
		timeValue(request.CompletedAt),
		request.IsOutOfScope,
		textValue(request.ErrorMessage),
		textValue(request.WarningMessage),
		textValue(request.SuggestedTitle),
		textValue(request.ShortSynopsis),
		levelValue(request.DetectedCurrentLevel),
		levelValue(request.DetectedTargetLevel),
		textValue(request.DetectedGoal),
		languageValue(request.DetectedLanguage),
		questionsJSON,
		request.CreatedAt,
		request.UpdatedAt,
	))
	if err != nil {
		return domain.GenerationRequest{}, err
	}
	return savedRequest, nil
}

func (r *GenerationRequestRepository) UpdateGenerationRequest(ctx context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	if err := request.Validate(); err != nil {
		return domain.GenerationRequest{}, err
	}

	questionsJSON, err := clarificationQuestionsJSON(request.ClarificationQuestions)
	if err != nil {
		return domain.GenerationRequest{}, err
	}

	updatedRequest, err := scanGenerationRequest(r.db.QueryRow(ctx, `
		UPDATE generation_requests
		SET
			initial_user_prompt = $2,
			pipeline_status = $3,
			current_step = $4,
			progress_percent = $5,
			failure_message = $6,
			started_at = $7,
			completed_at = $8,
			is_out_of_scope = $9,
			error_message = $10,
			warning_message = $11,
			suggested_title = $12,
			short_synopsis = $13,
			detected_current_level = $14,
			detected_target_level = $15,
			detected_goal = $16,
			detected_language = $17,
			clarification_questions = $18::jsonb,
			updated_at = $19
		WHERE id = $1
		RETURNING `+generationRequestColumns+`
	`,
		request.ID,
		request.InitialUserPrompt,
		string(request.PipelineStatus),
		textValue(request.CurrentStep),
		request.ProgressPercent,
		textValue(request.FailureMessage),
		timeValue(request.StartedAt),
		timeValue(request.CompletedAt),
		request.IsOutOfScope,
		textValue(request.ErrorMessage),
		textValue(request.WarningMessage),
		textValue(request.SuggestedTitle),
		textValue(request.ShortSynopsis),
		levelValue(request.DetectedCurrentLevel),
		levelValue(request.DetectedTargetLevel),
		textValue(request.DetectedGoal),
		languageValue(request.DetectedLanguage),
		questionsJSON,
		request.UpdatedAt,
	))
	if err != nil {
		return domain.GenerationRequest{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}
	return updatedRequest, nil
}

func (r *GenerationRequestRepository) FindGenerationRequestByID(ctx context.Context, id uuid.UUID) (domain.GenerationRequest, error) {
	request, err := scanGenerationRequest(r.db.QueryRow(ctx, `
		SELECT `+generationRequestColumns+`
		FROM generation_requests
		WHERE id = $1
	`, id))
	if err != nil {
		return domain.GenerationRequest{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}
	return request, nil
}

func (r *GenerationRequestRepository) FindGenerationRequestByCourseID(ctx context.Context, courseID uuid.UUID) (domain.GenerationRequest, error) {
	request, err := scanGenerationRequest(r.db.QueryRow(ctx, `
		SELECT `+generationRequestColumns+`
		FROM generation_requests
		WHERE id = (
			SELECT request_id
			FROM courses
			WHERE id = $1
		)
	`, courseID))
	if err != nil {
		return domain.GenerationRequest{}, mapNoRows(err, ErrGenerationRequestNotFound)
	}
	return request, nil
}
