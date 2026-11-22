import { render, screen, waitFor } from '@testing-library/vue'
import { afterEach, describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

import { createAppI18n } from '../i18n'
import type { Locale } from '../i18n'
import { impressionsOnPage } from '../lib/pageSections'
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

afterEach(() => {
  impressionsOnPage.value = false
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

  it('goes away with the section', async () => {
    const view = await renderPage('de', () => Promise.resolve({ posts }))
    await waitFor(() => {
      expect(impressionsOnPage.value).toBe(true)
    })

    view.unmount()

    expect(impressionsOnPage.value).toBe(false)
  })
})
