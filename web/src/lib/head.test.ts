import { computed, nextTick, ref } from 'vue'
import { beforeEach, describe, expect, it } from 'vitest'
import type { Composer } from 'vue-i18n'

import { createAppI18n, type Locale } from '../i18n'
import { usePageHead } from './head'
import { NOT_FOUND, canonical } from './seo'

function robots(): string | null {
  return document.querySelector('meta[name="robots"]')?.getAttribute('content') ?? null
}

function canonicalHref(): string | null {
  return document.querySelector('link[rel="canonical"]')?.getAttribute('href') ?? null
}

function ogUrl(): string | null {
  return document.querySelector('meta[property="og:url"]')?.getAttribute('content') ?? null
}

function alternates(): number {
  return document.querySelectorAll('link[rel="alternate"][hreflang]').length
}

describe('usePageHead', () => {
  beforeEach(() => {
    document.head.innerHTML = ''
  })

  it('keeps the 404 page out of the index and drops what points at another page', async () => {
    const route = ref<string>('home')
    const locale = ref<Locale>('de')
    usePageHead(
      computed(() => route.value),
      computed(() => locale.value),
      createAppI18n('de').global as unknown as Composer,
    )

    expect(robots()).toBe('index, follow')
    expect(canonicalHref()).toBe(canonical('de', 'home'))
    expect(ogUrl()).toBe(canonical('de', 'home'))
    expect(alternates()).toBeGreaterThan(0)

    route.value = NOT_FOUND
    await nextTick()

    expect(robots()).toBe('noindex')
    expect(canonicalHref()).toBeNull()
    expect(ogUrl()).toBeNull()
    expect(alternates()).toBe(0)

    route.value = 'home'
    await nextTick()

    expect(robots()).toBe('index, follow')
    expect(canonicalHref()).toBe(canonical('de', 'home'))
    expect(ogUrl()).toBe(canonical('de', 'home'))
    expect(alternates()).toBeGreaterThan(0)
  })
})
