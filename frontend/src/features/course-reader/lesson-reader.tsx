import * as Dialog from '@radix-ui/react-dialog'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, ArrowRight, BookOpen, Clock3, List, LoaderCircle, WandSparkles, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Course, Lesson } from '@/shared/api/types'
import { cn } from '@/shared/lib/cn'
import { formatDuration } from '@/shared/lib/format'
import { Button } from '@/shared/ui/button'
import { Markdown } from '@/shared/ui/markdown'
import { useGenerateLessonContent, useLesson, useLessonSolutions } from '@/features/catalog/api'
import { useRequestJobs } from '@/features/generation/jobs'
import { contentJobState } from '@/features/generation/job-state'
import { PartialGenerationNotice } from '@/features/generation/partial-generation-notice'
import { ApiErrorNotice } from '@/shared/ui/api-error-notice'
import { LessonActivities } from './lesson-activities'

function CurriculumNav({
  course,
  lessonId,
  onNavigate,
}: {
  course: Course
  lessonId: string
  onNavigate?: () => void
}) {
  const { t } = useTranslation()
  return (
    <nav aria-label={t('course.curriculum')} className="space-y-6">
      {course.modules.map((module) => (
        <section key={module.id}>
          <h2 className="mb-2 px-2 text-xs font-extrabold uppercase text-muted-foreground">
            {module.order}. {module.title}
          </h2>
          <div className="space-y-1">
            {module.lessons.map((lesson) => (
              <Link
                key={lesson.id}
                to="/courses/$courseId/lessons/$lessonId"
                params={{ courseId: course.id, lessonId: lesson.id }}
                onClick={onNavigate}
                className={cn(
                  'flex items-start gap-2 rounded-md px-2 py-2 text-sm text-muted-foreground hover:bg-muted hover:text-foreground',
                  lesson.id === lessonId && 'bg-success-soft font-semibold text-primary',
                )}
              >
                <BookOpen className="mt-0.5 size-4 shrink-0" />
                <span>
                  {lesson.order}. {lesson.title}
                </span>
              </Link>
            ))}
          </div>
        </section>
      ))}
    </nav>
  )
}

