import type { GenerationTracking, TrackingOperation } from '@/shared/api/types'

export function trackingFixture(overrides: Partial<GenerationTracking> = {}): GenerationTracking {
  return {
    requestId: '00000000-0000-4000-8000-000000000001',
    generationAttempt: 1,
    revision: '12',
    observedAt: '2026-09-19T12:00:00Z',
    pipelineStatus: 'running',
    title: 'Linux administration',
    courseId: '00000000-0000-4000-8000-000000000002',
    createdAt: '2026-09-19T11:00:00Z',
    completedAt: null,
    isOutOfScope: false,
    historyComplete: true,
    contentComplete: false,
    contentAvailability: 'partial',
    hasActiveWork: true,
    reconciliation: 'none',
    counts: {
      modules: 1,
      plansReady: 1,
      lessonsAvailable: 1,
      lessonsExpected: 3,
      jobsActive: 1,
      jobsFailed: 1,
      jobsCompleted: 1,
    },
    phases: [
      { kind: 'analysis', status: 'completed' },
      { kind: 'architecture', status: 'completed' },
      { kind: 'lesson_plan', status: 'completed' },
      { kind: 'lesson_content', status: 'running' },
      { kind: 'finalize_course', status: 'pending' },
    ],
    modules: [],
    operations: [],
    ...overrides,
  }
}

export function operationFixture(overrides: Partial<TrackingOperation> = {}): TrackingOperation {
  return {
    id: '00000000-0000-4000-8000-000000000003',
    targetId: null,
    parentJobId: null,
    kind: 'lesson_content',
    status: 'failed',
    operationVersion: 1,
    supersedesJobId: null,
    attemptCount: 3,
    maxAttempts: 3,
    availableAt: '2026-09-19T11:00:00Z',
    startedAt: null,
    completedAt: '2026-09-19T12:00:00Z',
    retryable: true,
    failureCode: null,
    ...overrides,
  }
}
