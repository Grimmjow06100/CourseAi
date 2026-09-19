import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { GenerationForm } from '@/features/generation/generation-form'
import { PageHeader } from '@/shared/ui/page'
import { Card, CardContent } from '@/components/ui/card'

function GeneratePage() {
  const { t } = useTranslation()
  return (
    <div className="mx-auto max-w-[800px]">
      <PageHeader title={t('generation.title')} description={t('generation.subtitle')} />
      <Card className="shadow-none">
        <CardContent>
          <GenerationForm />
        </CardContent>
      </Card>
      <p className="mt-5 text-sm leading-6 text-muted-foreground">{t('design.guidanceText')}</p>
    </div>
  )
}
export const Route = createFileRoute('/_authenticated/generate')({ component: GeneratePage })
