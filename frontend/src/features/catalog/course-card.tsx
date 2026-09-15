import { Link } from '@tanstack/react-router'
import { ArrowRight, BookOpen, Clock3, Layers3, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { CourseSummary } from '@/shared/api/types'
import { formatDate } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { useDeleteCourse } from './api'

export function CourseCard({ course }: { course: CourseSummary }) {
  const { t, i18n } = useTranslation()
  const remove = useDeleteCourse()
  return (
    <article className="studio-panel group flex min-w-0 flex-col overflow-hidden transition-colors hover:border-primary/50">
      <div
        className="course-cover flex h-24 items-center justify-between border-b border-border px-5"
        aria-hidden="true"
      >
        <span className="grid size-12 place-items-center rounded-xl border border-border bg-surface text-primary">
          <BookOpen className="size-6" />
        </span>
        <span className="rounded-lg bg-surface px-2.5 py-1.5 font-mono text-xs font-bold uppercase text-primary">
          {course.language}
        </span>
      </div>
      <div className="flex flex-1 flex-col p-5">
        <div className="mb-5 flex items-center justify-between gap-3">
          <Badge
            tone={course.status === 'completed' ? 'success' : course.status === 'failed' ? 'danger' : 'info'}
          >
            {t(`course.status.${course.status}`)}
          </Badge>
          <span className="text-xs text-muted-foreground">{t(`design.${course.targetLevel}`)}</span>
        </div>
        <h2 className="text-lg font-extrabold leading-snug">
          <Link to="/courses/$courseId" params={{ courseId: course.id }} className="hover:text-primary">
            {course.title}
          </Link>
        </h2>
        <p className="mt-2 mb-6 line-clamp-3 text-sm leading-6 text-muted-foreground">{course.synopsis}</p>
        <div className="mt-auto flex flex-wrap items-center justify-between gap-2 border-t border-border pt-4 text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <Clock3 className="size-3.5" />
            {formatDate(course.updatedAt, i18n.language)}
          </span>
          <div className="flex items-center gap-1">
            <ConfirmDialog
              trigger={
                <Button variant="ghost" className="min-h-10 px-2" aria-label={t('common.delete')}>
                  <Trash2 className="size-4" />
                </Button>
              }
              title={t('catalog.deleteTitle')}
              description={t('catalog.deleteDescription')}
              confirmLabel={t('common.delete')}
              cancelLabel={t('common.cancel')}
              pending={remove.isPending}
              onConfirm={() => remove.mutateAsync(course.id)}
            />
            <Button asChild variant="ghost" size="sm">
              <Link to="/courses/$courseId" params={{ courseId: course.id }}>
                {t('common.open')}
                <ArrowRight className="size-4" />
              </Link>
            </Button>
          </div>
        </div>
      </div>
    </article>
  )
}

export function CourseSummaryLine({ course }: { course: CourseSummary }) {
  const { t } = useTranslation()
  return (
    <article className="flex items-center gap-4 border-b border-border py-4 last:border-0">
      <span className="grid size-10 shrink-0 place-items-center rounded-md bg-success-soft text-primary">
        <Layers3 className="size-5" />
      </span>
      <div className="min-w-0 flex-1">
        <h3 className="line-clamp-2 font-bold">{course.title}</h3>
        <p className="truncate text-sm text-muted-foreground">{course.synopsis}</p>
      </div>
      <Button asChild variant="ghost" size="sm">
        <Link to="/courses/$courseId" params={{ courseId: course.id }}>
          {t('common.open')}
          <ArrowRight className="size-4" />
        </Link>
      </Button>
    </article>
  )
}
