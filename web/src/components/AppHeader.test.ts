import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { render, screen } from '@testing-library/vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import { createAppI18n } from '../i18n'
import { createGovueI18n } from '@ostendorf-cloud/govue'
import { HEADER_HEIGHT_PROPERTY } from '../lib/headerHeight'
import AppHeader from './AppHeader.vue'
import WordMark from './WordMark.vue'

function source(path: string): string {
  return readFileSync(join(process.cwd(), path), 'utf-8')
}

async function renderHeader(path = '/de') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/:locale', name: 'home', component: { template: '<div />' } },
      { path: '/:locale/imprint', name: 'imprint', component: { template: '<div />' } },
    ],
  })
  await router.push(path)
  await router.isReady()
  const locale = path.startsWith('/en') ? 'en' : 'de'
  const i18n = createAppI18n(locale)
  const view = render(AppHeader, {
    global: { plugins: [i18n, router, createGovueI18n({ i18n, locale, formality: 'formal' })] },
  })
  return { ...view, router }
}

describe('AppHeader on govue header', () => {
  it('puts the navigation at the end, next to the actions', async () => {
    const { container } = await renderHeader()
    const nav = screen.getByRole('navigation', { name: 'Hauptmenü' })
    expect(nav.closest('[class*="justify-end"]')).not.toBeNull()
    expect(container.querySelector('header')?.dataset.overlay).toBeUndefined()
  })

  it('makes the contact link the one call to action', async () => {
    const { container } = await renderHeader()
    const links = Array.from(
      screen.getByRole('navigation', { name: 'Hauptmenü' }).querySelectorAll('li a'),
    )
    const contact = links.find((a) => a.getAttribute('href') === '/de#contact')
    expect(contact).toBeDefined()
    expect(contact?.className).toMatch(/bg-accent-strong/)
    const ctas = links.filter((a) => /bg-accent-strong/.test(a.className))
    expect(ctas).toHaveLength(1)
    expect(container.innerHTML).not.toMatch(/ThemeToggle/)
  })

  it('offers each language as a link to the same page, with hreflang', async () => {
    await renderHeader('/de/imprint')
    const group = screen.getByRole('group', { name: /Sprache/ })
    const links = Array.from(group.querySelectorAll('a'))
    expect(links.map((a) => [a.getAttribute('hreflang'), a.getAttribute('href')])).toEqual([
      ['de', '/de/imprint'],
      ['en', '/en/imprint'],
    ])
    expect(links[0].getAttribute('aria-current')).toBe('true')
  })

  it('hands a language click to the router instead of reloading', async () => {
    const { router } = await renderHeader('/de/imprint')
    const english = screen.getByRole('link', { name: 'English' })
    const event = new MouseEvent('click', { bubbles: true, cancelable: true, button: 0 })
    english.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)
    await router.isReady()
    await vi.waitFor(() => {
      expect(router.currentRoute.value.fullPath).toBe('/en/imprint')
    })
  })

  it('shows the compact theme switch with three named modes', async () => {
    await renderHeader()
    const group = screen.getByRole('radiogroup')
    const options = Array.from(group.querySelectorAll('[role="radio"], button'))
    expect(options).toHaveLength(3)
    expect(group.querySelectorAll('svg').length).toBeGreaterThanOrEqual(3)
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  document.documentElement.style.removeProperty(HEADER_HEIGHT_PROPERTY)
})

describe('AppHeader', () => {
  it('measures the real header and publishes its height for the anchor offset', async () => {
    vi.stubGlobal('ResizeObserver', undefined)
    const spy = vi
      .spyOn(HTMLElement.prototype, 'getBoundingClientRect')
      .mockReturnValue({ height: 108 } as DOMRect)

    await renderHeader()

    expect(document.documentElement.style.getPropertyValue(HEADER_HEIGHT_PROPERTY)).toBe('108px')
    spy.mockRestore()
  })

  it('stops publishing the height when it is removed', async () => {
    const disconnect = vi.fn()
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe = vi.fn()
        disconnect = disconnect
      },
    )
    const spy = vi
      .spyOn(HTMLElement.prototype, 'getBoundingClientRect')
      .mockReturnValue({ height: 56 } as DOMRect)
    const view = await renderHeader()
    expect(document.documentElement.style.getPropertyValue(HEADER_HEIGHT_PROPERTY)).toBe('56px')

    view.unmount()

    expect(disconnect).toHaveBeenCalledOnce()
    expect(document.documentElement.style.getPropertyValue(HEADER_HEIGHT_PROPERTY)).toBe('')
    spy.mockRestore()
  })
})

describe('WordMark', () => {
  it('steps down below 480px so the brand and the actions fit one row at 360px', () => {
    const { container } = render(WordMark, { global: { plugins: [createAppI18n('de')] } })

    const img = container.querySelector('img')
    expect(img).not.toBeNull()
    const classes = img?.className.split(' ') ?? []
    expect(classes).toContain('w-24')
    expect(classes).toContain('min-[360px]:w-32')
    expect(classes).toContain('min-[400px]:w-36')
    expect(classes).toContain('min-[480px]:w-40')
    expect(classes).toContain('h-auto')
  })
})

describe('anchor offset', () => {
  const stylesheet = source('src/styles.css')

  it('comes from the measured header height through scroll-padding-top', () => {
    expect(stylesheet).toMatch(/scroll-padding-top:\s*calc\(var\(--header-height\)/)
  })

  it('has a phone fallback for the moment before the script has measured', () => {
    expect(stylesheet).toMatch(/width < 30rem[\s\S]*--header-height:\s*6\.75rem/)
  })

  it('leaves no per section scroll margin guess behind', () => {
    const offenders: string[] = []
    const walk = (dir: string): void => {
      for (const name of readdirSync(dir)) {
        const path = join(dir, name)
        if (statSync(path).isDirectory()) {
          walk(path)
        } else if (/\.(vue|ts)$/.test(name) && !name.endsWith('.test.ts')) {
          if (/scroll-mt-/.test(readFileSync(path, 'utf-8'))) {
            offenders.push(path)
          }
        }
      }
    }
    walk(join(process.cwd(), 'src'))
    expect(offenders).toEqual([])
  })
})
