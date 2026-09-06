import { getErrorMessage } from '@/shared/api/error-message'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useCourse, useLesson } from '@/features/catalog/api'
import { LessonReader } from '@/features/course-reader/lesson-reader'
import { ErrorState, LoadingState } from '@/shared/ui/feedback'

function LessonPage() {
  const { courseId, lessonId } = Route.useParams()
  const { t } = useTranslation()
  const course = useCourse(courseId)
  const lesson = useLesson(lessonId)
  if (course.isLoading || lesson.isLoading) return <LoadingState label={t('common.loading')} />
  if (!course.data || !lesson.data)
    return (
      <ErrorState
        title={t('errors.generic')}
        description={getErrorMessage(course.error ?? lesson.error, t)}
        onRetry={() => {
          void course.refetch()
          void lesson.refetch()
        }}
      />
    )
  return <LessonReader key={lessonId} course={course.data} initialLesson={lesson.data} />
}
export const Route = createFileRoute('/_authenticated/courses/$courseId/lessons/$lessonId')({
  component: LessonPage,
})
