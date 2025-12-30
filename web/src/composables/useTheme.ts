import { ref, computed, watchEffect } from 'vue'

export type Theme = 'dark' | 'light' | 'system'
export type ResolvedTheme = Exclude<Theme, 'system'>

const THEME_STORAGE_KEY = 'xvault-theme'

// Global reactive state
const theme = ref<Theme>((localStorage.getItem(THEME_STORAGE_KEY) as Theme) || 'system')

// Compute the resolved theme (actual light/dark)
const resolvedTheme = computed<ResolvedTheme>(() => {
  if (theme.value === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return theme.value
})

// Apply theme to document
function applyTheme(resolved: ResolvedTheme) {
  const root = window.document.documentElement
  root.classList.remove('light', 'dark')
  root.classList.add(resolved)
}

// Watch for theme changes
watchEffect(() => {
  applyTheme(resolvedTheme.value)
})

// Listen for system preference changes
if (typeof window !== 'undefined') {
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', () => {
    if (theme.value === 'system') {
      applyTheme(mediaQuery.matches ? 'dark' : 'light')
    }
  })
}

export function useTheme() {
  const setTheme = (newTheme: Theme) => {
    theme.value = newTheme
    localStorage.setItem(THEME_STORAGE_KEY, newTheme)
    applyTheme(resolvedTheme.value)
  }

  const toggleTheme = () => {
    const nextTheme = resolvedTheme.value === 'dark' ? 'light' : 'dark'
    setTheme(nextTheme)
  }

  return {
    theme,
    resolvedTheme,
    setTheme,
    toggleTheme,
  }
}
