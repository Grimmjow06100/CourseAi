import { useTranslation } from 'react-i18next'
import { ApiError } from '@/shared/api/errors'
import { getErrorMessage } from '@/shared/api/error-message'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

export function ApiErrorNotice({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const { t } = useTranslation()
  if (!error) return null
  return (
    <Alert variant="destructive" className="my-4">
      <AlertDescription className="space-y-2">
        <p>{getErrorMessage(error, t)}</p>
        {error instanceof ApiError && error.requestId ? (
          <p className="break-all font-mono text-xs">ID: {error.requestId}</p>
        ) : null}
        {error instanceof ApiError && error.status === 401 ? (
          <a className="underline" href="/sign-in">
            {t('errors.signIn')}
          </a>
        ) : null}
        {onRetry && !(error instanceof ApiError && error.code === 'session_expired') ? (
          <Button variant="secondary" size="sm" onClick={onRetry}>
            {t('common.retry')}
          </Button>
        ) : null}
      </AlertDescription>
    </Alert>
  )
}
