import { Link } from '@tanstack/react-router'
import { BookOpen, CheckCircle2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { GenerationTracking, TrackingModule, TrackingOperation } from '@/shared/api/types'

import { OperationDetails, OperationStatus, RetryOperationButton } from './tracking-operation'

function ModuleLessons({
  module,
  snapshot,
  operations,
  pending,
  retry,
}: {
  module: TrackingModule
  snapshot: GenerationTracking
  operations: Map<string, TrackingOperation>
  pending: boolean
  retry: (operation: TrackingOperation) => void
}) {
  const { t } = useTranslation()
  const [limit, setLimit] = useState(25)
  const plan = operations.get(`lesson_plan:${module.id}`)
  return (
    <div>
      {plan && (
        <div className="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-md bg-muted/50 p-3">
          <div className="space-y-2">
            <p className="text-xs font-medium">{t('tracking.kind.lesson_plan')}</p>
            <OperationDetails operation={plan} />
          </div>
          <div className="flex items-center gap-2">
            <OperationStatus status={plan.status} />
            <RetryOperationButton operation={plan} pending={pending} retry={retry} />
          </div>
        </div>
      )}
      {module.lessons.length === 0 && (
        <p className="py-3 text-sm text-muted-foreground">{t('tracking.waitingPlan')}</p>
      )}
      <ol className="divide-y">
        {module.lessons.slice(0, limit).map((lesson) => {
          const operation = operations.get(`lesson_content:${lesson.id}`)
          return (
            <li
              key={lesson.id}
              className="flex flex-col gap-3 py-4 sm:flex-row sm:items-start sm:justify-between"
            >
              <div className="min-w-0 space-y-2">
                <p className="flex items-start gap-2 text-sm font-medium">
                  <span className="mt-0.5 font-mono text-xs text-muted-foreground">
                    {lesson.order.toString().padStart(2, '0')}
                  </span>
                  {lesson.title}
                </p>
                {operation && <OperationDetails operation={operation} />}
                {!lesson.hasContent && !operation && (
                  <p className="text-xs text-muted-foreground">
                    {t(
                      snapshot.counts.lessonsExpected === null ? 'tracking.blocked' : 'tracking.unavailable',
                    )}
                  </p>
                )}
              </div>
              <div className="flex shrink-0 flex-wrap items-center gap-2">
                {lesson.hasContent ? (
                  <Badge variant="success">
                    <CheckCircle2 />
                    {t('tracking.available')}
                  </Badge>
                ) : (
                  <OperationStatus
                    status={
                      operation?.status ?? (snapshot.counts.lessonsExpected === null ? 'blocked' : 'pending')
                    }
                  />
                )}
                {lesson.hasContent && snapshot.courseId ? (
                  <Button asChild variant="ghost" size="sm">
                    <Link
                      to="/courses/$courseId/lessons/$lessonId"
                      params={{ courseId: snapshot.courseId, lessonId: lesson.id }}
                    >
                      <BookOpen className="size-3.5" />
                      {t('tracking.openLesson')}
                    </Link>
                  </Button>
                ) : operation ? (
                  <RetryOperationButton operation={operation} pending={pending} retry={retry} />
                ) : null}
              </div>
            </li>
          )
        })}
      </ol>
      {module.lessons.length > limit && (
        <Button variant="ghost" size="sm" onClick={() => setLimit((current) => current + 25)}>
          {t('tracking.moreLessons', { count: Math.min(25, module.lessons.length - limit) })}
        </Button>
      )}
    </div>
  )
}

export function TrackingModules({
  snapshot,
  pending,
  retry,
}: {
  snapshot: GenerationTracking
  pending: boolean
  retry: (operation: TrackingOperation) => void
}) {
  const { t } = useTranslation()
  const operations = new Map(snapshot.operations.map((op) => [`${op.kind}:${op.targetId ?? ''}`, op]))
  return (
    <Accordion type="multiple" className="border-y">
      {snapshot.modules.map((module) => {
        const count = module.lessons.filter((lesson) => lesson.hasContent).length
        const moduleOperations = snapshot.operations.filter(
          (op) => op.targetId === module.id || module.lessons.some((lesson) => lesson.id === op.targetId),
        )
        const incident = moduleOperations.some((op) => op.status === 'failed' || op.status === 'cancelled')
        const active = moduleOperations.some(
          (op) => op.status === 'running' || op.status === 'queued' || op.status === 'retry_scheduled',
        )
        return (
          <AccordionItem key={module.id} value={module.id}>
            <AccordionTrigger className="py-5 text-left hover:no-underline">
              <span className="flex min-w-0 flex-1 flex-wrap items-center justify-between gap-3 pr-2">
                <span className="min-w-0">
                  <span className="mb-1 block text-xs font-normal text-muted-foreground">
                    Module {module.order}
                  </span>
                  <span className="text-sm font-medium">{module.title}</span>
                  <span className="mt-1 block text-xs font-normal text-muted-foreground">
                    {module.lessons.length
                      ? t('tracking.moduleCount', { available: count, total: module.lessons.length })
                      : t('tracking.waitingPlan')}
                  </span>
                </span>
                <OperationStatus
                  status={
                    active
                      ? 'running'
                      : count > 0 && count === module.lessons.length
                        ? 'completed'
                        : incident
                          ? count > 0
                            ? 'partial'
                            : 'failed'
                          : 'pending'
                  }
                />
              </span>
            </AccordionTrigger>
            <AccordionContent>
              <ModuleLessons
                module={module}
                snapshot={snapshot}
                operations={operations}
                pending={pending}
                retry={retry}
              />
            </AccordionContent>
          </AccordionItem>
        )
      })}
    </Accordion>
  )
}
