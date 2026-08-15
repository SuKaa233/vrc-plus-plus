export type Theme = 'light' | 'dark'

export const THEME_STORAGE_KEY = 'vrc-harbor-theme'

export function resolveTheme(savedTheme: string | null, prefersLight: boolean): Theme {
  if (savedTheme === 'light' || savedTheme === 'dark') return savedTheme
  return prefersLight ? 'light' : 'dark'
}

export function readTheme(): Theme {
  let savedTheme: string | null = null
  try {
    savedTheme = localStorage.getItem(THEME_STORAGE_KEY)
  } catch {
    // Browsers with storage disabled can still use the system theme.
  }
  return resolveTheme(savedTheme, window.matchMedia('(prefers-color-scheme: light)').matches)
}

export function applyTheme(theme: Theme, persist = true) {
  document.documentElement.dataset.theme = theme
  document.documentElement.style.colorScheme = theme
  document.querySelector('meta[name="theme-color"]')?.setAttribute(
    'content',
    theme === 'light' ? '#f4f5f7' : '#111318',
  )

  if (!persist) return
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme)
  } catch {
    // Theme switching should remain usable even when storage is unavailable.
  }
}
