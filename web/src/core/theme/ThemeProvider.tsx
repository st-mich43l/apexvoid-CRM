import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { ThemeContext, type ResolvedTheme, type ThemePreference } from './context'

const storageKey = 'apexvoid.theme'

function readPreference(): ThemePreference {
  const value = window.localStorage.getItem(storageKey)
  return value === 'light' || value === 'dark' || value === 'system' ? value : 'system'
}

function resolve(preference: ThemePreference): ResolvedTheme {
  if (preference !== 'system') return preference
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function applyTheme(theme: ResolvedTheme) {
  document.documentElement.classList.toggle('dark', theme === 'dark')
  document.documentElement.dataset.theme = theme
  document.documentElement.style.colorScheme = theme
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', theme === 'dark' ? '#0a0a0b' : '#f5f6fa')
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [preference, setPreferenceState] = useState<ThemePreference>(readPreference)
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>(() => resolve(preference))

  const setPreference = useCallback((next: ThemePreference) => {
    window.localStorage.setItem(storageKey, next)
    setPreferenceState(next)
    const resolved = resolve(next)
    setResolvedTheme(resolved)
    applyTheme(resolved)
  }, [])

  const toggle = useCallback(() => {
    setPreference(resolvedTheme === 'dark' ? 'light' : 'dark')
  }, [resolvedTheme, setPreference])

  useEffect(() => {
    applyTheme(resolvedTheme)
    if (preference !== 'system') return
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const update = () => {
      const next = resolve('system')
      setResolvedTheme(next)
      applyTheme(next)
    }
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [preference, resolvedTheme])

  const value = useMemo(() => ({ preference, resolvedTheme, setPreference, toggle }), [preference, resolvedTheme, setPreference, toggle])
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}
