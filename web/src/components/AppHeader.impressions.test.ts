import { render, screen, waitFor } from '@testing-library/vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

import { createAppI18n } from '../i18n'
import type { Locale } from '../i18n'
import { IMPRESSIONS_HIDDEN_QUERY, impressionsOnPage } from '../lib/pageSections'
import AppHeader from './AppHeader.vue'
import InstagramFeed from './InstagramFeed.vue'

const posts = [{ id: 'a1', alt: 'Lichtpult im Saal', permalink: 'https://www.instagram.com/p/A/' }]

/** The header and the section side by side, as App.vue and HomeView.vue put them. */
async function renderPage(locale: Locale, answer: () => Promise<unknown>) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:locale', name: 'home', component: { template: '<div />' } }],
  })
  await router.push(`/${locale}`)
  await router.isReady()
  const client = { get: <T>() => answer() as Promise<T> }
  const page = defineComponent({
    render: () => [h(AppHeader), h(InstagramFeed, { client })],
  })
  return render(page, { global: { plugins: [createAppI18n(locale), router] } })
}

function navLabels(): string[] {
  const links = screen
    .getByRole('navigation', { name: /Hauptmenü|Main menu/ })
    .querySelectorAll('li a')
  return Array.from(links, (a) => (a.textContent ?? '').trim())
}

/** A viewport of the given width, as far as the header's one media query goes:
 *  true while the width is in the 640 to 1023 px range. Returns a function
 *  that resizes it and tells the listeners. */
function stubViewport(width: number): (next: number) => void {
  let current = width
  const listeners = new Set<() => void>()
  const inRange = (): boolean => current >= 640 && current <= 1023
  vi.stubGlobal('matchMedia', (query: string) => ({
    get matches() {
      return query === IMPRESSIONS_HIDDEN_QUERY && inRange()
    },
    addEventListener: (_: string, listener: () => void) => listeners.add(listener),
    removeEventListener: (_: string, listener: () => void) => listeners.delete(listener),
  }))
  return (next) => {
    current = next
    listeners.forEach((listener) => {
      listener()
    })
  }
}

afterEach(() => {
  impressionsOnPage.value = false
  vi.unstubAllGlobals()
})

describe('the Impressionen entry in the header', () => {
  it('is absent while the feed is empty', async () => {
    const { container } = await renderPage('de', () => Promise.resolve({ posts: [] }))

    await waitFor(() => {
      expect(container.querySelector('#impressions')).toBeNull()
    })
    expect(navLabels()).toEqual(['Leistungen', 'Bands', 'Über mich', 'Kontakt'])
    expect(screen.queryByRole('link', { name: 'Impressionen' })).toBeNull()
  })

  it('is absent when the feed request fails', async () => {
    await renderPage('de', () => Promise.reject(new Error('down')))

    await waitFor(() => {
      expect(impressionsOnPage.value).toBe(false)
    })
    expect(navLabels()).toEqual(['Leistungen', 'Bands', 'Über mich', 'Kontakt'])
  })

  it('appears between Leistungen and Bands once the section exists, in German', async () => {
    const { container } = await renderPage('de', () => Promise.resolve({ posts }))

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Impressionen' })).toBeInTheDocument()
    })
    expect(navLabels()).toEqual(['Leistungen', 'Impressionen', 'Bands', 'Über mich', 'Kontakt'])

    const link = screen.getByRole('link', { name: 'Impressionen' })
    expect(link.getAttribute('href')).toBe('/de#impressions')
    // The id it points at is on the page.
    expect(container.querySelector('#impressions')).not.toBeNull()
  })

  it('appears in English with its own label and the English path', async () => {
    const { container } = await renderPage('en', () => Promise.resolve({ posts }))

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Impressions' })).toBeInTheDocument()
    })
    expect(navLabels()).toEqual(['Services', 'Impressions', 'Bands', 'About me', 'Contact'])
    expect(screen.getByRole('link', { name: 'Impressions' }).getAttribute('href')).toBe(
      '/en#impressions',
    )
    expect(container.querySelector('#impressions')).not.toBeNull()
  })

  const withEntry = ['Leistungen', 'Impressionen', 'Bands', 'Über mich', 'Kontakt']
  const withoutEntry = ['Leistungen', 'Bands', 'Über mich', 'Kontakt']

  for (const width of [400, 1024, 1440]) {
    it(`is in the list at ${width} px: govue's menu below 640, the row from 1024`, async () => {
      stubViewport(width)
      await renderPage('de', () => Promise.resolve({ posts }))

      await waitFor(() => {
        expect(screen.getByRole('link', { name: 'Impressionen' })).toBeInTheDocument()
      })
      expect(navLabels()).toEqual(withEntry)
    })
  }

  for (const width of [640, 768, 900, 1023]) {
    it(`is not in the row at ${width} px, though the section is on the page`, async () => {
      stubViewport(width)
      const { container } = await renderPage('de', () => Promise.resolve({ posts }))

      await waitFor(() => {
        expect(container.querySelector('#impressions')).not.toBeNull()
      })
      expect(navLabels()).toEqual(withoutEntry)
    })
  }

  it('follows a resize across the range', async () => {
    const resize = stubViewport(1280)
    await renderPage('de', () => Promise.resolve({ posts }))
    await waitFor(() => {
      expect(navLabels()).toEqual(withEntry)
    })

    resize(800)
    await waitFor(() => {
      expect(navLabels()).toEqual(withoutEntry)
    })

    resize(1100)
    await waitFor(() => {
      expect(navLabels()).toEqual(withEntry)
    })
  })

  it('hides the entry only between the sm and lg breakpoints', () => {
    expect(IMPRESSIONS_HIDDEN_QUERY).toBe('(min-width: 40rem) and (max-width: 63.9375rem)')
  })

  it('goes away with the section', async () => {
    const view = await renderPage('de', () => Promise.resolve({ posts }))
    await waitFor(() => {
      expect(impressionsOnPage.value).toBe(true)
    })

    view.unmount()

    expect(impressionsOnPage.value).toBe(false)
  })
})
