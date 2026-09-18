import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useApiClient } from '@/shared/api/context'
import { unwrapApiResult } from '@/shared/api/errors'
import type { GenerationJob } from '@/shared/api/types'
import { courseKeys } from '@/features/catalog/query-keys'
import { generationKeys } from './query-keys'
import { isJobActive, latestTargetJobs } from './job-state'

export function useRequestJobs(requestId: string) {
  const client = useApiClient()
  const cache = useQueryClient()
  return useQuery({
    queryKey: generationKeys.jobs(requestId),
    queryFn: async ({ signal }) => {
      const jobs = unwrapApiResult<GenerationJob[]>(
        await client.GET('/api/generations/{requestID}/jobs', {
          params: { path: { requestID: requestId } },
          signal,
        }),
      )
      const previous = cache.getQueryData<GenerationJob[]>(generationKeys.jobs(requestId))
      const hasNewTerminal = jobs.some(
        (job) =>
          !isJobActive(job) && !previous?.some((old) => old.id === job.id && old.status === job.status),
      )
      if (hasNewTerminal) {
        void cache.invalidateQueries({ queryKey: courseKeys.all })
        void cache.invalidateQueries({ queryKey: generationKeys.lists() })
        void cache.invalidateQueries({ queryKey: generationKeys.detail(requestId) })
      }
      return jobs
    },
    staleTime: 0,
    refetchOnWindowFocus: true,
    refetchInterval: (query) =>
      !query.state.error && latestTargetJobs(query.state.data ?? []).some(isJobActive) ? 2_000 : false,
  })
}
