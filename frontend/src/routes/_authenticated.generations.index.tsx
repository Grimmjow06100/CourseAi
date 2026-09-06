import { getErrorMessage } from '@/shared/api/error-message'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useGenerationList } from '@/features/generation/api'
import { GenerationList } from '@/features/generation/generation-list'
import { generationListSearchSchema } from '@/features/generation/schemas'
import type { PipelineStatus } from '@/shared/api/types'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/feedback'
import { Select } from '@/shared/ui/form-controls'
import { PageHeader, Pagination } from '@/shared/ui/page'

function GenerationHistoryPage() {
  const { t } = useTranslation()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const query = useGenerationList({ ...(search.status ? { status: search.status } : {}), page: search.page })
  return (
    <div>
      <PageHeader
        title={t('generation.historyTitle')}
        description={t('generation.historySubtitle')}
        actions={
          <Select
            aria-label={t('generation.allStatuses')}
            value={search.status ?? ''}
            onChange={(event) =>
              void navigate({
                search: (previous) => ({
                  ...previous,
                  status: (event.target.value || undefined) as PipelineStatus | undefined,
                  page: 1,
                }),
              })
            }
          >
            <option value="">{t('generation.allStatuses')}</option>
            {(['queued', 'running', 'awaiting_clarification', 'completed', 'failed'] as const).map(
              (status) => (
                <option key={status} value={status}>
                  {t(`generation.status.${status}`)}
                </option>
              ),
            )}
          </Select>
        }
      />
      {query.isLoading ? (
        <LoadingState label={t('common.loading')} />
      ) : query.error ? (
        <ErrorState
          title={t('errors.generic')}
          description={getErrorMessage(query.error, t)}
          onRetry={() => void query.refetch()}
        />
      ) : query.data?.items.length ? (
        <>
          <GenerationList generations={query.data.items} />
          <Pagination
            page={query.data.page}
            totalPages={query.data.totalPages}
            hasPrevious={query.data.hasPrevious}
            hasNext={query.data.hasNext}
            label={t('common.page', { current: query.data.page, total: query.data.totalPages })}
            onPageChange={(page) => void navigate({ search: (previous) => ({ ...previous, page }) })}
          />
        </>
      ) : (
        <EmptyState title={t('common.noResult')} description={t('dashboard.emptyGenerations')} />
      )}
    </div>
  )
}
export const Route = createFileRoute('/_authenticated/generations/')({
  validateSearch: generationListSearchSchema,
  component: GenerationHistoryPage,
})
