import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { render } from '@testing-library/vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import { createAppI18n } from '../i18n'
import { HEADER_HEIGHT_PROPERTY } from '../lib/headerHeight'
import AppHeader from './AppHeader.vue'
import WordMark from './WordMark.vue'

function source(path: string): string {
  return readFileSync(join(process.cwd(), path), 'utf-8')
}

async function renderHeader() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:locale', name: 'home', component: { template: '<div />' } }],
  })
  await router.push('/de')
  await router.isReady()
  return render(AppHeader, { global: { plugins: [createAppI18n('de'), router] } })
}

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
    expect(stylesheet).toMatch(/max-width:\s*29\.99rem[\s\S]*--header-height:\s*6\.75rem/)
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
