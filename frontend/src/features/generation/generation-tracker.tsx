import { Link } from '@tanstack/react-router'
import { AlertCircle, ArrowUpRight, Check, CircleDashed, LoaderCircle, RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ApiError } from '@/shared/api/errors'
import type { GenerationTracking, TrackingOperation } from '@/shared/api/types'
import { ApiErrorNotice } from '@/shared/ui/api-error-notice'
import { LoadingState } from '@/shared/ui/feedback'
import { ClarificationForm } from './clarification-form'
import { useGenerationStatus, useRetryGeneration } from './api'
import { useRetryOperations } from './tracking-api'
import { TrackingHistory } from './tracking-history'
import { OperationDetails, OperationStatus, RetryOperationButton, TrackingModules } from './tracking-modules'

function ClarificationStep({ requestId }: { requestId: string }) {
  const { t } = useTranslation()
  const query = useGenerationStatus(requestId)
  return (
    <Card className="border-warning/40 shadow-none">
      <CardContent className="pt-6">
        <h2 className="mb-2 text-lg font-semibold">{t('generation.awaiting')}</h2>
        <p className="mb-6 text-sm text-muted-foreground">{t('generation.awaitingText')}</p>
        {query.data ? (
          <ClarificationForm status={query.data} />
        ) : query.isPending ? (
          <LoadingState label={t('common.loading')} />
        ) : (
          <ApiErrorNotice error={query.error} onRetry={() => void query.refetch()} />
        )}
      </CardContent>
    </Card>
  )
}

