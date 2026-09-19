import { Monitor, Moon, Sun } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useTheme } from '@/shared/hooks/use-theme'
import { saveTheme } from '@/shared/lib/theme'

const icons = { system: Monitor, light: Sun, dark: Moon }

export function ThemeControl() {
  const { t } = useTranslation()
  const { theme } = useTheme()
  const Icon = icons[theme]
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label={t('common.theme')} title={t('common.theme')}>
          <Icon className="size-4" aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        <DropdownMenuRadioGroup
          value={theme}
          onValueChange={(value) => {
            if (value === 'system' || value === 'light' || value === 'dark') saveTheme(value)
          }}
        >
          {(['system', 'light', 'dark'] as const).map((value) => {
            const ThemeIcon = icons[value]
            return (
              <DropdownMenuRadioItem key={value} value={value}>
                <ThemeIcon className="mr-2 size-4" aria-hidden="true" />
                {t(`design.${value}`)}
              </DropdownMenuRadioItem>
            )
          })}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
