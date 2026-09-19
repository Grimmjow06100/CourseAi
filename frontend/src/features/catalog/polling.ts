import { shouldPollQuery } from '@/shared/api/polling'
import type { CourseSummary } from '@/shared/api/types'

export function shouldPollCourses(
  courses: Pick<CourseSummary, 'status'>[],
  query: Parameters<typeof shouldPollQuery>[0],
) {
  return shouldPollQuery(
    query,
    courses.some(({ status }) => status !== 'completed' && status !== 'failed' && status !== 'partial'),
    // Older failed records can still be repaired by backend reconciliation.
    courses.some(({ status }) => status === 'failed'),
  )
}