export function GenerationTracker({
  snapshot,
  tab = 'modules',
  onTabChange,
}: {
  snapshot: GenerationTracking
  tab?: 'modules' | 'operations' | 'history'
  onTabChange?: (tab: 'modules' | 'operations' | 'history') => void
}) {
  const { t } = useTranslation()
  const retry = useRetryOperations(snapshot.requestId)
  const restart = useRetryGeneration(snapshot.requestId)
  const stopped = snapshot.operations.filter((operation) => operation.retryable)
  const expected = snapshot.counts.lessonsExpected
  const progress = expected ? (snapshot.counts.lessonsAvailable / expected) * 100 : null
  const retryPending = retry.isPending || restart.isPending
  const retrySelection = (operations: TrackingOperation[]) => {
    retry.mutate(
      operations.map((operation) => ({ jobId: operation.id, operationVersion: operation.operationVersion })),
      { onSuccess: () => toast.success(t('tracking.resumed')) },
    )
  }
  const summary = snapshot.isOutOfScope
    ? 'scope'
    : snapshot.contentComplete
      ? snapshot.reconciliation === 'pending'
        ? 'repair'
        : 'complete'
      : snapshot.pipelineStatus === 'partial'
        ? 'partial'
        : snapshot.pipelineStatus === 'failed'
          ? 'failed'
          : snapshot.counts.jobsFailed > 0
            ? 'incidents'
            : snapshot.reconciliation === 'attention_required'
              ? 'attention'
              : null
  const rootOperations = snapshot.operations.filter(
    (operation) => !['lesson_plan', 'lesson_content'].includes(operation.kind),
  )
  return (
    <div className="space-y-7">
      <Card className="gap-4 py-5 shadow-none">
        <CardContent className="space-y-5 px-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="space-y-3">
              <OperationStatus status={snapshot.pipelineStatus} />
              <p className="text-lg font-semibold tabular-nums">
                {expected === null
                  ? snapshot.counts.modules
                    ? t('tracking.plans', {
                        ready: snapshot.counts.plansReady,
                        total: snapshot.counts.modules,
                      })
                    : t('tracking.preparing')
                  : t('tracking.lessons', { available: snapshot.counts.lessonsAvailable, expected })}
              </p>
              <p className="text-xs text-muted-foreground">
                {t('tracking.active', { count: snapshot.counts.jobsActive })} ·{' '}
                {t('tracking.errors', { count: snapshot.counts.jobsFailed })}
              </p>
            </div>
            {snapshot.courseId && snapshot.counts.lessonsAvailable > 0 && (
              <Button asChild variant="brand">
                <Link to="/courses/$courseId" params={{ courseId: snapshot.courseId }}>
                  {t('tracking.openCourse')}
                  <ArrowUpRight className="size-4" />
                </Link>
              </Button>
            )}
          </div>
          <Progress
            value={progress}
            aria-label={t('tracking.lessons', {
              available: snapshot.counts.lessonsAvailable,
              expected: expected ?? '…',
            })}
            className="h-1.5"
          />
          {expected === null && <p className="text-xs text-muted-foreground">{t('tracking.unknown')}</p>}
          <p className="sr-only" role="status" aria-live="polite">
            {t('tracking.lessons', {
              available: snapshot.counts.lessonsAvailable,
              expected: expected ?? '…',
            })}
            . {t(`tracking.status.${snapshot.pipelineStatus}`)}
          </p>
          {summary && (
            <Alert className="border-0 bg-muted/50">
              <AlertCircle aria-hidden="true" />
              <AlertDescription>
                {t(`tracking.${summary}`, { count: snapshot.counts.jobsFailed })}
              </AlertDescription>
            </Alert>
          )}
          {stopped.length > 0 && (
            <div className="space-y-2">
              <Button variant="outline" disabled={retryPending} onClick={() => retrySelection(stopped)}>
                <RotateCcw className="size-4" />
                {retryPending ? t('tracking.retrying') : t('tracking.retryAll', { count: stopped.length })}
              </Button>
              <p className="text-xs text-muted-foreground">{t('tracking.retryNote')}</p>
            </div>
          )}
          {stopped.length === 0 &&
            snapshot.operations.length === 0 &&
            !snapshot.isOutOfScope &&
            !snapshot.contentComplete &&
            (snapshot.pipelineStatus === 'failed' || snapshot.pipelineStatus === 'partial') && (
              <Button variant="outline" disabled={retryPending} onClick={() => restart.mutate()}>
                <RotateCcw className="size-4" />
                {t('tracking.restart')}
              </Button>
            )}
          {retry.error instanceof ApiError && retry.error.status === 409 ? (
            <p role="status" className="text-sm text-muted-foreground">
              {t('tracking.stale')}
            </p>
          ) : (
            <ApiErrorNotice error={retry.error} />
          )}
          <ApiErrorNotice error={restart.error} />
        </CardContent>
      </Card>

      {snapshot.pipelineStatus === 'awaiting_clarification' && (
        <ClarificationStep requestId={snapshot.requestId} />
      )}

      <section aria-label={t('tracking.pipeline')}>
        <h2 className="mb-4 text-base font-semibold">{t('tracking.pipeline')}</h2>
        <ol className="grid gap-2 sm:grid-cols-2 xl:grid-cols-5">
          {snapshot.phases.map((phase, index) => (
            <li key={phase.kind} className="flex gap-3 rounded-lg border p-3">
              <span className="mt-0.5 text-muted-foreground" aria-hidden="true">
                {phase.status === 'completed' ? (
                  <Check className="size-4 text-success" />
                ) : phase.status === 'running' ? (
                  <LoaderCircle className="size-4 animate-spin" />
                ) : (
                  <CircleDashed className="size-4" />
                )}
              </span>
              <div className="flex min-w-0 flex-1 items-center justify-between gap-2 sm:block sm:space-y-2">
                <p className="text-xs font-medium">
                  {index + 1}. {t(`tracking.kind.${phase.kind}`)}
                </p>
                <OperationStatus status={phase.status} />
              </div>
            </li>
          ))}
        </ol>
      </section>

      <Tabs
        value={tab}
        onValueChange={(value) => {
          if (value === 'modules' || value === 'operations' || value === 'history') onTabChange?.(value)
        }}
      >
        <TabsList className="mb-5 flex h-auto w-fit max-w-full flex-wrap">
          <TabsTrigger value="modules">{t('tracking.modules')}</TabsTrigger>
          <TabsTrigger value="operations">{t('tracking.activity')}</TabsTrigger>
          <TabsTrigger value="history">{t('tracking.history')}</TabsTrigger>
        </TabsList>
        <TabsContent value="modules" className="space-y-3">
          <p className="text-xs text-muted-foreground">{t('tracking.jobExplanation')}</p>
          <TrackingModules
            snapshot={snapshot}
            pending={retryPending}
            retry={(operation) => retrySelection([operation])}
          />
          {snapshot.modules.length === 0 && (
            <p className="py-5 text-sm text-muted-foreground">
              {t('tracking.kind.architecture')} — {t('tracking.status.pending')}
            </p>
          )}
        </TabsContent>
        <TabsContent value="operations">
          <Accordion type="multiple">
            {rootOperations.map((operation) => (
              <AccordionItem key={operation.id} value={operation.id}>
                <AccordionTrigger className="hover:no-underline">
                  <span className="flex flex-wrap items-center gap-3">
                    <span>{t(`tracking.kind.${operation.kind}`)}</span>
                    <OperationStatus status={operation.status} />
                  </span>
                </AccordionTrigger>
                <AccordionContent className="space-y-3">
                  <OperationDetails operation={operation} />
                  <p className="break-all font-mono text-xs text-muted-foreground">{operation.id}</p>
                  <RetryOperationButton
                    operation={operation}
                    pending={retryPending}
                    retry={(op) => retrySelection([op])}
                  />
                </AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        </TabsContent>
        <TabsContent value="history">
          <TrackingHistory snapshot={snapshot} enabled={tab === 'history'} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
