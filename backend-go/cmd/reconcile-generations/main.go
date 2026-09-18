// Command reconcile-generations inspects completed-content inconsistencies.
// It is read-only unless -apply is explicitly supplied, and never invokes AI.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/config"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/db"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/clock"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
	"github.com/google/uuid"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	apply := flag.Bool("apply", false, "Apply guarded reconciliation; default is read-only")
	limit := flag.Int("limit", 100, "Maximum candidates (1-1000)")
	requestID := flag.String("request-id", "", "Inspect/reconcile only this request UUID")
	flag.Parse()
	if *limit < 1 || *limit > 1000 {
		return fmt.Errorf("limit must be between 1 and 1000")
	}
	var ids []uuid.UUID
	if *requestID != "" {
		id, err := uuid.Parse(*requestID)
		if err != nil {
			return fmt.Errorf("invalid request-id: %w", err)
		}
		ids = []uuid.UUID{id}
	}
	if err := config.Load(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := db.Open(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	repo := postgres.NewGenerationRequestRepository(pool)
	if len(ids) == 0 {
		ids, err = repo.ListCompletionCandidates(ctx, *limit)
		if err != nil {
			return err
		}
	}
	generator := service.NewCourseGeneratorService(nil, postgres.NewUnitOfWork(pool), clock.NewSystemClock(), service.CourseGeneratorConfig{})
	output := json.NewEncoder(os.Stdout)
	for _, id := range ids {
		status, err := repo.FindGenerationStatusByID(ctx, id)
		if err != nil {
			return err
		}
		// Write a safe pre-change snapshot before any repair; do not include prompts or raw errors.
		if err := output.Encode(map[string]any{"requestId": id, "pipelineStatus": status.PipelineStatus, "courseStatus": status.CourseStatus, "contentComplete": status.ContentComplete, "generationAttempt": status.GenerationAttempt, "apply": *apply}); err != nil {
			return err
		}
		if *apply {
			completed, err := generator.ReconcileCompletedGeneration(ctx, id)
			if err != nil {
				return err
			}
			if err := output.Encode(map[string]any{"requestId": id, "completed": completed}); err != nil {
				return err
			}
		}
	}
	return nil
}