export function LessonReader({ course, initialLesson }: { course: Course; initialLesson: Lesson }) {
  const { t, i18n } = useTranslation()
  const [solutionsEnabled, setSolutionsEnabled] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const lessonQuery = useLesson(initialLesson.id)
  const lesson = lessonQuery.data ?? initialLesson
  const generation = useGenerateLessonContent(course.id, lesson.id)
  const solutions = useLessonSolutions(lesson.id, solutionsEnabled)
  const flatLessons = useMemo(() => course.modules.flatMap((module) => module.lessons), [course.modules])
  const index = flatLessons.findIndex((item) => item.id === lesson.id)
  const previous = index > 0 ? flatLessons[index - 1] : undefined
  const next = index >= 0 ? flatLessons[index + 1] : undefined
  const jobs = useRequestJobs(course.requestId)
  const jobState = contentJobState(jobs.data ?? [], [lesson.id])
  const pending = generation.isPending || jobState.active

  return (
    <div className="-mx-4 -my-6 min-h-[calc(100vh-4rem)] sm:-mx-6 lg:-mx-8 lg:-my-8 lg:grid lg:grid-cols-[280px_minmax(0,1fr)]">
      <aside className="hidden border-r border-border bg-sidebar p-5 lg:block">
        <Link
          to="/courses/$courseId"
          params={{ courseId: course.id }}
          className="mb-7 flex items-center gap-2 text-sm font-bold text-primary"
        >
          <ArrowLeft className="size-4" />
          {course.title}
        </Link>
        <CurriculumNav course={course} lessonId={lesson.id} />
      </aside>
      <article className="min-w-0 px-4 py-6 sm:px-8 lg:px-12 lg:py-10 xl:px-16">
        <div className="mb-6 flex items-center justify-between lg:hidden">
          <Button asChild variant="ghost" size="sm">
            <Link to="/courses/$courseId" params={{ courseId: course.id }}>
              <ArrowLeft className="size-4" />
              {t('common.back')}
            </Link>
          </Button>
          <Dialog.Root open={drawerOpen} onOpenChange={setDrawerOpen}>
            <Dialog.Trigger asChild>
              <Button variant="secondary" size="sm">
                <List className="size-4" />
                {t('course.curriculum')}
              </Button>
            </Dialog.Trigger>
            <Dialog.Portal>
              <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
              <Dialog.Content
                aria-describedby={undefined}
                className="fixed inset-y-0 right-0 z-50 w-[min(88vw,360px)] overflow-y-auto border-l border-border bg-sidebar p-5"
              >
                <div className="mb-6 flex justify-between">
                  <Dialog.Title className="font-bold">{t('course.curriculum')}</Dialog.Title>
                  <Dialog.Close asChild>
                    <Button variant="icon" aria-label={t('common.close')}>
                      <X className="size-4" />
                    </Button>
                  </Dialog.Close>
                </div>
                <CurriculumNav course={course} lessonId={lesson.id} onNavigate={() => setDrawerOpen(false)} />
              </Dialog.Content>
            </Dialog.Portal>
          </Dialog.Root>
        </div>
        <div className="mx-auto max-w-4xl">
          <div className="border-b border-border pb-6">
            <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
              <span className="font-mono uppercase text-primary">{t(`lesson.type.${lesson.type}`)}</span>
              <span className="flex items-center gap-1">
                <Clock3 className="size-3.5" />
                {formatDuration(lesson.estimatedDurationMinutes, i18n.language)}
              </span>
            </div>
            <h1 className="mt-3 text-2xl font-bold leading-tight sm:text-3xl">{lesson.title}</h1>
            <p className="mt-3 text-sm leading-6 text-muted-foreground">
              <strong className="text-foreground">{t('lesson.objective')}:</strong> {lesson.learningGoal}
            </p>
          </div>
          {lesson.hasContent && lesson.contentMarkdown ? (
            <div className="py-8">
              <Markdown>{lesson.contentMarkdown}</Markdown>
              <ApiErrorNotice
                error={solutionsEnabled ? solutions.error : null}
                onRetry={() => void solutions.refetch()}
              />
              <LessonActivities
                exercises={lesson.exercises}
                quizzes={lesson.quizzes}
                solutions={solutionsEnabled ? solutions.data : undefined}
                onReveal={() => setSolutionsEnabled(true)}
              />
            </div>
          ) : (
            <div className="my-8 border-y border-dashed border-border py-12 text-center">
              <BookOpen className="mx-auto size-8 text-muted-foreground" />
              <p className="mt-4 text-sm text-muted-foreground">{t('lesson.unavailable')}</p>
              <Button
                className="mt-5"
                disabled={pending || jobs.isPending || jobs.isError || jobState.failed}
                onClick={() => generation.mutate()}
              >
                {pending ? (
                  <LoaderCircle className="size-4 animate-spin" />
                ) : (
                  <WandSparkles className="size-4" />
                )}
                {pending ? t('lesson.generating') : t('lesson.generate')}
              </Button>
            </div>
          )}
          <ApiErrorNotice
            error={generation.error ?? lessonQuery.error ?? jobs.error}
            onRetry={() => void jobs.refetch()}
          />
          {!lesson.hasContent && jobState.failed ? (
            <PartialGenerationNotice requestId={course.requestId} />
          ) : null}
          <nav className="mt-10 grid grid-cols-2 gap-3 border-t border-border pt-6">
            {previous ? (
              <Button asChild variant="secondary" className="justify-start">
                <Link
                  to="/courses/$courseId/lessons/$lessonId"
                  params={{ courseId: course.id, lessonId: previous.id }}
                >
                  <ArrowLeft className="size-4" />
                  <span className="truncate">{previous.title}</span>
                </Link>
              </Button>
            ) : (
              <span />
            )}
            {next ? (
              <Button asChild variant="secondary" className="justify-end">
                <Link
                  to="/courses/$courseId/lessons/$lessonId"
                  params={{ courseId: course.id, lessonId: next.id }}
                >
                  <span className="truncate">{next.title}</span>
                  <ArrowRight className="size-4" />
                </Link>
              </Button>
            ) : null}
          </nav>
        </div>
      </article>
    </div>
  )
}
