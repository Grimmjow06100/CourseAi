import { Link } from '@tanstack/react-router'
import { ArrowRight, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { GenerationSummary } from '@/shared/api/types'
import { formatDate } from '@/shared/lib/format'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { PipelineStatusBadge } from '@/shared/ui/status-badge'
import { useDeleteGeneration } from './api'

export function GenerationList({ generations, limit }: { generations: GenerationSummary[]; limit?: number }) {
  const { t, i18n } = useTranslation()
  const remove = useDeleteGeneration()
  return (
    <div className="divide-y divide-border border-y border-border">
      {generations.slice(0, limit).map((generation) => (
        <article
          key={generation.requestId}
          className="grid gap-3 py-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
        >
          <div className="min-w-0">
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <PipelineStatusBadge status={generation.pipelineStatus} />
              <span className="text-xs text-muted-foreground">
                {formatDate(generation.createdAt, i18n.language)}
              </span>
            </div>
            <h3 className="truncate font-bold">{generation.title}</h3>
            <p className="mt-1 line-clamp-1 text-sm text-muted-foreground">{generation.initialUserPrompt}</p>
          </div>
          <div className="flex items-center gap-2">
            <ConfirmDialog
              trigger={
                <Button variant="icon" aria-label={t('common.delete')}>
                  <Trash2 className="size-4" />
                </Button>
              }
              title={t('generation.deleteTitle')}
              description={t('generation.deleteDescription')}
              confirmLabel={t('common.delete')}
              cancelLabel={t('common.cancel')}
              pending={remove.isPending}
              onConfirm={() => remove.mutateAsync(generation.requestId)}
            />
            <Button asChild variant="secondary" size="sm">
              <Link to="/generations/$requestId" params={{ requestId: generation.requestId }}>
                {t('common.open')}
                <ArrowRight className="size-4" />
              </Link>
            </Button>
          </div>
        </article>
      ))}
    </div>
  )
}
