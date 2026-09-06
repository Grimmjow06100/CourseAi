import { getErrorMessage } from '@/shared/api/error-message'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useCourse } from '@/features/catalog/api'
import { CourseOverview } from '@/features/catalog/course-overview'
import { ErrorState, LoadingState } from '@/shared/ui/feedback'

function CoursePage() {
  const { courseId } = Route.useParams()
  const { t } = useTranslation()
  const query = useCourse(courseId)
  if (query.isLoading) return <LoadingState label={t('common.loading')} />
  if (query.error || !query.data)
    return (
      <ErrorState
        title={t('errors.generic')}
        description={getErrorMessage(query.error, t)}
        onRetry={() => void query.refetch()}
      />
    )
  return <CourseOverview course={query.data} />
}
export const Route = createFileRoute('/_authenticated/courses/$courseId/')({ component: CoursePage })
