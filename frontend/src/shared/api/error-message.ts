import type { TFunction } from 'i18next'
import { ApiError } from './errors'

export function getErrorMessage(error: unknown, t: TFunction) {
  if (error instanceof TypeError && /fetch|network|load failed/i.test(error.message))
    return t('errors.network')
  if (!(error instanceof ApiError)) return t('errors.generic')
  if (error.status === 401)
    return t(error.code === 'session_expired' ? 'errors.unauthorized' : 'errors.authenticationRejected')
  if (error.status === 404) return t('errors.notFound')
  if (error.status === 429) return t('errors.rateLimited')
  if (error.status === 503) return t('errors.unavailable')
  if (error.status >= 500) return t('errors.generic')
  return error.message || t('errors.generic')
}
