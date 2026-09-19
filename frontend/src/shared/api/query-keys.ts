import type { paths } from './schema.gen'
import type { PipelineStatus } from './types'

type CourseListQuery = NonNullable<paths['/api/courses']['get']['parameters']['query']>
type CourseListFilters = { [Key in keyof CourseListQuery]?: CourseListQuery[Key] | undefined }

export const courseKeys = {
  all: ['courses'] as const,
  lists: () => [...courseKeys.all, 'list'] as const,
  list: (filters: CourseListFilters) => [...courseKeys.lists(), filters] as const,
  detail: (courseId: string) => [...courseKeys.all, 'detail', courseId] as const,
  lesson: (lessonId: string) => [...courseKeys.all, 'lesson', lessonId] as const,
  solutions: (lessonId: string) => [...courseKeys.lesson(lessonId), 'solutions'] as const,
}

export const generationKeys = {
  all: ['generations'] as const,
  lists: () => [...generationKeys.all, 'list'] as const,
  list: (filters: { status?: PipelineStatus; page: number; pageSize: number }) =>
    [...generationKeys.lists(), filters] as const,
  detail: (requestId: string) => [...generationKeys.all, 'detail', requestId] as const,
  tracking: (requestId: string) => [...generationKeys.all, 'tracking', requestId] as const,
  events: (requestId: string) => [...generationKeys.all, 'events', requestId] as const,
}
