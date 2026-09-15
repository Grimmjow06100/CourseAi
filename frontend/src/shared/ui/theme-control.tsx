import { Monitor, Moon, Sun } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { getTheme, saveTheme, type Theme } from '@/shared/lib/theme'
import { Button } from './button'

const nextTheme: Record<Theme, Theme> = { system: 'light', light: 'dark', dark: 'system' }
const icons = { system: Monitor, light: Sun, dark: Moon }

export function ThemeControl() {
  const { t } = useTranslation()
  const [theme, setTheme] = useState(getTheme)
  const Icon = icons[theme]
  const label = t('design.themeSwitch', {
    current: t(`design.${theme}`),
    next: t(`design.${nextTheme[theme]}`),
  })
  return (
    <Button
      variant="icon"
      aria-label={label}
      title={label}
      onClick={() => {
        const next = nextTheme[theme]
        saveTheme(next)
        setTheme(next)
      }}
    >
      <Icon className="size-4" aria-hidden="true" />
    </Button>
  )
}
