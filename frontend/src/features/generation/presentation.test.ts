import { generationLabel, hasCompleteCourse, shouldPollGenerations } from './presentation'
import { contentJobState } from './job-state'
import type { GenerationJob, GenerationSummary } from '@/shared/api/types'

const failed: GenerationSummary = {
  requestId: 'request',
  courseId: 'course',
  courseStatus: 'failed',
  pipelineStatus: 'failed',
  initialUserPrompt: '',
  title: '',
  currentStep: null,
  progressPercent: 75,
  isOutOfScope: false,
  failureMessage: null,
  createdAt: '',
  updatedAt: '',
  contentComplete: false,
}

it('does not infer completion from a course id or a stale course status', () => {
  expect(hasCompleteCourse(failed)).toBe(false)
  expect(hasCompleteCourse({ ...failed, courseStatus: 'completed' })).toBe(false)
  expect(generationLabel(failed)).toBe('generation.status.failed')
})

it('shows availability separately and bounds reconciliation polling', () => {
  const complete = { ...failed, contentComplete: true }
  const query = { state: { dataUpdateCount: 100 } }
  expect(generationLabel(complete)).toBe('generation.contentAvailable')
  expect(shouldPollGenerations([complete], query)).toBe(true)
  query.state.dataUpdateCount += 30
  expect(shouldPollGenerations([complete], query)).toBe(false)
})

it('ignores failed targets belonging to older generation attempts', () => {
  const old = {
    id: 'old',
    generationAttempt: 1,
    targetId: 'lesson',
    kind: 'lesson_content',
    status: 'failed',
  } as GenerationJob
  const current = {
    id: 'new',
    generationAttempt: 2,
    targetId: 'module',
    kind: 'module_content',
    status: 'running',
  } as GenerationJob
  expect(contentJobState([old, current], ['lesson', 'module'])).toEqual({ active: true, failed: false })
})
