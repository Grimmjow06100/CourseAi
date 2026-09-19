import { Link } from '@tanstack/react-router'
import { ArrowLeft, ArrowRight, BookOpen, Clock3, List } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Course, Lesson } from '@/shared/api/types'
import { cn } from '@/shared/lib/cn'
import { formatDuration } from '@/shared/lib/format'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { Markdown } from '@/shared/ui/markdown'
import { useLesson, useLessonSolutions } from '@/features/catalog/api'
import { useGenerationTracking } from '@/features/generation/tracking-api'
import { OperationStatus } from '@/features/generation/tracking-modules'
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
          <h2 className="mb-2 px-2 text-xs font-semibold text-muted-foreground">
            {module.order}. {module.title}
          </h2>
          <div className="space-y-1">
            {module.lessons.map((lesson) => (
              <Link
                key={lesson.id}
                to="/courses/$courseId/lessons/$lessonId"
                params={{ courseId: course.id, lessonId: lesson.id }}
                onClick={onNavigate}
                aria-current={lesson.id === lessonId ? 'page' : undefined}
                className={cn(
                  'flex min-h-11 items-start gap-2 rounded-md px-2 py-3 text-sm text-muted-foreground hover:bg-muted hover:text-foreground',
                  lesson.id === lessonId && 'bg-muted font-medium text-foreground',
                )}
              >
                <BookOpen className="mt-0.5 size-4 shrink-0" />
                <span className="min-w-0 break-words">
                  {lesson.order}. {lesson.title}
                  {!lesson.hasContent ? (
                    <span className="mt-1 block text-xs font-normal">{t('design.planned')}</span>
                  ) : null}
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
  const [curriculumOpen, setCurriculumOpen] = useState(true)
  const lessonQuery = useLesson(initialLesson.id)
  const lesson = lessonQuery.data ?? initialLesson
  const solutions = useLessonSolutions(lesson.id, solutionsEnabled)
  const flatLessons = course.modules.flatMap((module) => module.lessons)
  const index = flatLessons.findIndex((item) => item.id === lesson.id)
  const previous = index > 0 ? flatLessons[index - 1] : undefined
  const next = index >= 0 ? flatLessons[index + 1] : undefined
  const tracking = useGenerationTracking(course.requestId)
  const operation = tracking.data?.operations.find(
    (item) => item.kind === 'lesson_content' && item.targetId === lesson.id,
  )

  return (
    <div className={cn('grid gap-8', curriculumOpen && 'xl:grid-cols-[220px_minmax(0,1fr)]')}>
      <aside
        id="lesson-curriculum"
        className={cn(
          'sticky top-20 hidden h-[calc(100dvh-6rem)] overflow-y-auto border-r border-border pr-4',
          curriculumOpen && 'xl:block',
        )}
      >
        <Button
          asChild
          variant="ghost"
          className="mb-5 h-auto w-full justify-start whitespace-normal text-left"
        >
          <Link to="/courses/$courseId" params={{ courseId: course.id }}>
            <ArrowLeft className="size-4 shrink-0" />
            {course.title}
          </Link>
        </Button>
        <CurriculumNav course={course} lessonId={lesson.id} />
      </aside>
      <article className="min-w-0">
        <div className="mb-4 hidden xl:flex">
          <Button
            variant="ghost"
            size="sm"
            aria-expanded={curriculumOpen}
            aria-controls="lesson-curriculum"
            onClick={() => setCurriculumOpen((open) => !open)}
          >
            <List className="size-4" />
            {t(curriculumOpen ? 'course.hideCurriculum' : 'course.showCurriculum')}
          </Button>
        </div>
        <div className="mb-6 flex items-center justify-between gap-2 xl:hidden">
          <Button asChild variant="ghost" size="sm">
            <Link to="/courses/$courseId" params={{ courseId: course.id }}>
              <ArrowLeft className="size-4" />
              {t('common.back')}
            </Link>
          </Button>
          <Sheet open={drawerOpen} onOpenChange={setDrawerOpen}>
            <SheetTrigger asChild>
              <Button variant="outline" size="sm">
                <List className="size-4" />
                {t('course.curriculum')}
              </Button>
            </SheetTrigger>
            <SheetContent
              closeLabel={t('common.close')}
              aria-describedby={undefined}
              className="overflow-y-auto"
            >
              <SheetHeader>
                <SheetTitle>{t('course.curriculum')}</SheetTitle>
              </SheetHeader>
              <div className="px-4 pb-6">
                <CurriculumNav course={course} lessonId={lesson.id} onNavigate={() => setDrawerOpen(false)} />
              </div>
            </SheetContent>
          </Sheet>
        </div>
        <div className="mx-auto max-w-[75ch]">
          <header className="border-b border-border pb-6">
            <p className="mb-4 text-xs text-muted-foreground">
              {t('design.lessonPosition', { current: index + 1, total: flatLessons.length })}
            </p>
            <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
              <span>{t(`lesson.type.${lesson.type}`)}</span>
              <span className="flex items-center gap-1">
                <Clock3 className="size-3.5" />
                {formatDuration(lesson.estimatedDurationMinutes, i18n.language)}
              </span>
            </div>
            <h1 className="mt-3 break-words text-2xl font-semibold leading-tight sm:text-3xl">
              {lesson.title}
            </h1>
            <p className="mt-5 border-l-2 border-border pl-4 text-sm leading-6 text-muted-foreground">
              <strong className="font-medium text-foreground">{t('lesson.objective')} :</strong>{' '}
              {lesson.learningGoal}
            </p>
          </header>
          {lesson.hasContent ? (
            <div className="py-8">
              {lesson.contentMarkdown ? <Markdown>{lesson.contentMarkdown}</Markdown> : null}
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
            <Card className="my-8">
              <CardContent className="flex flex-col items-center gap-4 py-6 text-center">
                <BookOpen className="size-6 text-muted-foreground" />
                <p className="text-sm text-muted-foreground">{t('lesson.unavailable')}</p>
                {operation ? <OperationStatus status={operation.status} /> : null}
                <Button asChild variant="outline">
                  <Link to="/generations/$requestId" params={{ requestId: course.requestId }}>
                    {t('tracking.openTracking')}
                  </Link>
                </Button>
              </CardContent>
            </Card>
          )}
          <ApiErrorNotice
            error={lessonQuery.error ?? tracking.error}
            onRetry={() => {
              void lessonQuery.refetch()
              void tracking.refetch()
            }}
          />
          {!next ? (
            <div className="mt-8 border-t border-border pt-5">
              <p className="mb-3 text-sm text-muted-foreground">{t('design.endLesson')}</p>
              <Button asChild variant="outline">
                <Link to="/courses/$courseId" params={{ courseId: course.id }}>
                  {t('design.backCurriculum')}
                  <ArrowRight className="size-4" />
                </Link>
              </Button>
            </div>
          ) : null}
          <nav
            aria-label={t('course.curriculum')}
            className="mt-10 grid grid-cols-1 gap-3 border-t border-border pt-6 sm:grid-cols-2"
          >
            {previous ? (
              <Button asChild variant="outline" className="min-w-0 justify-start">
                <Link
                  to="/courses/$courseId/lessons/$lessonId"
                  params={{ courseId: course.id, lessonId: previous.id }}
                >
                  <ArrowLeft className="size-4 shrink-0" />
                  <span className="truncate">{previous.title}</span>
                </Link>
              </Button>
            ) : (
              <span />
            )}
            {next ? (
              <Button asChild variant="outline" className="min-w-0 justify-end">
                <Link
                  to="/courses/$courseId/lessons/$lessonId"
                  params={{ courseId: course.id, lessonId: next.id }}
                >
                  <span className="truncate">{next.title}</span>
                  <ArrowRight className="size-4 shrink-0" />
                </Link>
              </Button>
            ) : null}
          </nav>
        </div>
      </article>
    </div>
  )
}
