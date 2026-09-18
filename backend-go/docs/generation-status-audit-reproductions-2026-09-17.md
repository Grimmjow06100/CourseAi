# Reproductions de l'audit des statuts - 2026-09-17

## Portee

Annexe de `generation-status-audit-2026-09-17.md`.
Revision auditee : `024575900a78f1a62c6ca01732250db67b068292`.

Ces tests de caracterisation **confirment des comportements incorrects existants**.
Leur succes ne valide pas le fonctionnement attendu du produit. Apres correction,
il faut remplacer les assertions par les invariants attendus decrits dans le rapport.
Les sondes ont ete executees puis retirees des repertoires de tests actifs.
Aucun appel OpenAI, aucun jeton Clerk reel, aucune ecriture en base ni en production.

## Backend

Emplacement temporaire : `backend-go/internal/service/generation_status_audit_test.go`.
Reutilise les doubles en memoire deja presents dans `course_generation_pipeline_test.go`.
Les erreurs injectees surviennent avant les ecritures concernees. Les doubles n'emulent
ni le verrouillage PostgreSQL ni les rollbacks : ces tests prouvent l'enchainement applicatif,
pas une course concurrente reelle ni la cause de l'incident de production.

Commande depuis `backend-go` :

```powershell
$env:GOMAXPROCS='2'
go test -p 1 -count=1 -run TestStatusAudit -v ./internal/service
```

Resultat : 4 tests passes.

```text
all content=true, course=failed, request=failed
all content=true, course=completed, request=failed; finalizer blocked
old version=1, new version=2, old failure makes new request=failed
command accepted: job=queued, request=failed; execution blocked
```

