import { createFileRoute, Link } from '@tanstack/react-router'
import { ArrowRight, BookOpen, FolderClock, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCourses } from '@/features/catalog/api'
import { CourseSummaryLine } from '@/features/catalog/course-card'
import { useGenerationList } from '@/features/generation/api'
import { GenerationForm } from '@/features/generation/generation-form'
import { GenerationList } from '@/features/generation/generation-list'
import { LearningPath } from '@/features/generation/learning-path'
import { Button } from '@/shared/ui/button'
import { ApiErrorNotice } from '@/shared/ui/api-error-notice'
import { EmptyState, LoadingState } from '@/shared/ui/feedback'
import { SectionHeader } from '@/shared/ui/page'

function Dashboard() {
  const { t } = useTranslation()
  const generations = useGenerationList({ page: 1, pageSize: 5 })
  const courses = useCourses({ page: 1, orderBy: 'created_at', orderDirection: 'desc' }, 5)
  return (
    <div>
      <section className="mb-7">
        <p className="text-[10px] font-extrabold tracking-[.18em] text-primary">{t('design.studio')}</p>
        <h1 className="mt-3 max-w-3xl text-3xl font-extrabold leading-tight tracking-tight sm:text-4xl xl:text-[42px]">
          {t('dashboard.title')}
        </h1>
        <p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground sm:text-base">
          {t('dashboard.subtitle')}
        </p>
      </section>
      <div className="grid grid-cols-1 items-stretch gap-5 xl:grid-cols-[minmax(0,1.7fr)_minmax(300px,1fr)]">
        <div className="studio-panel p-5 sm:p-7">
          <GenerationForm compact />
        </div>
        <LearningPath />
      </div>
      <div className="mt-5 flex flex-wrap gap-x-6 gap-y-2 text-xs text-muted-foreground">
        {courses.data && !courses.isError ? (
          <span className="flex items-center gap-2">
            <BookOpen className="size-4 text-primary" />
            {t('design.libraryCount', { count: courses.data.totalItems })}
          </span>
        ) : null}
        {generations.data && !generations.isError ? (
          <span className="flex items-center gap-2">
            <FolderClock className="size-4 text-primary" />
            {t('design.generationCount', { count: generations.data.totalItems })}
          </span>
        ) : null}
      </div>
      <div className="grid grid-cols-1 gap-8 py-9 xl:grid-cols-2">
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
            <EmptyState
              title={t('dashboard.emptyGenerations')}
              description={t('design.emptyGenerationsText')}
            />
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
            <div className="studio-panel px-5">
              {courses.data.items.map((course) => (
                <CourseSummaryLine key={course.id} course={course} />
              ))}
            </div>
          ) : (
            <EmptyState
              title={t('dashboard.emptyCourses')}
              description={t('design.emptyCoursesText')}
              action={
                <Button asChild variant="secondary">
                  <Link to="/generate">
                    <Plus className="size-4" />
                    {t('nav.generate')}
                  </Link>
                </Button>
              }
            />
          )}
        </section>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_authenticated/')({ component: Dashboard })
