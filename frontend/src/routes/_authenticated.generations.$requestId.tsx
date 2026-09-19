import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Toaster } from '@/components/ui/sonner'
import { useGenerationTracking } from '@/features/generation/tracking-api'
import { GenerationTracker } from '@/features/generation/generation-tracker'
import { getErrorMessage } from '@/shared/api/error-message'
import { ErrorState, LoadingState } from '@/shared/ui/feedback'
import { PageHeader } from '@/shared/ui/page'

function GenerationDetailPage() {
  const { requestId } = Route.useParams()
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  const { t, i18n } = useTranslation()
  const query = useGenerationTracking(requestId)
  if (query.isPending) return <LoadingState label={t('common.loading')} />
  if (!query.data)
    return (
      <ErrorState
        title={t('errors.generic')}
        description={getErrorMessage(query.error, t)}
        onRetry={() => void query.refetch()}
      />
    )
  return (
    <div>
      <PageHeader
        title={query.data.title || t('generation.trackingTitle')}
        description={t('generation.trackingSubtitle')}
      />
      <div className="mb-5 flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
        <div className="flex flex-wrap items-center gap-2">
          <span
            className={
              query.data.hasActiveWork
                ? 'size-1.5 rounded-full bg-brand'
                : 'size-1.5 rounded-full bg-muted-foreground'
            }
            aria-hidden="true"
          />
          <span>{t(query.data.hasActiveWork ? 'tracking.live' : 'tracking.paused')}</span>
          <span>
            ·{' '}
            {t('tracking.checked', {
              time: new Date(query.data.observedAt).toLocaleTimeString(i18n.language),
            })}
          </span>
        </div>
        <Button variant="ghost" size="sm" disabled={query.isFetching} onClick={() => void query.refetch()}>
          <RefreshCw className={query.isFetching ? 'size-3.5 animate-spin' : 'size-3.5'} aria-hidden="true" />
          {t('tracking.check')}
        </Button>
      </div>
      {query.error && (
        <Alert className="mb-5">
          <AlertDescription>{t('tracking.offline')}</AlertDescription>
        </Alert>
      )}
      <GenerationTracker
        snapshot={query.data}
        tab={tab ?? 'modules'}
        onTabChange={(tab) => void navigate({ search: { tab }, replace: true })}
      />
      <Toaster position="top-right" closeButton />
    </div>
  )
}

export const Route = createFileRoute('/_authenticated/generations/$requestId')({
  validateSearch: z.object({ tab: z.enum(['modules', 'operations', 'history']).optional().catch(undefined) }),
  component: GenerationDetailPage,
})
