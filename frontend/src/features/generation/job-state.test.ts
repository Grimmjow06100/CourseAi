import type { GenerationJob } from '@/shared/api/types'
import { contentJobState, latestTargetJobs } from './job-state'

function job(overrides: Partial<GenerationJob> = {}): GenerationJob {
  return {
    id: 'job',
    isCurrent: true,
    operationVersion: 1,
    supersedesJobId: null,
    requestId: 'request',
    parentJobId: null,
    targetId: 'lesson',
    kind: 'lesson_content',
    status: 'queued',
    priority: 0,
    attemptCount: 0,
    maxAttempts: 3,
    availableAt: '2026-09-06T00:00:00Z',
    lockedUntil: null,
    startedAt: null,
    completedAt: null,
    lastErrorCode: null,
    lastErrorMessage: null,
    createdAt: '2026-09-06T00:00:00Z',
    updatedAt: '2026-09-06T00:00:00Z',
    ...overrides,
  }
}

describe('partial generation state', () => {
  it('keeps a module active while children run after its own job completes', () => {
    expect(
      contentJobState(
        [
          job({ id: 'module-job', kind: 'module_content', targetId: 'module', status: 'completed' }),
          job({ status: 'running' }),
        ],
        ['module', 'lesson'],
      ),
    ).toEqual({ active: true, failed: false })
  })
  it('ends polling after failure and ignores unrelated targets', () => {
    expect(contentJobState([job({ status: 'failed' }), job({ targetId: 'other' })], ['lesson'])).toEqual({
      active: false,
      failed: true,
    })
  })
  it('uses the newest retry rather than a historical failure regardless of API ordering', () => {
    const retry = job({ id: 'retry', status: 'retry_scheduled', createdAt: '2026-09-06T01:00:00Z' })
    const failed = job({ status: 'failed', isCurrent: false })
    expect(latestTargetJobs([retry, failed])).toEqual([retry])
    expect(contentJobState([failed, retry], ['lesson'])).toEqual({ active: true, failed: false })
  })
  it('reports terminal cancellation instead of leaving generation busy forever', () => {
    expect(contentJobState([job({ status: 'cancelled' })], ['lesson'])).toEqual({
      active: false,
      failed: true,
    })
    expect(contentJobState([job({ status: 'completed' })], ['lesson'])).toEqual({
      active: false,
      failed: false,
    })
  })
})
