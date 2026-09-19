import { courseKeys, generationKeys } from '@/shared/api/query-keys'
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useRef } from 'react'
import { z } from 'zod'
import { useApiClient } from '@/shared/api/context'
import { ApiError, unwrapApiResult } from '@/shared/api/errors'
import { shouldPollQuery } from '@/shared/api/polling'
import type {
  GenerationEventPage,
  GenerationTracking,
  RetryOperation,
  RetryOperationsResult,
} from '@/shared/api/types'
import { newestTracking } from './tracking-state'

export function useGenerationTracking(requestId: string) {
  const client = useApiClient()
  const cache = useQueryClient()
  const key = generationKeys.tracking(requestId)
  return useQuery({
    queryKey: key,
    staleTime: 0,
    refetchOnWindowFocus: 'always',
    refetchOnReconnect: 'always',
    refetchIntervalInBackground: false,
    queryFn: async ({ signal }) => {
      const incoming = unwrapApiResult<GenerationTracking>(
        await client.GET('/api/generations/{requestID}/tracking', {
          params: { path: { requestID: requestId } },
          signal,
        }),
      )
      const previous = cache.getQueryData<GenerationTracking>(key)
      const snapshot = newestTracking(previous, incoming)
      if (
        snapshot.pipelineStatus !== previous?.pipelineStatus ||
        snapshot.counts.lessonsAvailable !== previous.counts.lessonsAvailable ||
        snapshot.generationAttempt !== previous.generationAttempt
      ) {
        void cache.invalidateQueries({ queryKey: generationKeys.lists() })
        void cache.invalidateQueries({ queryKey: courseKeys.all })
      }
      return snapshot
    },
    refetchInterval(query) {
      if (query.state.error instanceof ApiError && [401, 403, 404].includes(query.state.error.status))
        return false
      const state = query.state.data
      const active = state?.hasActiveWork === true || state?.pipelineStatus === 'queued'
      const recoverable =
        state?.reconciliation === 'pending' || (state?.pipelineStatus === 'running' && !state.hasActiveWork)
      return shouldPollQuery(query, active, recoverable) ? (query.state.error ? 10_000 : 2_000) : false
    },
  })
}

export function useGenerationEvents(requestId: string, enabled: boolean, active: boolean) {
  const client = useApiClient()
  return useInfiniteQuery({
    queryKey: generationKeys.events(requestId),
    enabled,
    initialPageParam: '0',
    maxPages: 5,
    queryFn: async ({ signal, pageParam }) =>
      unwrapApiResult<GenerationEventPage>(
        await client.GET('/api/generations/{requestID}/events', {
          params: { path: { requestID: requestId }, query: { cursor: pageParam, limit: 30 } },
          signal,
        }),
      ),
    getNextPageParam: (page) => page.nextCursor ?? undefined,
    refetchInterval: (query) => (active && query.state.data?.pages.length === 1 ? 5_000 : false),
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
  })
}

const selectionSchema = z
  .array(z.object({ jobId: z.uuid(), operationVersion: z.number().int().positive() }))
  .min(1)
  .max(500)

export function useRetryOperations(requestId: string) {
  const client = useApiClient()
  const cache = useQueryClient()
  const keys = useRef(new Map<string, string>())
  return useMutation({
    mutationFn: async (selection: RetryOperation[]) => {
      const operations = selectionSchema.parse(selection).sort((a, b) => a.jobId.localeCompare(b.jobId))
      const signature = JSON.stringify(operations)
      const key = keys.current.get(signature) ?? crypto.randomUUID()
      keys.current.set(signature, key)
      return unwrapApiResult<RetryOperationsResult>(
        await client.POST('/api/generations/{requestID}/retry-failed', {
          params: { path: { requestID: requestId }, header: { 'Idempotency-Key': key } },
          body: { operations },
        }),
      )
    },
    onSuccess: () =>
      Promise.all([
        cache.invalidateQueries({ queryKey: generationKeys.all }),
        cache.invalidateQueries({ queryKey: courseKeys.all }),
      ]),
    onError: (error) => {
      if (error instanceof ApiError && error.status === 409)
        void cache.invalidateQueries({ queryKey: generationKeys.tracking(requestId) })
    },
  })
}