```go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

// Temporary characterization probes for the status audit, not desired-behavior tests.
type statusAuditUOW struct {
	store               *pipelineMemoryStore
	failFinalizeEnqueue bool
	failFinalProgress   bool
}

func (u *statusAuditUOW) WithinTx(ctx context.Context, fn func(context.Context, contract.TransactionalRepositories) error) error {
	return fn(ctx, statusAuditRepositories{TransactionalRepositories: pipelineMemoryRepositories{store: u.store}, uow: u})
}

type statusAuditRepositories struct {
	contract.TransactionalRepositories
	uow *statusAuditUOW
}

func (r statusAuditRepositories) GenerationJobs() contract.GenerationJobQueue {
	return statusAuditQueue{GenerationJobQueue: r.TransactionalRepositories.GenerationJobs(), uow: r.uow}
}

func (r statusAuditRepositories) GenerationRequests() contract.GenerationRequestRepository {
	return statusAuditRequests{GenerationRequestRepository: r.TransactionalRepositories.GenerationRequests(), uow: r.uow}
}

type statusAuditQueue struct {
	contract.GenerationJobQueue
	uow *statusAuditUOW
}

func (q statusAuditQueue) Enqueue(ctx context.Context, job domain.GenerationJob) (domain.GenerationJob, error) {
	if q.uow.failFinalizeEnqueue && job.Kind == domain.GenerationJobKindFinalizeCourse {
		return domain.GenerationJob{}, errors.New("audit: finalize enqueue unavailable")
	}
	return q.GenerationJobQueue.Enqueue(ctx, job)
}

type statusAuditRequests struct {
	contract.GenerationRequestRepository
	uow *statusAuditUOW
}

func (r statusAuditRequests) UpdateGenerationRequest(ctx context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	if r.uow.failFinalProgress && request.ProgressPercent == 95 {
		return domain.GenerationRequest{}, errors.New("audit: final progress persistence unavailable")
	}
	return r.GenerationRequestRepository.UpdateGenerationRequest(ctx, request)
}

func prepareStatusAudit(t *testing.T) (*CourseGeneratorService, *statusAuditUOW, uuid.UUID, domain.GenerationJob) {
	t.Helper()
	uow := &statusAuditUOW{store: newPipelineMemoryStore()}
	s := NewCourseGeneratorService(&pipelineAIStub{}, uow, fixedClock{now: time.Date(2026, 9, 17, 16, 0, 0, 0, time.UTC)}, CourseGeneratorConfig{})
	started, err := s.StartFullCourseGeneration(authenticatedTestContext(), contract.StartGenerationParams{Prompt: "Linux course"})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []domain.GenerationJobKind{domain.GenerationJobKindAnalysis, domain.GenerationJobKindArchitecture, domain.GenerationJobKindLessonPlan} {
		job := pipelineJobByKind(t, uow.store, started.RequestID, kind)
		var err error
		switch kind {
		case domain.GenerationJobKindAnalysis:
			err = s.runAnalysisJob(authenticatedTestContext(), job)
		case domain.GenerationJobKindArchitecture:
			err = s.runArchitectureJob(authenticatedTestContext(), job)
		case domain.GenerationJobKindLessonPlan:
			err = s.runLessonPlanJob(authenticatedTestContext(), job)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	return s, uow, started.RequestID, pipelineJobByKind(t, uow.store, started.RequestID, domain.GenerationJobKindLessonContent)
}

func TestStatusAuditCompleteContentCanBeFailed(t *testing.T) {
	s, uow, id, contentJob := prepareStatusAudit(t)
	uow.failFinalizeEnqueue = true
	cause := s.runLessonContentJob(authenticatedTestContext(), contentJob)
	if cause == nil {
		t.Fatal("expected injected enqueue failure")
	}
	if err := s.handleTerminalJobFailure(authenticatedTestContext(), contentJob, cause); err != nil {
		t.Fatal(err)
	}
	course, err := uow.store.courseByRequestID(id)
	if err != nil {
		t.Fatal(err)
	}
	if !course.HasCompleteContent() || course.Status != domain.CourseStatusFailed || uow.store.requests[id].PipelineStatus != domain.PipelineStatusFailed {
		t.Fatal("inconsistency not reproduced")
	}
	t.Logf("all content=%t, course=%s, request=%s", course.HasCompleteContent(), course.Status, uow.store.requests[id].PipelineStatus)
}

func TestStatusAuditCompletedCourseCanHaveFailedRequest(t *testing.T) {
	s, uow, id, contentJob := prepareStatusAudit(t)
	if err := s.runLessonContentJob(authenticatedTestContext(), contentJob); err != nil {
		t.Fatal(err)
	}
	finalize := pipelineJobByKind(t, uow.store, id, domain.GenerationJobKindFinalizeCourse)
	uow.failFinalProgress = true
	cause := s.runFinalizeCourseJob(authenticatedTestContext(), finalize)
	if cause == nil {
		t.Fatal("expected injected progress failure")
	}
	if err := s.handleTerminalJobFailure(authenticatedTestContext(), finalize, cause); err != nil {
		t.Fatal(err)
	}
	course, err := uow.store.courseByRequestID(id)
	if err != nil {
		t.Fatal(err)
	}
	if course.Status != domain.CourseStatusCompleted || uow.store.requests[id].PipelineStatus != domain.PipelineStatusFailed {
		t.Fatal("inconsistency not reproduced")
	}
	if err := s.runFinalizeCourseJob(authenticatedTestContext(), finalize); !errors.Is(err, ErrGenerationNotRetryable) {
		t.Fatalf("expected blocked finalizer, got %v", err)
	}
	t.Logf("all content=%t, course=%s, request=%s; finalizer blocked", course.HasCompleteContent(), course.Status, uow.store.requests[id].PipelineStatus)
}

func TestStatusAuditOldFailurePoisonsNewAttempt(t *testing.T) {
	s, uow, id, contentJob := prepareStatusAudit(t)
	if err := s.handleTerminalJobFailure(authenticatedTestContext(), contentJob, errors.New("old attempt failed")); err != nil {
		t.Fatal(err)
	}
	oldVersion := uow.store.requests[id].ClarificationVersion
	if _, err := s.RetryFullCourseGeneration(authenticatedTestContext(), id); err != nil {
		t.Fatal(err)
	}
	newVersion := uow.store.requests[id].ClarificationVersion
	if newVersion <= oldVersion || uow.store.requests[id].PipelineStatus != domain.PipelineStatusRunning {
		t.Fatal("retry did not start")
	}
	if err := s.handleTerminalJobFailure(authenticatedTestContext(), contentJob, errors.New("late reconciliation of old failure")); err != nil {
		t.Fatal(err)
	}
	if uow.store.requests[id].PipelineStatus != domain.PipelineStatusFailed {
		t.Fatal("old failure did not poison new attempt")
	}
	t.Logf("old version=%d, new version=%d, old failure makes new request=%s", oldVersion, newVersion, uow.store.requests[id].PipelineStatus)
}

func TestStatusAuditTargetedJobAcceptedButCannotRunOnFailedRequest(t *testing.T) {
	s, uow, id, contentJob := prepareStatusAudit(t)
	if err := s.handleTerminalJobFailure(authenticatedTestContext(), contentJob, errors.New("content failed")); err != nil {
		t.Fatal(err)
	}
	course, err := uow.store.courseByRequestID(id)
	if err != nil {
		t.Fatal(err)
	}
	started, err := s.EnqueueModuleContentGeneration(authenticatedTestContext(), course.Modules[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	job, err := uow.store.jobs.FindByID(authenticatedTestContext(), started.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.runModuleContentJob(authenticatedTestContext(), job); !errors.Is(err, ErrGenerationNotRetryable) {
		t.Fatalf("expected failed request guard, got %v", err)
	}
	t.Logf("command accepted: job=%s, request=%s; execution blocked", started.JobStatus, started.Status)
}
```

