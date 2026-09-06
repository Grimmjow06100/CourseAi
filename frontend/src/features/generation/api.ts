import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useApiClient } from '@/shared/api/context'
import { courseKeys } from '@/features/catalog/query-keys'
import { ensureApiSuccess, unwrapApiResult } from '@/shared/api/errors'
import type { GenerationPage, GenerationStarted, GenerationStatus, PipelineStatus } from '@/shared/api/types'
import type { ClarificationFormValues } from './schemas'
import { generationKeys } from './query-keys'

export function useGenerationList(filters: { status?: PipelineStatus; page: number; pageSize?: number }) {
  const client = useApiClient()
  const pageSize = filters.pageSize ?? 20
  return useQuery({
    queryKey: generationKeys.list({ ...filters, pageSize }),
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
    queryFn: async ({ signal }) => {
      const status = unwrapApiResult<GenerationStatus>(
        await client.GET('/api/generations/{requestID}/status', {
          params: { path: { requestID: requestId } },
          signal,
        }),
      )
      const previous = cache.getQueryData<GenerationStatus>(generationKeys.detail(requestId))
      if (status.pipelineStatus !== previous?.pipelineStatus) {
        void cache.invalidateQueries({ queryKey: courseKeys.all })
        void cache.invalidateQueries({ queryKey: generationKeys.lists() })
      }
      return status
    },
    refetchInterval(query) {
      const status = query.state.data?.pipelineStatus
      return !query.state.error && (status === 'queued' || status === 'running') ? 2_000 : false
    },
  })
}

export function useStartGeneration() {
  const client = useApiClient()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ prompt, idempotencyKey }: { prompt: string; idempotencyKey: string }) =>
      unwrapApiResult<GenerationStarted>(
        await client.POST('/api/generations', {
          params: { header: { 'Idempotency-Key': idempotencyKey } },
          body: { prompt },
        }),
      ),
    onSuccess: async ({ requestId }) => {
      await queryClient.invalidateQueries({ queryKey: generationKeys.lists() })
      await navigate({ to: '/generations/$requestId', params: { requestId } })
    },
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
      await queryClient.invalidateQueries({ queryKey: generationKeys.detail(requestId) })
      await queryClient.invalidateQueries({ queryKey: generationKeys.lists() })
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
      queryClient.removeQueries({ queryKey: generationKeys.jobs(requestId) })
      await queryClient.invalidateQueries({ queryKey: generationKeys.lists() })
      await queryClient.invalidateQueries({ queryKey: courseKeys.all })
    },
  })
}
