import { render } from '@testing-library/vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'

import { serviceIcons } from '../data/services'
import { createAppI18n } from '../i18n'
import type { Locale } from '../i18n'
import de from '../i18n/locales/de.json'
import en from '../i18n/locales/en.json'
import HomeView from './HomeView.vue'

const catalogues = { de, en }

async function renderServices(locale: Locale) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/:locale?/privacy', name: 'privacy', component: { render: () => null } },
      { path: '/:pathMatch(.*)*', component: { render: () => null } },
    ],
  })
  await router.push('/')
  const { container } = render(HomeView, { global: { plugins: [createAppI18n(locale), router] } })
  const section = container.querySelector<HTMLElement>('#services')
  if (!section) throw new Error('the Leistungen section is missing')
  return section
}

describe('the Leistungen section', () => {
  for (const locale of ['de', 'en'] as const) {
    const texts = catalogues[locale].home.services

    describe(locale, () => {
      it('keeps its anchor and is labelled by its heading', async () => {
        const section = await renderServices(locale)

        expect(section.id).toBe('services')
        const title = section.querySelector('#services-title')
        expect(title?.tagName).toBe('H2')
        expect(section.getAttribute('aria-labelledby')).toBe('services-title')
      })

      it('renders one card per service with its icon, heading and text', async () => {
        const section = await renderServices(locale)

        const cards = section.querySelectorAll('.service-card')
        expect(cards).toHaveLength(6)
        expect(cards).toHaveLength(serviceIcons.length)
        cards.forEach((card, index) => {
          expect(card.querySelectorAll('svg')).toHaveLength(1)
          expect(card.querySelector('h4')).toHaveTextContent(texts.items[index].title)
          expect(card.querySelector('p')).toHaveTextContent(texts.items[index].text)
        })
      })

      it('steps the card grid from one to two to three columns', async () => {
        const section = await renderServices(locale)

        const grid = section.querySelector('.service-grid')
        expect(grid).toHaveClass('grid-cols-1', 'md:grid-cols-2', 'xl:grid-cols-3')
        expect(grid?.children).toHaveLength(6)
      })

      it('lists all ten areas as plain list items, not chips', async () => {
        const section = await renderServices(locale)

        const areas = section.querySelectorAll('.areas-list > li')
        expect(areas).toHaveLength(10)
        expect(Array.from(areas, (area) => area.textContent.trim())).toEqual(texts.areas)
        expect(section.querySelector('.areas-list')).toHaveClass(
          'grid-cols-1',
          'sm:grid-cols-2',
          'xl:grid-cols-3',
        )
        areas.forEach((area) => expect(area).not.toHaveClass('rounded-sm', 'bg-bg-subtle'))
      })

      it('keeps the group labels, the closing link and a valid heading order', async () => {
        const section = await renderServices(locale)

        const levels = Array.from(section.querySelectorAll('h1, h2, h3, h4, h5, h6'), (heading) =>
          Number(heading.tagName.slice(1)),
        )
        expect(levels).toEqual([2, 3, 4, 4, 4, 4, 4, 4, 3])
        const groups = Array.from(section.querySelectorAll('h3'), (h) => h.textContent.trim())
        expect(groups).toEqual([texts.panelTechnology, texts.panelAreas])
        expect(section.querySelector('a[href="mailto:simon@ostendorf.tech"]')).toHaveTextContent(
          texts.more,
        )
      })
    })
  }
})