## Frontend

Emplacement temporaire : `frontend/src/features/generation/status-audit.test.tsx`.
Vitest/jsdom avec les vrais hooks TanStack Query et le vrai composant de suivi.
Seul le client HTTP et la traduction sont simules ; les timers sont controles.

Commande depuis `frontend` :

```powershell
npm.cmd test -- --maxWorkers=1 src/features/generation/status-audit.test.tsx
```

Resultat : 4 tests passes. Le premier lancement etait bloque par `spawn EPERM`
dans la sandbox Windows ; le lancement autorise hors sandbox a reussi.

```tsx
import {
  act,
  cleanup,
  render,
  renderHook,
  screen,
} from "@testing-library/react";
import { focusManager, QueryClientProvider } from "@tanstack/react-query";
import type { PropsWithChildren } from "react";
import { createQueryClient } from "@/app/query-client";
import { generationKeys } from "./query-keys";
import { useGenerationList, useGenerationStatus } from "./api";
import { useRequestJobs } from "./jobs";
import { GenerationTracker } from "./generation-tracker";
import type { GenerationStatus } from "@/shared/api/types";

const { get } = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/shared/api/context", () => ({ useApiClient: () => ({ GET: get }) }));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

function setup() {
  const cache = createQueryClient();
  function Wrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={cache}>{children}</QueryClientProvider>;
  }
  return { cache, Wrapper };
}

function success(data: unknown) {
  return { data, response: new Response("{}", { status: 200 }) };
}

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  get.mockReset();
  focusManager.setFocused(undefined);
});

it("audit: failed pipeline hides the available completed course", () => {
  const { cache, Wrapper } = setup();
  render(
    <GenerationTracker
      status={
        {
          requestId: "request",
          courseId: "course",
          courseStatus: "completed",
          pipelineStatus: "failed",
          progressPercent: 95,
          failureMessage:
            "generation failed; retry the operation or contact support with the request id",
          isOutOfScope: false,
        } as GenerationStatus
      }
    />,
    { wrapper: Wrapper },
  );
  expect(screen.getByText("generation.failed")).toBeInTheDocument();
  expect(screen.queryByRole("link")).not.toBeInTheDocument();
  cache.clear();
});

it("audit: a failed status stays cached after the API recovers and window focus returns", async () => {
  vi.useFakeTimers();
  const { cache, Wrapper } = setup();
  get.mockResolvedValue(
    success({ pipelineStatus: "failed", requestId: "request" }),
  );
  const hook = renderHook(() => useGenerationStatus("request"), {
    wrapper: Wrapper,
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(10);
  });
  expect(hook.result.current.data?.pipelineStatus).toBe("failed");
  get.mockResolvedValue(
    success({ pipelineStatus: "completed", requestId: "request" }),
  );
  await act(async () => {
    await vi.advanceTimersByTimeAsync(60_000);
    focusManager.setFocused(false);
    focusManager.setFocused(true);
    await vi.advanceTimersByTimeAsync(10);
  });
  expect(get).toHaveBeenCalledTimes(1);
  expect(hook.result.current.data?.pipelineStatus).toBe("failed");
  hook.unmount();
  cache.clear();
});

it("audit: terminal jobs invalidate lists but not the generation detail", async () => {
  vi.useFakeTimers();
  const { cache, Wrapper } = setup();
  cache.setQueryData(generationKeys.detail("request"), {
    pipelineStatus: "failed",
  });
  cache.setQueryData(generationKeys.list({ page: 1, pageSize: 20 }), {
    items: [],
  });
  get.mockResolvedValue(success([{ id: "job", status: "completed" }]));
  const hook = renderHook(() => useRequestJobs("request"), {
    wrapper: Wrapper,
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(10);
  });
  expect(
    cache.getQueryState(generationKeys.detail("request"))?.isInvalidated,
  ).toBe(false);
  expect(
    cache.getQueryState(generationKeys.list({ page: 1, pageSize: 20 }))
      ?.isInvalidated,
  ).toBe(true);
  hook.unmount();
  cache.clear();
});

it("audit: the mounted history does not refresh an active generation by itself", async () => {
  vi.useFakeTimers();
  const { cache, Wrapper } = setup();
  get.mockResolvedValue(
    success({ items: [{ pipelineStatus: "running" }], total: 1 }),
  );
  const hook = renderHook(() => useGenerationList({ page: 1 }), {
    wrapper: Wrapper,
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(10);
  });
  get.mockResolvedValue(
    success({ items: [{ pipelineStatus: "completed" }], total: 1 }),
  );
  await act(async () => {
    await vi.advanceTimersByTimeAsync(60_000);
  });
  expect(get).toHaveBeenCalledTimes(1);
  hook.unmount();
  cache.clear();
});
```
