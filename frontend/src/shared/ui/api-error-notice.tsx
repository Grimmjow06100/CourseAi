import { useTranslation } from 'react-i18next'
import { ApiError } from '@/shared/api/errors'
import { getErrorMessage } from '@/shared/api/error-message'
import { Button } from './button'

export function ApiErrorNotice({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const { t } = useTranslation()
  if (!error) return null
  return (
    <div role="alert" className="my-4 space-y-2 border-l-2 border-danger pl-3 text-sm text-danger-strong">
      <p>{getErrorMessage(error, t)}</p>
      {error instanceof ApiError && error.requestId ? (
        <p className="break-all font-mono text-xs">ID: {error.requestId}</p>
      ) : null}
      {error instanceof ApiError && error.status === 401 ? (
        <a className="underline" href="/sign-in">
          {t('errors.signIn')}
        </a>
      ) : onRetry ? (
        <Button variant="secondary" size="sm" onClick={onRetry}>
          {t('common.retry')}
        </Button>
      ) : null}
    </div>
  )
}
