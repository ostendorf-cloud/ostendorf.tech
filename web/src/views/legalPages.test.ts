import { render, screen } from '@testing-library/vue'
import { createGovueI18n, legalPath } from '@ostendorf-cloud/govue'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import App from '../App.vue'
import { createAppI18n, type Locale } from '../i18n'
import { SITE_LEGAL_PAGES, siteRouteDefs } from '../lib/legal'
import { routeTable } from '../lib/routeTable'
import { attachLocaleSync, createAppRouter } from '../router'

const LOCALES: Locale[] = ['de', 'en']

async function renderPage(locale: Locale, page: string) {
  const i18n = createAppI18n(locale)
  const router = createAppRouter(createMemoryHistory())
  attachLocaleSync(router, i18n.global.locale)
  await router.push({ name: page, params: { locale } })
  await router.isReady()
  return render(App, {
    global: { plugins: [router, i18n, createGovueI18n({ i18n, locale })] },
  })
}

describe('legal pages', () => {
  it('serves the pages govue names, and the route table agrees', () => {
    const slugs = routeTable.map((entry) => entry.slug).filter((slug) => slug !== '')
    expect(slugs).toEqual([...SITE_LEGAL_PAGES])
    expect(siteRouteDefs.map((def) => def.path)).toEqual(['/', ...slugs.map((s) => `/${s}`)])
  })

  describe.each(LOCALES)('in %s', (locale) => {
    it.each([...SITE_LEGAL_PAGES])('renders %s with one h1 and the footer links', async (page) => {
      const { container } = await renderPage(locale, page)

      expect(container.querySelectorAll('h1')).toHaveLength(1)
      const footer = screen.getByRole('contentinfo')
      for (const legal of SITE_LEGAL_PAGES) {
        const link = footer.querySelector(`a[href="${legalPath(legal, locale)}"]`)
        expect(link, `footer link to ${legalPath(legal, locale)}`).not.toBeNull()
      }
    })

    it('states the operator facts in the imprint', async () => {
      const { container } = await renderPage(locale, 'imprint')
      const text = container.textContent ?? ''

      for (const fact of [
        'Simon Ostendorf',
        'Frankenstr. 8',
        '52070 Aachen',
        '+49 1573 0971784',
        'simon@ostendorf.tech',
        'DE362164357',
      ]) {
        expect(text).toContain(fact)
      }
    })

    it('names own hardware, no hosting provider and states the counting plainly', async () => {
      const { container } = await renderPage(locale, 'privacy')
      const text = (container.textContent ?? '').replace(/\s+/g, ' ')

      expect(text).toContain(locale === 'de' ? 'eigener Hardware' : 'own hardware')
      expect(text).toContain(locale === 'de' ? 'nach 30 Tagen' : 'after 30 days')
      expect(text).toContain(locale === 'de' ? 'nach 14 Monaten' : 'after 14 months')
      expect(text).not.toMatch(/hetzner|gunzenhausen|rechenzentrum|data centre/i)
      expect(text).not.toMatch(
        /wenn die statistik|sofern die statistik|eingeschaltet|if set up|switched on|when the statistics are/i,
      )
    })

    it('has no consumer dispute resolution section in the imprint', async () => {
      const { container } = await renderPage(locale, 'imprint')
      const text = container.textContent ?? ''

      expect(text).not.toMatch(
        /streitbeilegung|schlichtung|dispute resolution|arbitration|odr-plattform/i,
      )
    })
  })

  it('keeps the German binding notice on the English pages', async () => {
    for (const page of SITE_LEGAL_PAGES) {
      const { container, unmount } = await renderPage('en', page)
      expect(container.textContent).toContain('The German version is the legally binding one.')
      unmount()
    }
  })
})
