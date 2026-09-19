import { LoaderCircle, RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { TrackingOperation } from '@/shared/api/types'

export function OperationStatus({ status }: { status: string }) {
  const { t } = useTranslation()
  const variant =
    status === 'failed'
      ? 'destructive'
      : status === 'completed'
        ? 'success'
        : status === 'retry_scheduled' || status === 'partial'
          ? 'warning'
          : status === 'running'
            ? 'info'
            : 'secondary'
  return (
    <Badge variant={variant}>
      {status === 'running' && <LoaderCircle className="animate-spin" aria-hidden="true" />}
      {t(`tracking.status.${status}`)}
    </Badge>
  )
}

export function OperationDetails({ operation }: { operation: TrackingOperation }) {
  const { t, i18n } = useTranslation()
  return (
    <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
      {operation.failureCode && (
        <span className="basis-full">{t(`tracking.failure.${operation.failureCode}`)}</span>
      )}
      <span>{t('tracking.attempts', { current: operation.attemptCount, max: operation.maxAttempts })}</span>
      {operation.startedAt && (
        <time dateTime={operation.startedAt}>
          {t('tracking.started', { time: new Date(operation.startedAt).toLocaleString(i18n.language) })}
        </time>
      )}
      {operation.completedAt && (
        <time dateTime={operation.completedAt}>
          {t('tracking.ended', { time: new Date(operation.completedAt).toLocaleString(i18n.language) })}
        </time>
      )}
      {operation.operationVersion > 1 && (
        <span>{t('tracking.operationVersion', { version: operation.operationVersion - 1 })}</span>
      )}
      {operation.status === 'retry_scheduled' && (
        <time dateTime={operation.availableAt}>
          {t('tracking.retryAt', { time: new Date(operation.availableAt).toLocaleTimeString(i18n.language) })}
        </time>
      )}
    </div>
  )
}

export function RetryOperationButton({
  operation,
  pending,
  retry,
}: {
  operation: TrackingOperation
  pending: boolean
  retry: (operation: TrackingOperation) => void
}) {
  const { t } = useTranslation()
  return operation.retryable ? (
    <Button variant="outline" size="sm" disabled={pending} onClick={() => retry(operation)}>
      <RotateCcw className="size-3.5" aria-hidden="true" />
      {t('tracking.retry')}
    </Button>
  ) : null
}
