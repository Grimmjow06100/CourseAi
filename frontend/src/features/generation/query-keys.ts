import type { PipelineStatus } from '@/shared/api/types'

export const generationKeys = {
  all: ['generations'] as const,
  lists: () => [...generationKeys.all, 'list'] as const,
  list: (filters: { status?: PipelineStatus; page: number; pageSize: number }) =>
    [...generationKeys.lists(), filters] as const,
  detail: (requestId: string) => [...generationKeys.all, 'detail', requestId] as const,
  job: (jobId: string) => [...generationKeys.all, 'job', jobId] as const,
  jobs: (requestId: string) => [...generationKeys.all, 'jobs', requestId] as const,
  tracking: (requestId: string) => [...generationKeys.all, 'tracking', requestId] as const,
  events: (requestId: string) => [...generationKeys.all, 'events', requestId] as const,
}
