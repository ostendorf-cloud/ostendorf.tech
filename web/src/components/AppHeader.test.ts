import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { render, screen, within } from '@testing-library/vue'
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
    await renderHeader()
    const nav = screen.getByRole('navigation', { name: 'Hauptmenü' })
    expect(nav.closest('[class*="justify-end"]')).not.toBeNull()
  })

  it('lies over the hero on the home page only, transparent before a scroll', async () => {
    const home = await renderHeader('/de')
    const header = home.container.querySelector('header')
    expect(header?.dataset.overlay).toBe('true')
    expect(header?.dataset.state).toBe('transparent')
    home.unmount()

    const imprint = await renderHeader('/de/imprint')
    const solid = imprint.container.querySelector('header')
    expect(solid?.dataset.overlay).toBeUndefined()
    expect(solid?.dataset.state).toBe('solid')
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

  it('hands a language click to the router instead of reloading', async () => {
    const { router } = await renderHeader('/de/imprint')
    const group = screen.getByRole('group', { name: /Sprache/ })
    const english = within(group).getByRole('link', { name: 'English' })
    const event = new MouseEvent('click', { bubbles: true, cancelable: true, button: 0 })
    english.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)
    await router.isReady()
    await vi.waitFor(() => {
      expect(router.currentRoute.value.fullPath).toBe('/en/imprint')
    })
  })

  it('puts the theme button and the language link last, after the navigation', async () => {
    const { container } = await renderHeader()
    const actions = container.querySelector('header .ml-auto') as HTMLElement
    const nav = screen.getByRole('navigation', { name: 'Hauptmenü' })
    expect(actions.contains(nav)).toBe(false)
    const children = Array.from(actions.children)
    expect(children).toHaveLength(2)
    expect(children[0].matches('button[data-mode]')).toBe(true)
    expect(children[1].matches('[role="group"]')).toBe(true)
    expect(nav.compareDocumentPosition(actions) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })
})

describe.each([
  {
    locale: 'de',
    path: '/de/imprint',
    other: ['en', 'English', '/en/imprint'],
    current: 'Deutsch',
    cycle: /^Darstellung: (Dem System folgen|Heller Modus|Dunkler Modus)\. Als Nächstes: /,
  },
  {
    locale: 'en',
    path: '/en/imprint',
    other: ['de', 'Deutsch', '/de/imprint'],
    current: 'English',
    cycle: /^Theme: (Follow the system|Light mode|Dark mode)\. Next: /,
  },
])('AppHeader controls at every width, $locale', ({ path, other, current, cycle }) => {
  it('shows only the link to the other language, with hreflang and lang', async () => {
    const { container } = await renderHeader(path)
    const links = Array.from(container.querySelectorAll('[role="group"] a'))
    expect(
      links.map((a) => [a.getAttribute('hreflang'), a.textContent?.trim(), a.getAttribute('href')]),
    ).toEqual([other])
    expect(links[0].getAttribute('lang')).toBe(other[0])
    expect(links[0].getAttribute('aria-current')).toBeNull()
    expect(screen.queryByText(current)).toBeNull()
  })

  it('shows the one cycle theme button and none of the compact radios', async () => {
    const { container } = await renderHeader(path)
    expect(container.querySelectorAll('[role="radiogroup"]')).toHaveLength(0)
    expect(container.querySelectorAll('[role="radio"]')).toHaveLength(0)
    const buttons = Array.from(container.querySelectorAll('button[data-mode]'))
    expect(buttons).toHaveLength(1)
    expect(buttons[0].getAttribute('type')).toBe('button')
    expect(buttons[0].getAttribute('aria-label')).toMatch(cycle)
  })

  it('has no width variant wrapper around either control', async () => {
    const { container } = await renderHeader(path)
    expect(container.querySelectorAll('[role="group"]')).toHaveLength(1)
    expect(container.innerHTML).not.toMatch(/lg:hidden|lg:contents/)
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

  // The prerender fallback has to equal what trackHeaderHeight measures, or
  // the hero's top padding jumps when the script runs. jsdom has no layout,
  // so the expected heights are derived from govue's header classes, read
  // from the installed package, and compared with the stylesheet's values.
  const govue = (name: string) =>
    readFileSync(join(process.cwd(), `node_modules/@ostendorf-cloud/govue/src/${name}`), 'utf-8')
  const REM = 16
  const BORDER = 1 // border-b
  const ROW_PADDING = 8 // py-2, above and below, below 480px only
  const ROW_GAP = 4 // gap-y-1
  const TARGET = 44 // min-h-11 and size-11

  /** px of a fallback value: a rem or px length, or a calc() sum of them. */
  function pixels(value: string, spacingHeader: number): number {
    const inner = value.trim().replace(/^calc\((.*)\)$/, '$1')
    return inner.split('+').reduce((sum, term) => {
      const part = term.trim()
      if (part === 'var(--spacing-header)') return sum + spacingHeader
      const length = /^([\d.]+)(rem|px)$/.exec(part)
      if (!length) throw new Error(`unexpected term ${part}`)
      return sum + Number(length[1]) * (length[2] === 'rem' ? REM : 1)
    }, 0)
  }

  function fallback(css: string, phone: boolean): string {
    const block = phone
      ? /@media \(width < 480px\)\s*\{\s*:root\s*\{([^}]*)\}/.exec(css)?.[1]
      : /^:root\s*\{([^}]*)\}/m.exec(css)?.[1]
    const value = /--header-height:\s*([^;]+);/.exec(block ?? '')?.[1]
    expect(value, phone ? 'below 480px' : 'from 480px').toBeDefined()
    return value ?? ''
  }

  const spacingHeader = (): number => {
    const value = /--spacing-header:\s*([\d.]+)rem;/.exec(govue('styles/base.css'))?.[1]
    expect(value).toBeDefined()
    return Number(value) * REM
  }

  it('reads govue header classes the fallback is derived from', () => {
    const header = govue('components/GovueHeader.vue')
    for (const cls of ['border-b', 'py-2', 'gap-y-1', 'min-[480px]:h-header', 'min-[480px]:py-0']) {
      expect(header, cls).toContain(cls)
    }
    expect(header).toMatch(/order-1 flex min-h-11/)
    expect(govue('components/GovueNav.vue')).toMatch(/inline-flex min-h-11[^"]*sm:hidden/)
    expect(govue('components/GovueThemeSwitch.vue')).toMatch(/inline-flex size-11/)
    expect(govue('components/GovueLanguageSwitch.vue')).toMatch(/inline-flex min-h-11/)
    expect(spacingHeader()).toBe(56)
  })

  it('falls back to one row of 57px from 480px', () => {
    const expected = spacingHeader() + BORDER
    expect(expected).toBe(57)
    expect(pixels(fallback(stylesheet, false), spacingHeader())).toBe(expected)
  })

  it('falls back to two rows of 109px below 480px', () => {
    const expected = ROW_PADDING + TARGET + ROW_GAP + TARGET + ROW_PADDING + BORDER
    expect(expected).toBe(109)
    expect(pixels(fallback(stylesheet, true), spacingHeader())).toBe(expected)
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
