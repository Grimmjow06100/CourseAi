export type Theme = 'system' | 'light' | 'dark'

export function getTheme(): Theme {
  try {
    const value = localStorage.getItem('course-ai-theme')
    return value === 'light' || value === 'dark' ? value : 'system'
  } catch {
    return 'system'
  }
}

export function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme
}

export function saveTheme(theme: Theme) {
  applyTheme(theme)
  try {
    localStorage.setItem('course-ai-theme', theme)
  } catch {
    // The current page still uses the chosen theme when storage is unavailable.
  }
}
