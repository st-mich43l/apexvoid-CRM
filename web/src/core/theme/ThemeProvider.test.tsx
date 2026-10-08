import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { ThemeProvider } from './ThemeProvider'
import { useTheme } from './context'

function ThemeProbe() {
  const { preference, resolvedTheme, setPreference } = useTheme()
  return <div><span>{preference}</span><span>{resolvedTheme}</span><button onClick={() => setPreference('light')}>Light</button></div>
}

describe('ThemeProvider', () => {
  beforeEach(() => window.localStorage.clear())

  it('persists a selected theme and applies semantic document state', () => {
    render(<ThemeProvider><ThemeProbe /></ThemeProvider>)
    fireEvent.click(screen.getByRole('button', { name: 'Light' }))
    expect(screen.getAllByText('light')).toHaveLength(2)
    expect(window.localStorage.getItem('apexvoid.theme')).toBe('light')
    expect(document.documentElement.dataset.theme).toBe('light')
    expect(document.documentElement).not.toHaveClass('dark')
  })
})
