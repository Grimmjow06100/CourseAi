import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

export function PartialGenerationNotice({ requestId }: { requestId: string }) {
  const { t } = useTranslation()
  return (
    <div role="alert" className="my-4 text-sm text-danger-strong">
      <p>{t('generation.failed')}</p>
      <Link className="mt-2 inline-block underline" to="/generations/$requestId" params={{ requestId }}>
        {t('generation.trackingTitle')}
      </Link>
    </div>
  )
}
