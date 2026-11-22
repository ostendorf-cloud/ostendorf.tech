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
      expect(text).toContain(
        locale === 'de' ? 'noch bis zu 15 Tage' : 'for up to a further 15 days',
      )
      expect(text).not.toMatch(/in der regel noch bis|typically remains/i)
      expect(text).toContain(locale === 'de' ? 'Art. 6 Abs. 1 lit. f DSGVO' : 'Art. 6(1)(f) GDPR')
      expect(text).toContain(
        locale === 'de' ? 'vorgeschaltete Ingress' : 'the ingress in front of the application',
      )
      expect(text).not.toMatch(/beim aufruf\s+jeder seite|when you load any page/i)
      expect(text).toContain(
        locale === 'de'
          ? 'Betrieb der Website sicherzustellen, Fehler zu analysieren und Missbrauch abzuwehren'
          : 'to keep the website running, to analyse errors and to defend against misuse',
      )
      expect(text).toContain(
        locale === 'de'
          ? 'um zu verstehen, wie die Website genutzt wird'
          : 'to understand how the website is used',
      )
      expect(text).toContain(
        locale === 'de'
          ? 'gegen Ihre Rechte und Interessen abgewogen; die Daten werden auf das dafür Erforderliche beschränkt'
          : 'weighed this interest against your rights and interests; the data is limited to what this requires',
      )
      expect(text).toContain(
        locale === 'de'
          ? 'Die einzige Ausnahme von diesen 30 Tagen'
          : 'The only exception to these 30 days',
      )
      expect(text).not.toMatch(/hetzner|gunzenhausen|rechenzentrum|data centre/i)
      expect(text).not.toMatch(
        /wenn die statistik|sofern die statistik|eingeschaltet|if set up|switched on|when the statistics are/i,
      )
    })

    it('describes the stateless contact form and no cookie', async () => {
      const { container } = await renderPage(locale, 'privacy')
      const text = (container.textContent ?? '').replace(/\s+/g, ' ')

      // govue's contact form sets no cookie and keeps no session.
      expect(text).not.toMatch(/__Host-session|Sitzungs-Cookie|session cookie|SameSite|HttpOnly/i)
      expect(text).toContain(
        locale === 'de'
          ? 'Diese Website setzt keine Cookies, auch nicht beim Kontaktformular.'
          : 'This website sets no cookies, not even for the contact form.',
      )
      expect(text).toContain(
        locale === 'de'
          ? 'Kurz vorab: Diese Website setzt keine Cookies'
          : 'A brief summary up front: this website sets no cookies',
      )
      // The localStorage text of the colour scheme stays.
      expect(text).toContain('govue-theme')

      for (const fact of locale === 'de'
        ? [
            'aber weder Ihre IP-Adresse noch Angaben zu Ihrem Browser',
            'diese Angabe wird nicht an unseren Server übertragen',
            'Unser Server speichert die ausgegebenen Aufgaben nicht.',
            'bis die Aufgabe abläuft, höchstens 10 Minuten',
            'IP-Adresse (bei IPv6 die ersten 64 Bit)',
            'nie Ihren Namen, Ihre E-Mail-Adresse oder Ihre Nachricht',
            'für die beiden Schritte des Kontaktformulars und für Anfragen an Adressen',
          ]
        : [
            'but neither your IP address nor any details about your browser',
            'this information is not transmitted to our server',
            'Our server does not store the tasks it issues.',
            'until the task expires, after 10 minutes at most',
            'IP address (for IPv6, its first 64 bits)',
            'never your name, your email address or your message',
            'to the two steps of the contact form, and to requests for addresses',
          ]) {
        expect(text).toContain(fact)
      }
    })

    it('states which version is binding, once, in the language it belongs to', async () => {
      for (const page of SITE_LEGAL_PAGES) {
        const { container, unmount } = await renderPage(locale, page)
        const text = (container.textContent ?? '').replace(/\s+/g, ' ')
        const german = text.split('Rechtlich verbindlich ist die deutsche Fassung.').length - 1
        const english = text.split('The German version is the legally binding one.').length - 1

        expect(german, `${page} German notice`).toBe(locale === 'de' ? 1 : 0)
        expect(english, `${page} English notice`).toBe(locale === 'en' ? 1 : 0)
        unmount()
      }
    })

    it('lists the application log and the ingress log fields exactly', async () => {
      const { container } = await renderPage(locale, 'privacy')
      const norm = (s: string | null | undefined) => (s ?? '').replace(/\s+/g, ' ').trim()
      const heading = locale === 'de' ? 'Server-Log-Dateien' : 'Server log files'
      const start = Array.from(container.querySelectorAll('h3')).find(
        (h) => norm(h.textContent) === heading,
      )
      if (!start) throw new Error(`heading ${heading} not found`)

      const lists: string[][] = []
      let sectionText = ''
      for (let el = start.nextElementSibling; el && el.tagName !== 'H3';) {
        sectionText += ' ' + norm(el.textContent)
        if (el.tagName === 'UL') {
          lists.push(Array.from(el.querySelectorAll('li')).map((li) => norm(li.textContent)))
        }
        el = el.nextElementSibling
      }

      const appLog =
        locale === 'de'
          ? [
              'Zeitpunkt der Anfrage',
              'IP-Adresse',
              'eine zufällige, von unserem Server vergebene Anfrage-Kennung, mit der sich zusammengehörige Protokollzeilen einer einzelnen Anfrage zuordnen lassen',
              'Anfragemethode (z. B. GET oder POST)',
              'was angefragt wurde, als internes Muster: bei einem Seitenaufruf, welche Seite in welcher Sprache (z. B. „das Impressum auf Deutsch“), bei allen übrigen Anfragen die jeweilige Art der Anfrage (z. B. eine Design- oder Bilddatei, die Sitemap, eine Anfrage des Kontaktformulars). In keinem Fall die vollständige aufgerufene Adresse mit allen enthaltenen Parametern',
              'HTTP-Statuscode und übertragene Datenmenge',
              'Bearbeitungsdauer der Anfrage',
            ]
          : [
              'the time of the request',
              'the IP address',
              'a random request id our server assigns, so that log lines belonging to one request can be matched to each other',
              'the request method (for example, GET or POST)',
              'what was requested, as an internal pattern: for a page view, which page in which language (for example "the imprint in German"); for every other request, the kind of request it was (for example a design or image file, the sitemap, a contact form request). Never the full address requested with all its parameters',
              'the HTTP status code and the amount of data transferred',
              'how long the request took to answer',
            ]
      const ingressLog =
        locale === 'de'
          ? [
              'IP-Adresse',
              'Datum und Uhrzeit der Serveranfrage',
              'Anfragezeile (aufgerufene Seite bzw. Datei mit etwaigen Parametern, Anfragemethode und Protokollversion)',
              'HTTP-Statuscode und übertragene Datenmenge',
              'Browsertyp und Browserversion sowie verwendetes Betriebssystem (User-Agent)',
              'Referrer URL, sofern Ihr Browser sie übermittelt',
            ]
          : [
              'the IP address',
              'the date and time of the server request',
              'the request line (page or file requested, with any parameters, request method and protocol version)',
              'the HTTP status code and the amount of data transferred',
              'browser type and version, and operating system used (user agent)',
              'the referrer URL, where your browser transmits it',
            ]

      expect(lists).toEqual([appLog, ingressLog])
      expect(lists[0].join(' ')).not.toMatch(/user.?agent|referrer/i)
      expect(sectionText).toContain(
        locale === 'de'
          ? 'Nicht erfasst werden im Zugriffsprotokoll der Anwendung der Browsertyp Ihres Geräts (User-Agent) und eine etwaige Referrer-URL.'
          : "Your device's browser type (user agent) and any referrer URL are not recorded in the application access log.",
      )
      expect(sectionText).not.toMatch(/anonym/i)
    })

    it('names own hardware in the headline wording and does not call data anonymous', async () => {
      const { container } = await renderPage(locale, 'privacy')
      const text = (container.textContent ?? '').replace(/\s+/g, ' ')

      expect(text).toContain(
        locale === 'de' ? 'Hosting auf eigener Hardware' : 'Hosting on our own hardware',
      )
      expect(text).not.toMatch(/eigenen Servern|our own servers/i)
      expect(text).not.toMatch(/jeden Seitenaufruf|every page view|jeden Aufruf einer Seite/i)
      expect(text).toContain(locale === 'de' ? 'zählt Seitenaufrufe' : 'counts page views')
      expect(text).toContain(
        locale === 'de' ? 'die Kennung wechselt jeden Monat' : 'the identifier changes every month',
      )
      expect(text).not.toMatch(/deshalb jeden Monat|identifier therefore changes/i)
      expect(text).not.toMatch(/anonym/i)
    })

    it('states the dispute resolution position and the editorial responsibility in the imprint', async () => {
      const { container } = await renderPage(locale, 'imprint')
      const text = (container.textContent ?? '').replace(/\s+/g, ' ')

      if (locale === 'de') {
        expect(text).toContain('Streitbeilegung')
        expect(text).toContain(
          'zur Teilnahme an einem Streitbeilegungsverfahren vor einer Verbraucherschlichtungsstelle weder verpflichtet noch bereit',
        )
        expect(text).toContain('Redaktionell verantwortlich nach § 18 Abs. 2 MStV')
      } else {
        expect(text).toContain('Dispute resolution')
        expect(text).toContain(
          'neither obliged nor willing to take part in dispute resolution proceedings before a consumer arbitration board',
        )
        expect(text).toContain('Editorially responsible under Section 18(2)')
      }
    })
  })

  it('says on the English pages that they are a convenience translation', async () => {
    for (const page of SITE_LEGAL_PAGES) {
      const { container, unmount } = await renderPage('en', page)
      expect(container.textContent).toContain('This English text is a convenience translation.')
      expect(container.textContent).toContain('The German version is the legally binding one.')
      expect(container.textContent).not.toContain('only available in German')
      unmount()
    }
  })
})
