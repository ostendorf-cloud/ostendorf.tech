import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'

import { createAppI18n, type Locale } from '../i18n'
import { attachLocaleSync, createAppRouter } from '../router'
import LocaleSwitch from './LocaleSwitch.vue'

async function renderSwitch(locale: Locale) {
  const i18n = createAppI18n(locale)
  const router = createAppRouter(createMemoryHistory())
  attachLocaleSync(router, i18n.global.locale)
  await router.push({ name: 'home', params: { locale } })
  await router.isReady()
  return render(LocaleSwitch, { global: { plugins: [router, i18n] } })
}

describe('LocaleSwitch', () => {
  it.each([
    ['de', 'en'],
    ['en', 'de'],
  ] as const)('marks the link from %s as %s content', async (from, to) => {
    await renderSwitch(from)
    const link = screen.getByRole('link')

    expect(link.getAttribute('href')).toBe(`/${to}/`)
    expect(link.getAttribute('lang')).toBe(to)
    expect(link.getAttribute('hreflang')).toBe(to)
  })
})
