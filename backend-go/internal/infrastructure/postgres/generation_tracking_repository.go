package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/google/uuid"
)

type GenerationTrackingRepository struct{ queries *dbsqlc.Queries }

var _ contract.GenerationTrackingRepository = (*GenerationTrackingRepository)(nil)

func NewGenerationTrackingRepository(db DBTX) *GenerationTrackingRepository {
	return &GenerationTrackingRepository{queries: dbsqlc.New(db)}
}
func (r *GenerationTrackingRepository) Snapshot(ctx context.Context, id uuid.UUID) (contract.GenerationTracking, error) {
	data, err := r.queries.GetGenerationTrackingSnapshot(ctx, id)
	var result contract.GenerationTracking
	if err != nil {
		return result, mapNoRows(err, contract.ErrGenerationRequestNotFound)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("decode tracking snapshot: %w", err)
	}
	return result, nil
}
func (r *GenerationTrackingRepository) Events(ctx context.Context, id uuid.UUID, cursor int64, limit int) ([]contract.GenerationEvent, error) {
	rows, err := r.queries.ListPublicGenerationEvents(ctx, dbsqlc.ListPublicGenerationEventsParams{RequestID: id, CursorID: cursor, LimitRows: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := make([]contract.GenerationEvent, 0, len(rows))
	for _, row := range rows {
		item := contract.GenerationEvent{ID: strconv.FormatInt(row.ID, 10), GenerationAttempt: int(row.GenerationAttempt), JobID: uuidPointerFromPGType(row.JobID), Kind: row.Kind, Status: row.Status, TargetID: uuidPointerFromPGType(row.TargetID), OccurredAt: row.OccurredAt}
		if row.OperationVersion != nil {
			v := int(*row.OperationVersion)
			item.OperationVersion = &v
		}
		if row.AttemptCount != nil {
			v := int(*row.AttemptCount)
			item.AttemptCount = &v
		}
		items = append(items, item)
	}
	return items, nil
}
func (r *GenerationTrackingRepository) Supersede(ctx context.Context, id uuid.UUID, version int) error {
	n, err := r.queries.SupersedeGenerationOperation(ctx, dbsqlc.SupersedeGenerationOperationParams{ID: id, OperationVersion: int32(version)})
	if err != nil {
		return err
	}
	if n != 1 {
		return contract.ErrGenerationJobIdempotencyConflict
	}
	return nil
}
func (r *GenerationTrackingRepository) RetryReceipt(ctx context.Context, id uuid.UUID, key string) (json.RawMessage, json.RawMessage, error) {
	row, err := r.queries.GetGenerationRetryReceipt(ctx, dbsqlc.GetGenerationRetryReceiptParams{RequestID: id, IdempotencyKey: key})
	return row.Payload, row.Result, mapNoRows(err, contract.ErrGenerationJobNotFound)
}
func (r *GenerationTrackingRepository) SaveRetryReceipt(ctx context.Context, id uuid.UUID, key string, payload, result json.RawMessage) error {
	return r.queries.SaveGenerationRetryReceipt(ctx, dbsqlc.SaveGenerationRetryReceiptParams{RequestID: id, IdempotencyKey: key, Payload: payload, Result: result})
}
