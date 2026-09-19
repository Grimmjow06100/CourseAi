import { Link } from '@tanstack/react-router'
import { ArrowUpRight, Ellipsis, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { GenerationSummary } from '@/shared/api/types'
import { formatDate } from '@/shared/lib/format'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { PipelineStatusBadge } from './status-badge'
import { useDeleteGeneration } from './api'
import { hasCompleteCourse } from './presentation'

export function GenerationList({ generations, limit }: { generations: GenerationSummary[]; limit?: number }) {
  const { t, i18n } = useTranslation()
  const remove = useDeleteGeneration()
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('generation.titleLabel')}</TableHead>
          <TableHead className="hidden sm:table-cell">{t('generation.allStatuses')}</TableHead>
          <TableHead className="hidden lg:table-cell">Date</TableHead>
          <TableHead className="text-right">
            <span className="sr-only">Actions</span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {generations.slice(0, limit).map((generation) => (
          <TableRow key={generation.requestId}>
            <TableCell className="min-w-0 py-4 whitespace-normal">
              <Link
                to="/generations/$requestId"
                params={{ requestId: generation.requestId }}
                className="line-clamp-2 font-medium hover:underline"
              >
                {generation.title}
              </Link>
              <p className="mt-1 line-clamp-1 max-w-md text-xs text-muted-foreground">
                {generation.initialUserPrompt}
              </p>
              {(hasCompleteCourse(generation) || generation.pipelineStatus === 'partial') && (
                <p className="mt-1 max-w-md text-xs text-muted-foreground">
                  {t(
                    hasCompleteCourse(generation) ? 'generation.availableAll' : 'generation.availablePartial',
                  )}
                </p>
              )}
              <span className="mt-2 block sm:hidden">
                <PipelineStatusBadge generation={generation} />
              </span>
            </TableCell>
            <TableCell className="hidden sm:table-cell">
              <PipelineStatusBadge generation={generation} />
            </TableCell>
            <TableCell className="hidden text-xs text-muted-foreground lg:table-cell">
              {formatDate(generation.createdAt, i18n.language)}
            </TableCell>
            <TableCell>
              <div className="flex justify-end gap-1">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={t('generation.actionsFor', { title: generation.title })}
                    >
                      <Ellipsis className="size-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <ConfirmDialog
                      trigger={
                        <DropdownMenuItem onSelect={(event) => event.preventDefault()}>
                          <Trash2 className="size-4" />
                          {t('common.delete')}
                        </DropdownMenuItem>
                      }
                      title={t('generation.deleteTitle')}
                      description={t('generation.deleteDescription')}
                      confirmLabel={t('common.delete')}
                      cancelLabel={t('common.cancel')}
                      pending={remove.isPending}
                      onConfirm={() => remove.mutateAsync(generation.requestId)}
                    />
                  </DropdownMenuContent>
                </DropdownMenu>
                <Button asChild variant="ghost" size="icon">
                  <Link
                    to="/generations/$requestId"
                    params={{ requestId: generation.requestId }}
                    aria-label={t('common.open')}
                    title={t('common.open')}
                  >
                    <ArrowUpRight className="size-4" />
                  </Link>
                </Button>
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
