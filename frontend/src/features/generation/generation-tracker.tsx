import { Link } from '@tanstack/react-router'
import { AlertCircle, Check, CircleDashed, ExternalLink, RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { GenerationStatus } from '@/shared/api/types'
import { Button } from '@/shared/ui/button'
import { ApiErrorNotice } from '@/shared/ui/api-error-notice'
import { Progress } from '@/shared/ui/progress'
import { ClarificationForm } from './clarification-form'
import { useRetryGeneration } from './api'

const steps = ['analysis', 'architecture', 'lessons', 'content', 'finalization'] as const

export function GenerationTracker({ status }: { status: GenerationStatus }) {
  const { t } = useTranslation()
  const retry = useRetryGeneration(status.requestId)
  const activeIndex =
    status.progressPercent >= 95
      ? 4
      : status.progressPercent >= 75
        ? 3
        : status.progressPercent >= 60
          ? 2
          : status.progressPercent >= 30
            ? 1
            : 0

  if (status.pipelineStatus === 'awaiting_clarification')
    return (
      <section className="rounded-lg border border-warning/30 bg-warning-soft/35 p-5 sm:p-7">
        <h2 className="text-xl font-bold">{t('generation.awaiting')}</h2>
        <p className="mt-2 mb-7 text-sm leading-6 text-muted-foreground">{t('generation.awaitingText')}</p>
        <ClarificationForm status={status} />
      </section>
    )
  if (status.isOutOfScope)
    return (
      <section className="border-y border-warning/30 bg-warning-soft px-5 py-10 text-center">
        <AlertCircle className="mx-auto size-8 text-warning" />
        <h2 className="mt-4 text-xl font-bold">{t('generation.outOfScope')}</h2>
        <p className="mx-auto mt-2 max-w-xl text-sm text-muted-foreground">{status.errorMessage}</p>
        <Button asChild className="mt-6">
          <Link to="/generate">{t('generation.title')}</Link>
        </Button>
      </section>
    )
  if (status.pipelineStatus === 'failed')
    return (
      <section className="border-y border-danger/20 bg-danger-soft px-5 py-10 text-center">
        <AlertCircle className="mx-auto size-8 text-danger" />
        <h2 className="mt-4 text-xl font-bold">{t('generation.failed')}</h2>
        <p className="mx-auto mt-2 max-w-xl text-sm text-muted-foreground">
          {status.failureMessage ?? t('generation.failedText')}
        </p>
        <Button className="mt-6" disabled={retry.isPending} onClick={() => retry.mutate()}>
          <RotateCcw className="size-4" />
          {t('common.retry')}
        </Button>
        <ApiErrorNotice error={retry.error} />
      </section>
    )
  if (status.pipelineStatus === 'completed' && status.courseId)
    return (
      <section className="border-y border-success/20 bg-success-soft px-5 py-10 text-center">
        <span className="mx-auto grid size-10 place-items-center rounded-full bg-success text-white">
          <Check className="size-6" />
        </span>
        <h2 className="mt-4 text-xl font-bold">{t('generation.completed')}</h2>
        <Button asChild className="mt-6">
          <Link to="/courses/$courseId" params={{ courseId: status.courseId }}>
            {t('generation.openCourse')}
            <ExternalLink className="size-4" />
          </Link>
        </Button>
      </section>
    )

  return (
    <section aria-live="polite">
      <div className="mb-3 flex items-center justify-between text-sm">
        <span className="font-semibold">{t(`generation.${steps[activeIndex]}`)}</span>
        <span className="font-mono text-muted-foreground">{status.progressPercent}%</span>
      </div>
      <Progress value={status.progressPercent} />
      <ol className="mt-8 grid gap-3 sm:grid-cols-5">
        {steps.map((step, index) => (
          <li
            key={step}
            className="flex items-center gap-2 text-xs font-semibold text-muted-foreground sm:flex-col sm:items-start"
          >
            <span
              className={`grid size-7 shrink-0 place-items-center rounded-full border ${index < activeIndex ? 'border-primary bg-primary text-primary-foreground' : index === activeIndex ? 'border-primary text-primary' : 'border-border'}`}
            >
              {index < activeIndex ? <Check className="size-4" /> : <CircleDashed className="size-4" />}
            </span>
            {t(`generation.${step}`)}
          </li>
        ))}
      </ol>
      {status.warningMessage ? (
        <p className="mt-6 border-l-2 border-warning pl-3 text-sm text-muted-foreground">
          {status.warningMessage}
        </p>
      ) : null}
    </section>
  )
}
