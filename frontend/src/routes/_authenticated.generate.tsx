import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { GenerationForm } from '@/features/generation/generation-form'
import { PageHeader } from '@/shared/ui/page'
import { BookOpen, Lightbulb, Target, Telescope } from 'lucide-react'

function GeneratePage() {
  const { t } = useTranslation()
  return (
    <div>
      <PageHeader
        eyebrow={t('dashboard.eyebrow')}
        title={t('generation.title')}
        description={t('generation.subtitle')}
      />
      <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
        <div className="studio-panel p-5 sm:p-7">
          <GenerationForm />
        </div>
        <aside className="rounded-2xl border border-border bg-accent-soft/40 p-6">
          <Lightbulb className="mb-4 size-6 text-warning-strong" />
          <h2 className="text-lg font-bold">{t('design.guidance')}</h2>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">{t('design.guidanceText')}</p>
          <ul className="mt-6 space-y-6">
            {[
              { key: 'guidanceSubject', icon: Telescope },
              { key: 'guidanceLevel', icon: BookOpen },
              { key: 'guidanceGoal', icon: Target },
            ].map(({ key, icon: Icon }) => (
              <li key={key} className="flex gap-3">
                <Icon className="mt-1 size-4 shrink-0 text-primary" />
                <div>
                  <h3 className="text-sm font-bold">{t(`design.${key}`)}</h3>
                  <p className="mt-1 text-xs leading-5 text-muted-foreground">{t(`design.${key}Text`)}</p>
                </div>
              </li>
            ))}
          </ul>
        </aside>
      </div>
    </div>
  )
}
export const Route = createFileRoute('/_authenticated/generate')({ component: GeneratePage })
