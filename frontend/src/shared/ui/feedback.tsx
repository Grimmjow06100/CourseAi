import { AlertTriangle, Inbox, LoaderCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { ReactNode } from 'react'
import { Button } from './button'

export function LoadingState({ label }: { label: string }) {
  return (
    <div
      role="status"
      className="studio-panel flex min-h-48 items-center justify-center gap-3 text-sm text-muted-foreground"
    >
      <LoaderCircle className="size-5 animate-spin" aria-hidden="true" />
      {label}
    </div>
  )
}

interface EmptyStateProps {
  title: string
  description?: string
  action?: ReactNode
}

export function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <div className="studio-panel flex min-h-60 flex-col items-center justify-center px-6 py-10 text-center">
      <span className="mb-4 grid size-14 place-items-center rounded-2xl bg-success-soft text-primary">
        <Inbox className="size-6" aria-hidden="true" />
      </span>
      <h2 className="text-base font-semibold text-foreground">{title}</h2>
      {description ? <p className="mt-2 max-w-md text-sm text-muted-foreground">{description}</p> : null}
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  )
}

export function ErrorState({
  title,
  description,
  onRetry,
}: {
  title: string
  description: string
  onRetry?: () => void
}) {
  const { t } = useTranslation()
  return (
    <div
      role="alert"
      className="flex min-h-52 flex-col items-center justify-center rounded-2xl border border-danger/20 bg-danger-soft/40 px-6 py-10 text-center"
    >
      <AlertTriangle className="mb-4 size-8 text-danger" aria-hidden="true" />
      <h2 className="text-base font-semibold text-foreground">{title}</h2>
      <p className="mt-2 max-w-lg text-sm text-muted-foreground">{description}</p>
      {onRetry ? (
        <Button className="mt-5" variant="secondary" onClick={onRetry}>
          {t('common.retry')}
        </Button>
      ) : null}
    </div>
  )
}
