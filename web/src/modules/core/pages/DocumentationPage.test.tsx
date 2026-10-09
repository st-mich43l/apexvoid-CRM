import { fireEvent, render, screen, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { DocumentationPage } from './DocumentationPage'
import { categories, guides, searchGuides } from './documentation/content'
import { coreModule } from '../index'

function renderDocs(path = '/docs') {
  return render(<MemoryRouter initialEntries={[path]}><Routes>
    <Route path="/docs" element={<DocumentationPage />} />
    <Route path="/docs/:slug" element={<DocumentationPage />} />
  </Routes></MemoryRouter>)
}

describe('Enterprise documentation center', () => {
  it('registers a public-to-members platform navigation entry and both routes', () => {
    expect(coreModule.navigation?.find(item => item.id === 'documentation')).toMatchObject({ path: '/docs', label: 'Documentation' })
    expect(coreModule.routes?.map(route => route.path)).toContain('docs')
    expect(coreModule.routes?.map(route => route.path)).toContain('docs/:slug')
  })

  it('renders the learning center with categorized, linked guides', () => {
    renderDocs()
    expect(screen.getByRole('heading', { name: /Build confidently on ApexVoid/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Get started/i })).toHaveAttribute('href', '/docs/local-setup')
    expect(screen.getByRole('navigation', { name: 'Build applications' })).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: /Build your first external app/i }).length).toBeGreaterThan(0)
  })

  it('opens a deep-linked article with section navigation and code samples', () => {
    renderDocs('/docs/manifest-and-enrollment')
    expect(screen.getByRole('heading', { level: 1, name: 'Manifest and secure enrollment' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Signed discovery and one-time code' })).toBeInTheDocument()
    expect(screen.getByText(/X-ApexVoid-Manifest-Signature/)).toBeInTheDocument()
    const toc = screen.getByRole('navigation', { name: 'Article sections' })
    expect(within(toc).getByRole('link', { name: 'Encrypted enrollment handoff' })).toHaveAttribute('href', '#handoff')
  })

  it('searches guide content, including troubleshooting details', () => {
    renderDocs()
    const search = screen.getByRole('searchbox', { name: 'Search documentation' })
    fireEvent.change(search, { target: { value: 'Firefox' } })
    expect(screen.getByRole('heading', { name: 'Search results' })).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: /Troubleshooting/i }).length).toBeGreaterThan(0)
    fireEvent.change(search, { target: { value: 'NO-SUCH-TOPIC-000' } })
    expect(screen.getByText('No matching guides')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Clear documentation search' }))
    expect(screen.getByRole('heading', { name: /Build confidently on ApexVoid/i })).toBeInTheDocument()
  })

  it('has stable unique guide/section slugs and relevant deep search results', () => {
    expect(new Set(guides.map(guide => guide.slug)).size).toBe(guides.length)
    expect(categories.length).toBe(4)
    for (const guide of guides) {
      expect(categories).toContain(guide.category)
      expect(guide.sections.length).toBeGreaterThan(0)
      expect(new Set(guide.sections.map(section => section.id)).size).toBe(guide.sections.length)
      expect(guide.related.every(slug => guides.some(other => other.slug === slug))).toBe(true)
    }
    expect(searchGuides('DATABASE_PROVISIONING_KEY').some(guide => guide.slug === 'local-setup')).toBe(true)
    expect(searchGuides('identity assertion').some(guide => guide.slug === 'security-and-rbac')).toBe(true)
  })

  it('provides an accessible fallback for unknown guides', () => {
    renderDocs('/docs/not-existing')
    expect(screen.getByRole('heading', { name: 'Guide not found' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Back to documentation/i })).toHaveAttribute('href', '/docs')
  })
})
