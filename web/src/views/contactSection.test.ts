/**
 * The contact section renders govue's contact form with the site's own
 * consent text and the privacy link of the page's language. govue's own
 * tests cover the form itself: its fields, the proof of work, the request
 * body and the errors.
 */
import { render, screen } from '@testing-library/vue'
import { createGovueI18n } from '@ostendorf-cloud/govue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import App from '../App.vue'
import { createAppI18n, type Locale } from '../i18n'
import de from '../i18n/locales/de.json'
import en from '../i18n/locales/en.json'
import { attachLocaleSync, createAppRouter } from '../router'

const CONSENT: Record<Locale, string> = {
  de: 'Ich bin damit einverstanden, dass die angegebenen Daten gemäß der Datenschutzerklärung zur Bearbeitung meiner Anfrage gespeichert werden dürfen.',
  en: 'I agree that the data I provide may be stored under the privacy policy to process my request.',
}

async function renderHome(locale: Locale) {
  const i18n = createAppI18n(locale)
  const router = createAppRouter(createMemoryHistory())
  attachLocaleSync(router, i18n.global.locale)
  await router.push({ name: 'home', params: { locale } })
  await router.isReady()
  return render(App, {
    global: { plugins: [router, i18n, createGovueI18n({ i18n, locale })] },
  })
}

beforeEach(() => {
  // The Impressionen section asks the feed route on mount. An empty feed
  // renders no section, which is all this test needs.
  vi.stubGlobal(
    'fetch',
    vi.fn(() =>
      Promise.resolve(
        new Response(JSON.stringify({ posts: [] }), {
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    ),
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe.each(['de', 'en'] as const)('the contact section in %s', (locale) => {
  it("renders govue's form with the site's consent text and privacy link", async () => {
    const { container } = await renderHome(locale)
    const section = container.querySelector('#contact')
    expect(section).not.toBeNull()

    // govue's form: its fields, labelled from govue's catalogue.
    const form = section?.querySelector('form')
    expect(form).not.toBeNull()
    for (const name of ['name', 'email', 'message', 'website']) {
      expect(form?.querySelector(`[name="${name}"]`), name).not.toBeNull()
    }

    // The site's consent text, as the label of the required checkbox.
    expect(CONSENT[locale]).toBe(
      locale === 'de' ? de.home.contact.consent : en.home.contact.consent,
    )
    const consent = screen.getByRole('checkbox', { name: CONSENT[locale] })
    expect(form?.contains(consent)).toBe(true)

    // The privacy link of the page's own language.
    const privacy = form?.querySelector(`a[href="/${locale}/privacy"]`)
    expect(privacy, `/${locale}/privacy`).not.toBeNull()

    // The site's heading of the form.
    expect(section?.querySelector('h3')?.textContent?.trim()).toBe(
      locale === 'de' ? 'Nachricht senden' : 'Send a message',
    )
  })
})
