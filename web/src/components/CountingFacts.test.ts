import { render, screen } from '@testing-library/vue'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { Fact } from '../lib/facts'
import CountingFacts from './CountingFacts.vue'

/** jsdom has no IntersectionObserver at all, so the component's own
 *  feature check never finds one unless a test supplies this fake, which
 *  records every instance and lets a test fire its callback by hand. */
class FakeIntersectionObserver implements IntersectionObserver {
  static instances: FakeIntersectionObserver[] = []
  readonly root = null
  readonly rootMargin = ''
  readonly thresholds: ReadonlyArray<number> = []
  disconnected = false
  private readonly callback: IntersectionObserverCallback

  constructor(callback: IntersectionObserverCallback) {
    this.callback = callback
    FakeIntersectionObserver.instances.push(this)
  }

  observe(): void {}
  unobserve(): void {}
  takeRecords(): IntersectionObserverEntry[] {
    return []
  }

  disconnect(): void {
    this.disconnected = true
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

function latestObserver(): FakeIntersectionObserver {
  const observer = FakeIntersectionObserver.instances.at(-1)
  if (!observer) {
    throw new Error('no IntersectionObserver was created')
  }
  return observer
}

describe('CountingFacts', () => {
  beforeEach(() => {
    FakeIntersectionObserver.instances = []
    vi.stubGlobal('IntersectionObserver', FakeIntersectionObserver)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('shows the final values up front, before anything has scrolled into view', () => {
    render(CountingFacts, { props: { facts } })

    expect(screen.getByText('10+')).toBeVisible()
    expect(screen.getByText('4')).toBeVisible()
    expect(screen.getByText('1')).toBeVisible()
  })

  it('counts up from zero, as one motion, once the list first scrolls into view', async () => {
    vi.useFakeTimers()
    render(CountingFacts, { props: { facts } })

    latestObserver().trigger(true)
    await nextTick()
    // The animation resets every value to zero in the same tick it starts,
    // before the first animation frame has even run.
    expect(screen.getByText('0+')).toBeVisible()
    expect(screen.getAllByText('0')).toHaveLength(2)

    await vi.advanceTimersByTimeAsync(700)

    expect(screen.getByText('10+')).toBeVisible()
    expect(screen.getByText('4')).toBeVisible()
    expect(screen.getByText('1')).toBeVisible()
  })

  it('never counts a second time once it has already counted', async () => {
    vi.useFakeTimers()
    render(CountingFacts, { props: { facts } })

    const observer = latestObserver()
    observer.trigger(true)
    await vi.advanceTimersByTimeAsync(700)
    expect(screen.getByText('10+')).toBeVisible()

    // A second intersection entry, for example scrolling the section out
    // and back in, must not restart the count.
    observer.trigger(true)
    await vi.advanceTimersByTimeAsync(700)

    expect(screen.getByText('10+')).toBeVisible()
    expect(screen.queryByText('0+')).not.toBeInTheDocument()
  })
})
