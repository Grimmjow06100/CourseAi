import { Pagination as PaginationRoot } from '@/components/ui/pagination'
import type { ReactNode } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTranslation } from 'react-i18next'

export function PageHeader({
  title,
  description,
  eyebrow,
  actions,
}: {
  title: string
  description?: string
  eyebrow?: string
  actions?: ReactNode
}) {
  return (
    <header className="mb-8 flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
      <div className="min-w-0">
        {eyebrow ? (
          <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">{eyebrow}</p>
        ) : null}
        <h1 className="text-2xl font-semibold leading-tight tracking-tight sm:text-3xl">{title}</h1>
        {description ? (
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground sm:text-base">{description}</p>
        ) : null}
      </div>
      {actions ? <div className="shrink-0">{actions}</div> : null}
    </header>
  )
}

export function SectionHeader({ title, action }: { title: string; action?: ReactNode }) {
  return (
    <div className="mb-4 flex items-center justify-between gap-4">
      <h2 className="text-base font-semibold">{title}</h2>
      {action}
    </div>
  )
}

export function Pagination({
  page,
  totalPages,
  hasPrevious,
  hasNext,
  onPageChange,
  label,
}: {
  page: number
  totalPages: number
  hasPrevious: boolean
  hasNext: boolean
  onPageChange: (page: number) => void
  label: string
}) {
  const { t } = useTranslation()
  if (totalPages <= 1) return null
  return (
    <PaginationRoot
      className="mt-8 flex items-center justify-between border-t border-border pt-5"
      aria-label="Pagination"
    >
      <Button
        variant="secondary"
        aria-label={t('common.previous')}
        disabled={!hasPrevious}
        onClick={() => onPageChange(page - 1)}
      >
        <ChevronLeft className="size-4" />{' '}
      </Button>
      <span className="text-sm text-muted-foreground">{label}</span>
      <Button
        variant="secondary"
        aria-label={t('common.next')}
        disabled={!hasNext}
        onClick={() => onPageChange(page + 1)}
      >
        <ChevronRight className="size-4" />{' '}
      </Button>
    </PaginationRoot>
  )
}
