export type Theme = 'system' | 'light' | 'dark'

const themeEvent = 'course-ai:theme'
let volatileTheme: Theme | undefined

export function getTheme(): Theme {
  try {
    const value = localStorage.getItem('course-ai-theme')
    return value === 'light' || value === 'dark' ? value : 'system'
  } catch {
    return volatileTheme ?? 'system'
  }
}

export function applyTheme(theme: Theme) {
  const dark =
    theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.classList.toggle('dark', dark)
  document.documentElement.removeAttribute('data-theme')
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', dark ? '#18181b' : '#ffffff')
  window.dispatchEvent(new Event(themeEvent))
}

export function saveTheme(theme: Theme) {
  volatileTheme = theme
  try {
    localStorage.setItem('course-ai-theme', theme)
  } catch {
    // The current page still uses the chosen theme when storage is unavailable.
  }
  applyTheme(theme)
}

export function initializeTheme() {
  applyTheme(getTheme())
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  const sync = () => applyTheme(getTheme())
  media.addEventListener('change', sync)
  window.addEventListener('storage', sync)
  return () => {
    media.removeEventListener('change', sync)
    window.removeEventListener('storage', sync)
  }
}

export function subscribeTheme(listener: () => void) {
  window.addEventListener(themeEvent, listener)
  return () => window.removeEventListener(themeEvent, listener)
}
