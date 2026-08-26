//go:build integration

package postgresintegration_test

import (
	"context"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	"github.com/Grimmjow06100/course-ai/backend-go/tests/testkit"
	"github.com/jackc/pgx/v5"
)

func TestClerkUserIDPersistenceAndTransitiveOwnership(t *testing.T) {
	ctx, pool := testkit.OpenPostgres(t, 20*time.Second)
	tx := testkit.BeginRollback(t, ctx, pool)
	assertTableDoesNotExist(t, ctx, tx, "users")
	assertTableDoesNotExist(t, ctx, tx, "clerk_webhook_events")

	now := time.Now().UTC().Truncate(time.Millisecond)
	clerkUserID := "user_integration"

	request, err := domain.NewGenerationRequestAt("Linux", clerkUserID, now)
	if err != nil {
		t.Fatal(err)
	}
	request, err = postgres.NewGenerationRequestRepository(tx).SaveGenerationRequest(ctx, request)
	if err != nil {
		t.Fatalf("save generation request: %v", err)
	}

	course, err := domain.NewCourseAt(domain.NewCourseParams{
		RequestID:         request.ID,
		ClerkUserID:       clerkUserID,
		Language:          domain.CourseLanguageEN,
		InitialUserPrompt: request.InitialUserPrompt,
		Title:             "Linux",
		Synopsis:          "Linux course",
		CurrentLevel:      domain.LevelBeginner,
		TargetLevel:       domain.LevelAdvanced,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	course, err = postgres.NewCourseRepository(tx).SaveCourse(ctx, course)
	if err != nil {
		t.Fatalf("save course: %v", err)
	}
	if course.ClerkUserID != clerkUserID {
		t.Fatalf("course clerk user id = %q, want %q", course.ClerkUserID, clerkUserID)
	}

	module, err := domain.NewModuleAt(domain.NewModuleParams{
		CourseID: course.ID, Order: 1, Title: "Foundations", Description: "Linux foundations",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	module, err = postgres.NewModuleRepository(tx).SaveModule(ctx, module)
	if err != nil {
		t.Fatalf("save module: %v", err)
	}

	lesson, err := domain.NewLessonAt(domain.NewLessonParams{
		ModuleID: module.ID, Order: 1, Title: "Shell", Type: domain.LessonTypeTheory,
		EstimatedDurationMinutes: 20, LearningGoal: "Use the shell",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	lesson, err = postgres.NewLessonRepository(tx).SaveLesson(ctx, lesson)
	if err != nil {
		t.Fatalf("save lesson: %v", err)
	}

	ownership := postgres.NewOwnershipRepository(tx)
	assertOwned := func(name string, lookup func(string) (bool, error)) {
		t.Helper()
		owned, err := lookup(clerkUserID)
		if err != nil || !owned {
			t.Fatalf("%s owner lookup = %t, %v", name, owned, err)
		}
		owned, err = lookup("user_other")
		if err != nil || owned {
			t.Fatalf("%s cross-owner lookup = %t, %v", name, owned, err)
		}
	}

	assertOwned("request", func(userID string) (bool, error) {
		return ownership.OwnsGenerationRequest(ctx, request.ID, userID)
	})
	assertOwned("course", func(userID string) (bool, error) {
		return ownership.OwnsCourse(ctx, course.ID, userID)
	})
	assertOwned("module", func(userID string) (bool, error) {
		return ownership.OwnsModule(ctx, module.ID, userID)
	})
	assertOwned("lesson", func(userID string) (bool, error) {
		return ownership.OwnsLesson(ctx, lesson.ID, userID)
	})
}

func assertTableDoesNotExist(t *testing.T, ctx context.Context, tx pgx.Tx, tableName string) {
	t.Helper()

	var relationName *string
	if err := tx.QueryRow(ctx, "SELECT to_regclass($1)::text", "public."+tableName).Scan(&relationName); err != nil {
		t.Fatalf("look up table %s: %v", tableName, err)
	}
	if relationName != nil {
		t.Fatalf("table %s still exists", tableName)
	}
}
