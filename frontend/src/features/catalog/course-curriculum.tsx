import { Link } from '@tanstack/react-router'
import { BookOpen, Clock3, FileText } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Course } from '@/shared/api/types'
import { formatDuration } from '@/shared/lib/format'
import { Button } from '@/components/ui/button'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'

export function CourseCurriculum({ course }: { course: Course }) {
  const { t, i18n } = useTranslation()
  return (
    <Accordion type="multiple" defaultValue={course.modules[0] ? [course.modules[0].id] : []}>
      {course.modules.map((module) => {
        const available = module.lessons.filter((lesson) => lesson.hasContent).length
        return (
          <AccordionItem key={module.id} value={module.id}>
            <AccordionTrigger className="gap-3 py-5 hover:no-underline">
              <span className="min-w-0 flex-1">
                <span className="block text-xs font-normal text-muted-foreground">Module {module.order}</span>
                <span className="mt-1 block text-base font-semibold">{module.title}</span>
              </span>
              <Badge variant="secondary" className="shrink-0">
                {module.lessons.length ? `${available}/${module.lessons.length}` : t('design.planned')}
              </Badge>
            </AccordionTrigger>
            <AccordionContent>
              <p className="mb-4 text-sm leading-6 text-muted-foreground">{module.description}</p>
              <ol className="divide-y divide-border">
                {module.lessons.map((lesson) => (
                  <li key={lesson.id}>
                    <Link
                      to="/courses/$courseId/lessons/$lessonId"
                      params={{ courseId: course.id, lessonId: lesson.id }}
                      className="flex items-center gap-3 rounded-md px-2 py-4 transition-colors hover:bg-muted"
                    >
                      {lesson.hasContent ? (
                        <BookOpen className="size-4 shrink-0 text-muted-foreground" />
                      ) : (
                        <FileText className="size-4 shrink-0 text-muted-foreground" />
                      )}
                      <span className="min-w-0 flex-1">
                        <span className="block font-medium leading-6">
                          {lesson.order}. {lesson.title}
                        </span>
                        <span className="mt-1 block text-xs text-muted-foreground">
                          {t(`lesson.type.${lesson.type}`)} ·{' '}
                          {lesson.hasContent ? t('design.ready') : t('design.planned')}
                        </span>
                      </span>
                      <span className="hidden shrink-0 items-center gap-1 text-xs text-muted-foreground sm:flex">
                        <Clock3 className="size-3.5" />
                        {formatDuration(lesson.estimatedDurationMinutes, i18n.language)}
                      </span>
                    </Link>
                  </li>
                ))}
              </ol>
              {available < module.lessons.length || !module.lessons.length ? (
                <Button asChild variant="outline" size="sm" className="mt-3">
                  <Link to="/generations/$requestId" params={{ requestId: course.requestId }}>
                    {t('tracking.openTracking')}
                  </Link>
                </Button>
              ) : null}
            </AccordionContent>
          </AccordionItem>
        )
      })}
    </Accordion>
  )
}
