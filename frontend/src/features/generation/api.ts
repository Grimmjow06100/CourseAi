import { courseKeys, generationKeys } from '@/shared/api/query-keys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useApiClient } from '@/shared/api/context'
import { ensureApiSuccess, unwrapApiResult } from '@/shared/api/errors'
import type { GenerationPage, GenerationStarted, GenerationStatus, PipelineStatus } from '@/shared/api/types'
import type { ClarificationFormValues } from './schemas'
import { shouldPollGenerations } from './presentation'

export function useGenerationList(filters: { status?: PipelineStatus; page: number; pageSize?: number }) {
  const client = useApiClient()
  const pageSize = filters.pageSize ?? 20
  return useQuery<GenerationPage>({
    queryKey: generationKeys.list({ ...filters, pageSize }),
    refetchOnWindowFocus: 'always',
    refetchOnReconnect: 'always',
    refetchInterval: (query) =>
      !query.state.error && shouldPollGenerations(query.state.data?.items ?? [], query) ? 5_000 : false,
    queryFn: async ({ signal }) =>
      unwrapApiResult<GenerationPage>(
        await client.GET('/api/generations', {
          signal,
          params: {
            query: { ...(filters.status ? { status: filters.status } : {}), page: filters.page, pageSize },
          },
        }),
      ),
  })
}

export function useGenerationStatus(requestId: string) {
  const client = useApiClient()
  const cache = useQueryClient()
  return useQuery({
    queryKey: generationKeys.detail(requestId),
    staleTime: 0,
    refetchOnWindowFocus: 'always',
    refetchOnReconnect: 'always',
    queryFn: async ({ signal }) => {
      const status = unwrapApiResult<GenerationStatus>(
        await client.GET('/api/generations/{requestID}/status', {
          params: { path: { requestID: requestId } },
          signal,
        }),
      )
      const previous = cache.getQueryData<GenerationStatus>(generationKeys.detail(requestId))
      if (
        status.pipelineStatus !== previous?.pipelineStatus ||
        status.courseStatus !== previous.courseStatus ||
        status.contentComplete !== previous.contentComplete ||
        status.generationAttempt !== previous.generationAttempt
      ) {
        void cache.invalidateQueries({ queryKey: courseKeys.all })
        void cache.invalidateQueries({ queryKey: generationKeys.lists() })
      }
      return status
    },
    refetchInterval(query) {
      return !query.state.error && shouldPollGenerations(query.state.data ? [query.state.data] : [], query)
        ? 2_000
        : false
    },
  })
}

export function useStartGeneration() {
  const client = useApiClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ prompt, idempotencyKey }: { prompt: string; idempotencyKey: string }) =>
      unwrapApiResult<GenerationStarted>(
        await client.POST('/api/generations', {
          params: { header: { 'Idempotency-Key': idempotencyKey } },
          body: { prompt },
        }),
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: generationKeys.lists() }),
  })
}

export function useSubmitClarifications(requestId: string) {
  const client = useApiClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (values: ClarificationFormValues) =>
      unwrapApiResult<GenerationStarted>(
        await client.POST('/api/generations/{requestID}/clarifications', {
          params: { path: { requestID: requestId } },
          body: {
            title: values.title,
            synopsis: values.synopsis,
            language: values.language,
            answers: Object.entries(values.answers).map(([questionId, selectedValues]) => ({
              questionId: questionId as 'goals' | 'currentLevel' | 'targetLevel',
              selectedValues,
            })),
          },
        }),
      ),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: generationKeys.tracking(requestId) }),
        queryClient.invalidateQueries({ queryKey: generationKeys.detail(requestId) }),
        queryClient.invalidateQueries({ queryKey: generationKeys.lists() }),
      ])
    },
  })
}

export function useRetryGeneration(requestId: string) {
  const client = useApiClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () =>
      unwrapApiResult<GenerationStarted>(
        await client.POST('/api/generations/{requestID}/retry', {
          params: { path: { requestID: requestId } },
        }),
      ),
    onSuccess: async () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: generationKeys.all }),
        queryClient.invalidateQueries({ queryKey: courseKeys.all }),
      ]),
  })
}

export function useDeleteGeneration() {
  const client = useApiClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (requestId: string) =>
      ensureApiSuccess(
        await client.DELETE('/api/generations/{requestID}', { params: { path: { requestID: requestId } } }),
      ),
    onSuccess: async (_, requestId) => {
      queryClient.removeQueries({ queryKey: generationKeys.detail(requestId) })
      queryClient.removeQueries({ queryKey: generationKeys.tracking(requestId) })
      queryClient.removeQueries({ queryKey: generationKeys.events(requestId) })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: generationKeys.lists() }),
        queryClient.invalidateQueries({ queryKey: courseKeys.all }),
      ])
    },
  })
}
