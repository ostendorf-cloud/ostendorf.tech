import { render, screen } from '@testing-library/vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { Fact } from '../lib/facts'
// Vitest hoists vi.mock above every import in this file, including this
// one: a separate file, because vi.mock applies to every test in the
// module it is called in, and only this one case needs
// prefers-reduced-motion forced to true.
import CountingFacts from './CountingFacts.vue'

vi.mock('../lib/motion', () => ({ prefersReducedMotion: () => true }))

class FakeIntersectionObserver implements IntersectionObserver {
  static instances: FakeIntersectionObserver[] = []
  readonly root = null
  readonly rootMargin = ''
  readonly thresholds: ReadonlyArray<number> = []
  private readonly callback: IntersectionObserverCallback

  constructor(callback: IntersectionObserverCallback) {
    this.callback = callback
    FakeIntersectionObserver.instances.push(this)
  }

  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
  takeRecords(): IntersectionObserverEntry[] {
    return []
  }

  trigger(isIntersecting: boolean): void {
    const entry = { isIntersecting } as IntersectionObserverEntry
    this.callback([entry], this)
  }
}

const facts: Fact[] = [
  { label: 'Jahre Erfahrung', target: 10, suffix: '+' },
  { label: 'Gewerke', target: 4 },
  { label: 'Ansprechpartner', target: 1 },
]

describe('CountingFacts with prefers-reduced-motion: reduce', () => {
  beforeEach(() => {
    FakeIntersectionObserver.instances = []
    vi.stubGlobal('IntersectionObserver', FakeIntersectionObserver)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('shows the final values immediately, with no counting, once in view', () => {
    render(CountingFacts, { props: { facts } })

    const observer = FakeIntersectionObserver.instances.at(-1)
    observer?.trigger(true)

    expect(screen.getByText('10+')).toBeVisible()
    expect(screen.getByText('4')).toBeVisible()
    expect(screen.getByText('1')).toBeVisible()
    expect(screen.queryByText('0+')).not.toBeInTheDocument()
  })
})
