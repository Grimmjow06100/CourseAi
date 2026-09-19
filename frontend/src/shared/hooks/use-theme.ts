import { useSyncExternalStore } from 'react'
import { getTheme, subscribeTheme } from '@/shared/lib/theme'

export function useTheme() {
  const theme = useSyncExternalStore(subscribeTheme, getTheme, () => 'system' as const)
  const resolvedTheme = useSyncExternalStore(
    subscribeTheme,
    () => (document.documentElement.classList.contains('dark') ? 'dark' : 'light'),
    () => 'light' as const,
  )
  return { theme, resolvedTheme }
}
