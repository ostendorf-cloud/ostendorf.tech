import { render } from '@testing-library/vue'
import { createGovueI18n } from '@ostendorf-cloud/govue'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import App from '../App.vue'
import { createAppI18n, type Locale } from '../i18n'
import { SITE_LEGAL_PAGES } from '../lib/legal'
import { attachLocaleSync, createAppRouter } from '../router'

// #app clips horizontal overflow, so a long URL on a 360 px phone is cut off unless the
// text may wrap anywhere. jsdom applies no stylesheet, so this checks the two halves
// separately: the rule exists, and its selector matches the main element of every legal page.
const WRAP_RULE = /main\.max-w-reading\s*\{[^}]*overflow-wrap:\s*anywhere\s*;[^}]*\}/

describe('long strings on the legal pages', () => {
  it('may wrap anywhere inside the reading width main', () => {
    // Read from disk: vitest hands a stylesheet import over as an empty string.
    const css = readFileSync(resolve(process.cwd(), 'src/styles.css'), 'utf8')
    expect(css).toMatch(WRAP_RULE)
  })

  describe.each(['de', 'en'] as Locale[])('in %s', (locale) => {
    it.each([...SITE_LEGAL_PAGES])('%s renders inside main.max-w-reading', async (page) => {
      const i18n = createAppI18n(locale)
      const router = createAppRouter(createMemoryHistory())
      attachLocaleSync(router, i18n.global.locale)
      await router.push({ name: page, params: { locale } })
      await router.isReady()
      const { container } = render(App, {
        global: { plugins: [router, i18n, createGovueI18n({ i18n, locale })] },
      })

      expect(container.querySelector('main.max-w-reading')).not.toBeNull()
    })
  })
})
