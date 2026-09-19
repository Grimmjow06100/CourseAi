import { Link } from '@tanstack/react-router'
import { CheckCircle2, Clock3, Flag, Target, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Course } from '@/shared/api/types'
import { formatDuration } from '@/shared/lib/format'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useGenerationTracking } from '@/features/generation/tracking-api'
import { CourseCurriculum } from './course-curriculum'

function ListSection({
  title,
  values,
  icon: Icon,
}: {
  title: string
  values: string[]
  icon: typeof Target
}) {
  if (values.length === 0) return null
  return (
    <section>
      <h2 className="flex items-center gap-2 text-base font-semibold">
        <Icon className="size-4 text-muted-foreground" />
        {title}
      </h2>
      <ul className="mt-3 space-y-2">
        {values.map((value) => (
          <li key={value} className="flex gap-2 text-sm leading-6 text-muted-foreground">
            <CheckCircle2 className="mt-1 size-4 shrink-0 text-muted-foreground" />
            {value}
          </li>
        ))}
      </ul>
    </section>
  )
}

export function CourseOverview({ course }: { course: Course }) {
  const { t, i18n } = useTranslation()
  const lessons = course.modules.flatMap((module) => module.lessons)
  const firstLesson = lessons.find((lesson) => lesson.hasContent)
  const available = lessons.filter((lesson) => lesson.hasContent).length
  useGenerationTracking(course.requestId)
  return (
    <div>
      <div className="border-b border-border pb-8">
        <p className="mb-5 text-[10px] font-semibold tracking-[.16em] text-muted-foreground">
          {t('design.courseLabel')}
        </p>
        <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
          <span>{t('course.modules', { count: course.modules.length })}</span>
          <span>{t('course.lessons', { count: lessons.length })}</span>
          <span className="flex items-center gap-1">
            <Clock3 className="size-4" />
            {formatDuration(course.totalDurationMinutes, i18n.language)}
          </span>
        </div>
        <h1 className="mt-4 max-w-4xl text-3xl font-semibold leading-tight tracking-tight sm:text-4xl">
          {course.title}
        </h1>
        <p className="mt-4 max-w-4xl text-base leading-7 text-muted-foreground">{course.synopsis}</p>
        {firstLesson ? (
          <Button asChild variant="brand" className="mt-6">
            <Link
              to="/courses/$courseId/lessons/$lessonId"
              params={{ courseId: course.id, lessonId: firstLesson.id }}
            >
              {t('course.start')}
            </Link>
          </Button>
        ) : null}
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <Badge variant={course.status === 'partial' ? 'warning' : 'secondary'}>
            {t(`course.status.${course.status}`)}
          </Badge>
          <span className="text-sm text-muted-foreground">
            {available}/{lessons.length} {t('design.ready')}
          </span>
          <Button asChild variant="link" size="sm">
            <Link to="/generations/$requestId" params={{ requestId: course.requestId }}>
              {t('tracking.openTracking')}
            </Link>
          </Button>
        </div>
      </div>
      <div className="grid gap-10 py-8 lg:grid-cols-[minmax(0,1fr)_320px]">
        <div>
          <h2 className="mb-4 text-xl font-semibold">{t('course.curriculum')}</h2>
          <CourseCurriculum course={course} />
        </div>
        <aside>
          <Card>
            <CardContent className="space-y-7">
              <section>
                <h2 className="text-sm font-semibold">{t('design.level')}</h2>
                <p className="mt-2 text-sm text-muted-foreground">
                  {t(`design.${course.currentLevel}`)} → {t(`design.${course.targetLevel}`)}
                </p>
              </section>
              {course.targetAudience ? (
                <section>
                  <h2 className="flex items-center gap-2 text-base font-semibold">
                    <Users className="size-4 text-muted-foreground" />
                    {t('course.audience')}
                  </h2>
                  <p className="mt-3 text-sm leading-6 text-muted-foreground">{course.targetAudience}</p>
                </section>
              ) : null}
              <ListSection title={t('course.objectives')} values={course.goals} icon={Target} />
              <ListSection
                title={t('course.prerequisites')}
                values={course.prerequisites}
                icon={CheckCircle2}
              />
              <ListSection title={t('course.skills')} values={course.acquiredSkills} icon={Flag} />
              {course.finalProjectTitle ? (
                <section className="border-l-2 border-border pl-4">
                  <h2 className="flex items-center gap-2 text-base font-semibold">
                    <Flag className="size-4 text-muted-foreground" />
                    {t('course.finalProject')}
                  </h2>
                  <h3 className="mt-3 text-sm font-semibold">{course.finalProjectTitle}</h3>
                  <p className="mt-2 text-sm leading-6 text-muted-foreground">
                    {course.finalProjectDescription}
                  </p>
                  {course.finalProjectConstraints.length ? (
                    <ul className="mt-3 list-disc space-y-2 pl-4 text-xs leading-5 text-muted-foreground">
                      {course.finalProjectConstraints.map((constraint) => (
                        <li key={constraint}>{constraint}</li>
                      ))}
                    </ul>
                  ) : null}
                </section>
              ) : null}
            </CardContent>
          </Card>
        </aside>
      </div>
    </div>
  )
}
