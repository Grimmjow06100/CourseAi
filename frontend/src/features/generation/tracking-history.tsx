import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ApiErrorNotice } from '@/shared/ui/api-error-notice'
import type { GenerationTracking } from '@/shared/api/types'
import { useGenerationEvents } from './tracking-api'
import { generationKeys } from './query-keys'

export function TrackingHistory({ snapshot, enabled }: { snapshot: GenerationTracking; enabled: boolean }) {
  const { t, i18n } = useTranslation()
  const cache = useQueryClient()
  const events = useGenerationEvents(snapshot.requestId, enabled, snapshot.hasActiveWork)
  const targetTitles = new Map(
    snapshot.modules.flatMap((module) => [
      [module.id, module.title] as const,
      ...module.lessons.map((lesson) => [lesson.id, lesson.title] as const),
    ]),
  )
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">
          {t('tracking.attempt', { count: snapshot.generationAttempt })}
        </p>
        <Button
          variant="outline"
          size="sm"
          disabled={events.isFetching}
          onClick={() => void cache.resetQueries({ queryKey: generationKeys.events(snapshot.requestId) })}
        >
          {t('tracking.historyRefresh')}
        </Button>
      </div>
      {!snapshot.historyComplete && (
        <p className="text-sm text-muted-foreground">{t('tracking.historyIncomplete')}</p>
      )}
      <ApiErrorNotice error={events.error} />
      {events.isPending && (
        <p role="status" className="text-sm text-muted-foreground">
          {t('common.loading')}
        </p>
      )}
      <ol className="divide-y">
        {events.data?.pages
          .flatMap((page) => page.items)
          .map((event) => (
            <li key={event.id} className="grid gap-2 py-4 sm:grid-cols-[150px_1fr_auto] sm:items-start">
              <time dateTime={event.occurredAt} className="font-mono text-xs text-muted-foreground">
                {new Date(event.occurredAt).toLocaleString(i18n.language)}
              </time>
              <div className="space-y-1">
                <p className="text-sm font-medium">{t(`tracking.kind.${event.kind}`)}</p>
                <p className="text-xs text-muted-foreground">
                  {t('tracking.attempt', { count: event.generationAttempt })}
                  {event.operationVersion !== null
                    ? ` · ${t('tracking.eventVersion', { version: event.operationVersion, attempt: event.attemptCount ?? 0 })}`
                    : ''}
                </p>
                {event.targetId && (
                  <p className="break-all font-mono text-[11px] text-muted-foreground">
                    {targetTitles.get(event.targetId) ?? event.targetId}
                  </p>
                )}
              </div>
              <Badge
                variant={
                  event.status === 'failed'
                    ? 'destructive'
                    : event.status === 'completed'
                      ? 'success'
                      : 'secondary'
                }
              >
                {t(`tracking.status.${event.status}`)}
              </Badge>
            </li>
          ))}
      </ol>
      {events.data?.pages[0]?.items.length === 0 && (
        <p className="py-6 text-sm text-muted-foreground">{t('tracking.noEvents')}</p>
      )}
      {events.hasNextPage && (
        <Button
          variant="outline"
          disabled={events.isFetchingNextPage}
          onClick={() => void events.fetchNextPage()}
        >
          {t('tracking.more')}
        </Button>
      )}
    </div>
  )
}
