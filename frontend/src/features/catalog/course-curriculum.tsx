import { Link } from '@tanstack/react-router'
import { BookOpen, Clock3, FileText, LoaderCircle, WandSparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Course, Module } from '@/shared/api/types'
import { formatDuration } from '@/shared/lib/format'
import { Button } from '@/shared/ui/button'
import { useGenerateModuleContent } from './api'
import { useRequestJobs } from '@/features/generation/jobs'
import { contentJobState } from '@/features/generation/job-state'
import { PartialGenerationNotice } from '@/features/generation/partial-generation-notice'
import { ApiErrorNotice } from '@/shared/ui/api-error-notice'

function ModuleBlock({ course, module }: { course: Course; module: Module }) {
  const { t, i18n } = useTranslation()
  const mutation = useGenerateModuleContent(course.id, module.id)
  const jobs = useRequestJobs(course.requestId)
  const state = contentJobState(jobs.data ?? [], [module.id, ...module.lessons.map((lesson) => lesson.id)])
  const missing = module.lessons.some((lesson) => !lesson.hasContent)
  const pending = mutation.isPending || state.active
  return (
    <section className="studio-panel p-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="font-mono text-xs font-bold text-primary">
            MODULE {String(module.order).padStart(2, '0')}
          </p>
          <h3 className="mt-1 text-lg font-bold">{module.title}</h3>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-muted-foreground">{module.description}</p>
        </div>
        {missing ? (
          <Button
            variant="secondary"
            size="sm"
            disabled={pending || jobs.isPending || jobs.isError || state.failed}
            onClick={() => mutation.mutate()}
          >
            {pending ? <LoaderCircle className="size-4 animate-spin" /> : <WandSparkles className="size-4" />}
            {pending ? t('course.generatingModule') : t('course.generateModule')}
          </Button>
        ) : null}
      </div>
      <ApiErrorNotice error={mutation.error ?? jobs.error} onRetry={() => void jobs.refetch()} />
      {missing && state.failed ? <PartialGenerationNotice requestId={course.requestId} /> : null}
      <ol className="mt-4 divide-y divide-border border-t border-border">
        {module.lessons.map((lesson) => (
          <li key={lesson.id}>
            <Link
              to="/courses/$courseId/lessons/$lessonId"
              params={{ courseId: course.id, lessonId: lesson.id }}
              className="flex items-center gap-3 rounded-lg px-2 py-4 text-sm transition-colors hover:bg-muted"
            >
              <span
                className={`grid size-8 shrink-0 place-items-center rounded-md ${lesson.hasContent ? 'bg-success-soft text-primary' : 'bg-muted text-muted-foreground'}`}
              >
                {lesson.hasContent ? <BookOpen className="size-4" /> : <FileText className="size-4" />}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block font-semibold leading-6">
                  {lesson.order}. {lesson.title}
                </span>
                <span className="mt-0.5 block text-xs text-muted-foreground">
                  {t(`lesson.type.${lesson.type}`)} ·{' '}
                  {lesson.hasContent ? t('design.ready') : t('design.planned')}
                </span>
              </span>
              <span className="flex shrink-0 items-center gap-1 text-xs text-muted-foreground">
                <Clock3 className="size-3.5" />
                {formatDuration(lesson.estimatedDurationMinutes, i18n.language)}
              </span>
            </Link>
          </li>
        ))}
      </ol>
    </section>
  )
}

export function CourseCurriculum({ course }: { course: Course }) {
  return (
    <div className="space-y-4">
      {course.modules.map((module) => (
        <ModuleBlock key={module.id} course={course} module={module} />
      ))}
    </div>
  )
}
