import { Link } from '@tanstack/react-router'
import { ArrowRight, FolderClock, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { GenerationSummary } from '@/shared/api/types'
import { formatDate } from '@/shared/lib/format'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { PipelineStatusBadge } from './status-badge'
import { useDeleteGeneration } from './api'

export function GenerationList({ generations, limit }: { generations: GenerationSummary[]; limit?: number }) {
  const { t, i18n } = useTranslation()
  const remove = useDeleteGeneration()
  return (
    <div className="studio-panel divide-y divide-border px-5">
      {generations.slice(0, limit).map((generation) => (
        <article
          key={generation.requestId}
          className="grid gap-3 py-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
        >
          <div className="flex min-w-0 items-start gap-3">
            <span className="mt-1 hidden size-10 shrink-0 place-items-center rounded-xl bg-success-soft text-primary sm:grid">
              <FolderClock className="size-5" aria-hidden="true" />
            </span>
            <div className="min-w-0">
              <div className="mb-2 flex flex-wrap items-center gap-2">
                <PipelineStatusBadge generation={generation} />
                <span className="text-xs text-muted-foreground">
                  {formatDate(generation.createdAt, i18n.language)}
                </span>
              </div>
              <h3 className="line-clamp-2 font-bold">{generation.title}</h3>
              <p className="mt-1 line-clamp-1 text-sm text-muted-foreground">
                {generation.initialUserPrompt}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <ConfirmDialog
              trigger={
                <Button variant="ghost" aria-label={t('common.delete')}>
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
                {generation.pipelineStatus === 'awaiting_clarification'
                  ? t('course.continue')
                  : t('common.open')}
                <ArrowRight className="size-4" />
              </Link>
            </Button>
          </div>
        </article>
      ))}
    </div>
  )
}
