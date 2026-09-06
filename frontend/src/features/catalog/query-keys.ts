import type { CatalogSearch } from './schemas'

export const courseKeys = {
  all: ['courses'] as const,
  lists: () => [...courseKeys.all, 'list'] as const,
  list: (filters: CatalogSearch & { pageSize: number }) => [...courseKeys.lists(), filters] as const,
  detail: (courseId: string) => [...courseKeys.all, 'detail', courseId] as const,
  lesson: (lessonId: string) => [...courseKeys.all, 'lesson', lessonId] as const,
  moduleLessons: (moduleId: string) => [...courseKeys.all, 'module-lessons', moduleId] as const,
  solutions: (lessonId: string) => [...courseKeys.lesson(lessonId), 'solutions'] as const,
}
