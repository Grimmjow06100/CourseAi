import type { GenerationSummary } from '@/shared/api/types'
import { shouldPollQuery } from '@/shared/api/polling'

type GenerationState = Pick<
  GenerationSummary,
  'pipelineStatus' | 'courseStatus' | 'courseId' | 'contentComplete'
>

export function hasCompleteCourse(state: GenerationState) {
  return Boolean(state.courseId) && (state.contentComplete ?? state.courseStatus === 'completed')
}

export function generationLabel(state: GenerationState) {
  if (hasCompleteCourse(state) && state.pipelineStatus !== 'completed') return 'generation.contentAvailable'
  return `generation.status.${state.pipelineStatus}`
}

export function shouldPollGenerations(
  states: GenerationState[],
  query: Parameters<typeof shouldPollQuery>[0],
) {
  return shouldPollQuery(
    query,
    states.some((state) => state.pipelineStatus === 'running' || state.pipelineStatus === 'queued'),
    states.some((state) => hasCompleteCourse(state) && state.pipelineStatus === 'failed'),
  )
}
