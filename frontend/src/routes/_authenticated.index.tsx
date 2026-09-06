import { createFileRoute, Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCourses } from '@/features/catalog/api'
import { CourseSummaryLine } from '@/features/catalog/course-card'
import { useGenerationList } from '@/features/generation/api'
import { GenerationForm } from '@/features/generation/generation-form'
import { GenerationList } from '@/features/generation/generation-list'
import { Button } from '@/shared/ui/button'
import { ApiErrorNotice } from '@/shared/ui/api-error-notice'
import { LoadingState } from '@/shared/ui/feedback'
import { SectionHeader } from '@/shared/ui/page'

function Dashboard() {
  const { t } = useTranslation()
  const generations = useGenerationList({ page: 1, pageSize: 5 })
  const courses = useCourses({ page: 1, orderBy: 'created_at', orderDirection: 'desc' }, 5)
  return (
    <div>
      <section className="border-b border-border pb-10">
        <p className="text-xs font-extrabold uppercase text-primary">{t('dashboard.eyebrow')}</p>
        <h1 className="mt-3 max-w-3xl text-3xl font-bold leading-tight sm:text-4xl">
          {t('dashboard.title')}
        </h1>
        <p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground sm:text-base">
          {t('dashboard.subtitle')}
        </p>
        <div className="mt-7 max-w-4xl rounded-lg border border-border bg-surface p-5 sm:p-6">
          <GenerationForm compact />
        </div>
      </section>
      <div className="grid gap-10 py-9 xl:grid-cols-2">
        <section>
          <SectionHeader
            title={t('dashboard.recentGenerations')}
            action={
              <Button asChild variant="ghost" size="sm">
                <Link to="/generations" search={{ page: 1 }}>
                  {t('dashboard.viewAll')}
                  <ArrowRight className="size-4" />
                </Link>
              </Button>
            }
          />
          {generations.isLoading ? (
            <LoadingState label={t('common.loading')} />
          ) : generations.error ? (
            <ApiErrorNotice error={generations.error} onRetry={() => void generations.refetch()} />
          ) : generations.data?.items.length ? (
            <GenerationList generations={generations.data.items} limit={5} />
          ) : (
            <p className="border-y border-border py-8 text-sm text-muted-foreground">
              {t('dashboard.emptyGenerations')}
            </p>
          )}
        </section>
        <section>
          <SectionHeader
            title={t('dashboard.recentCourses')}
            action={
              <Button asChild variant="ghost" size="sm">
                <Link to="/courses" search={{ page: 1, orderBy: 'created_at', orderDirection: 'desc' }}>
                  {t('dashboard.viewAll')}
                  <ArrowRight className="size-4" />
                </Link>
              </Button>
            }
          />
          {courses.isLoading ? (
            <LoadingState label={t('common.loading')} />
          ) : courses.error ? (
            <ApiErrorNotice error={courses.error} onRetry={() => void courses.refetch()} />
          ) : courses.data?.items.length ? (
            <div>
              {courses.data.items.map((course) => (
                <CourseSummaryLine key={course.id} course={course} />
              ))}
            </div>
          ) : (
            <p className="border-y border-border py-8 text-sm text-muted-foreground">
              {t('dashboard.emptyCourses')}
            </p>
          )}
        </section>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_authenticated/')({ component: Dashboard })
