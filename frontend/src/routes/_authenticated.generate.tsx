import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { GenerationForm } from '@/features/generation/generation-form'
import { PageHeader } from '@/shared/ui/page'

function GeneratePage() {
  const { t } = useTranslation()
  return (
    <div>
      <PageHeader title={t('generation.title')} description={t('generation.subtitle')} />
      <div className="rounded-lg border border-border bg-surface p-5 sm:p-7">
        <GenerationForm />
      </div>
    </div>
  )
}
export const Route = createFileRoute('/_authenticated/generate')({ component: GeneratePage })
