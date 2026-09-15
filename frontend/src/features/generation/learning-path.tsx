import { BookOpen, Compass, Sparkles, Target } from 'lucide-react'
import { useTranslation } from 'react-i18next'

const steps = [Target, Compass, BookOpen]

export function LearningPath() {
  const { t } = useTranslation()
  return (
    <aside className="studio-path flex flex-col rounded-2xl p-6 sm:p-8">
      <div className="flex items-center gap-2 text-[10px] font-bold tracking-[.15em] text-[#c7dfb5]">
        <Sparkles className="size-4" aria-hidden="true" />
        {t('design.pathEyebrow')}
      </div>
      <h2 className="mt-7 whitespace-pre-line text-2xl font-bold leading-snug tracking-tight">
        {t('design.pathTitle')}
      </h2>
      <p className="mt-3 text-sm leading-6 text-[#c7d8cd]">{t('design.pathDescription')}</p>
      <ol className="mt-8 space-y-6">
        {steps.map((Icon, index) => (
          <li key={index} className="flex items-start gap-3">
            <span className="grid size-10 shrink-0 place-items-center rounded-xl border border-white/15 bg-white/5 text-[#c7dfb5]">
              <Icon className="size-4" aria-hidden="true" />
            </span>
            <div>
              <h3 className="text-sm font-bold">{t(`design.step${index + 1}`)}</h3>
              <p className="mt-1 text-xs leading-5 text-[#c7d8cd]">{t(`design.step${index + 1}Text`)}</p>
            </div>
          </li>
        ))}
      </ol>
    </aside>
  )
}
