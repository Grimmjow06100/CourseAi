import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useApiClient } from '@/shared/api/context'
import { ensureApiSuccess, unwrapApiResult } from '@/shared/api/errors'
import type { Course, CoursePage, GenerationStarted, Lesson, LessonSolutions } from '@/shared/api/types'
import { generationKeys } from '@/features/generation/query-keys'
import { shouldPollQuery } from '@/shared/api/polling'
import { courseKeys } from './query-keys'
import type { CatalogSearch } from './schemas'

export function useCourses(filters: CatalogSearch, pageSize = 12) {
  const client = useApiClient()
  const query = {
    page: filters.page,
    pageSize,
    orderBy: filters.orderBy,
    orderDirection: filters.orderDirection,
    ...(filters.search ? { search: filters.search } : {}),
    ...(filters.status ? { status: filters.status } : {}),
    ...(filters.language ? { language: filters.language } : {}),
  }
  return useQuery<CoursePage>({
    queryKey: courseKeys.list({ ...filters, pageSize }),
    refetchOnWindowFocus: 'always',
    refetchOnReconnect: 'always',
    refetchInterval: (query) => {
      const courses = query.state.data?.items ?? []
      return !query.state.error &&
        shouldPollQuery(
          query,
          courses.some((course) => course.status !== 'completed' && course.status !== 'failed'),
          courses.some((course) => course.status === 'failed'),
        )
        ? 5_000
        : false
    },
    queryFn: async ({ signal }) =>
      unwrapApiResult<CoursePage>(await client.GET('/api/courses', { params: { query }, signal })),
  })
}

export function useCourse(courseId: string) {
  const client = useApiClient()
  return useQuery({
    queryKey: courseKeys.detail(courseId),
    refetchOnWindowFocus: 'always',
    queryFn: async ({ signal }) =>
      unwrapApiResult<Course>(
        await client.GET('/api/courses/{courseID}', { params: { path: { courseID: courseId } }, signal }),
      ),
  })
}

export function useLesson(lessonId: string) {
  const client = useApiClient()
  return useQuery({
    queryKey: courseKeys.lesson(lessonId),
    queryFn: async ({ signal }) =>
      unwrapApiResult<Lesson>(
        await client.GET('/api/lessons/{lessonID}', { params: { path: { lessonID: lessonId } }, signal }),
      ),
  })
}

export function useLessonSolutions(lessonId: string, enabled: boolean) {
  const client = useApiClient()
  return useQuery({
    queryKey: courseKeys.solutions(lessonId),
    enabled,
    queryFn: async ({ signal }) =>
      unwrapApiResult<LessonSolutions>(
        await client.GET('/api/lessons/{lessonID}/solutions', {
          params: { path: { lessonID: lessonId } },
          signal,
        }),
      ),
    staleTime: Number.POSITIVE_INFINITY,
  })
}

export function useGenerateLessonContent(courseId: string, lessonId: string) {
  const client = useApiClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () =>
      unwrapApiResult<GenerationStarted>(
        await client.POST('/api/generations/lessons/{lessonID}/content', {
          params: { path: { lessonID: lessonId } },
        }),
      ),
    onSuccess: async ({ requestId }) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: generationKeys.jobs(requestId) }),
        queryClient.invalidateQueries({ queryKey: generationKeys.detail(requestId) }),
        queryClient.invalidateQueries({ queryKey: generationKeys.lists() }),
        queryClient.invalidateQueries({ queryKey: courseKeys.lesson(lessonId) }),
        queryClient.invalidateQueries({ queryKey: courseKeys.detail(courseId) }),
      ]),
  })
}

export function useGenerateModuleContent(courseId: string, moduleId: string) {
  const client = useApiClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () =>
      unwrapApiResult<GenerationStarted>(
        await client.POST('/api/generations/modules/{moduleID}/contents', {
          params: { path: { moduleID: moduleId } },
        }),
      ),
    onSuccess: async ({ requestId }) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: courseKeys.detail(courseId) }),
        queryClient.invalidateQueries({ queryKey: generationKeys.jobs(requestId) }),
        queryClient.invalidateQueries({ queryKey: generationKeys.detail(requestId) }),
        queryClient.invalidateQueries({ queryKey: generationKeys.lists() }),
      ]),
  })
}

export function useDeleteCourse() {
  const client = useApiClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (courseId: string) =>
      ensureApiSuccess(
        await client.DELETE('/api/courses/{courseID}', { params: { path: { courseID: courseId } } }),
      ),
    onSuccess: async () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: courseKeys.all }),
        queryClient.invalidateQueries({ queryKey: generationKeys.all }),
      ]),
  })
}
