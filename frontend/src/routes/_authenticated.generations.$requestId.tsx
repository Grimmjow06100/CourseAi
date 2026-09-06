import { getErrorMessage } from '@/shared/api/error-message'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useGenerationStatus } from '@/features/generation/api'
import { GenerationTracker } from '@/features/generation/generation-tracker'
import { ErrorState, LoadingState } from '@/shared/ui/feedback'
import { PageHeader } from '@/shared/ui/page'

function GenerationDetailPage() {
  const { requestId } = Route.useParams()
  const { t } = useTranslation()
  const query = useGenerationStatus(requestId)
  if (query.isLoading) return <LoadingState label={t('common.loading')} />
  if (query.error || !query.data)
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
        title={query.data.suggestedTitle ?? t('generation.trackingTitle')}
        description={t('generation.trackingSubtitle')}
      />
      <GenerationTracker status={query.data} />
    </div>
  )
}
export const Route = createFileRoute('/_authenticated/generations/$requestId')({
  component: GenerationDetailPage,
})
