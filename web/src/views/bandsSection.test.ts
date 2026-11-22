import { render, screen } from '@testing-library/vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'

import { bands } from '../data/bands'
import { createAppI18n } from '../i18n'
import type { Locale } from '../i18n'
import HomeView from './HomeView.vue'

async function renderHome(locale: Locale) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/:locale?/privacy', name: 'privacy', component: { render: () => null } },
      { path: '/:pathMatch(.*)*', component: { render: () => null } },
    ],
  })
  await router.push('/')
  return render(HomeView, { global: { plugins: [createAppI18n(locale), router] } })
}

describe('the bands section logos', () => {
  for (const [locale, alt] of [
    ['de', 'Logo von Herrenjedeck'],
    ['en', 'Herrenjedeck logo'],
  ] as const) {
    it(`shows the Herrenjedeck logo with alt text, size and lazy loading in ${locale}`, async () => {
      const { container } = await renderHome(locale)

      const logo = screen.getByRole('img', { name: alt })
      expect(logo).toHaveAttribute('width', '480')
      expect(logo).toHaveAttribute('height', '489')
      expect(logo).toHaveAttribute('loading', 'lazy')
      expect(logo.closest('[aria-roledescription="slide"]')).toHaveTextContent('Herrenjedeck')
      expect(container.querySelectorAll('.band-logo')).toHaveLength(1)
    })
  }

  it('renders no logo for the six bands that have none', async () => {
    const { container } = await renderHome('de')

    const slides = container.querySelectorAll('[aria-roledescription="slide"]')
    expect(slides).toHaveLength(bands.length)
    bands.forEach((band, index) => {
      expect(slides[index].querySelector('.band-logo') !== null).toBe(band.logo !== undefined)
    })
    expect(bands.filter((band) => band.logo === undefined)).toHaveLength(6)
  })
})
