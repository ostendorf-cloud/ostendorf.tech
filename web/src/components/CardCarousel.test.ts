import { readFileSync } from 'node:fs'
import { fireEvent, render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

import { createAppI18n } from '../i18n'
import type { Locale } from '../i18n'
import CardCarousel from './CardCarousel.vue'

const items = ['Rubik', 'Bademeister', 'Fourfold', 'Big Band', 'Combo', 'Herren', 'Last']

const scrollBy = vi.fn()

beforeEach(() => {
  // jsdom has no layout and no Element.scrollBy.
  HTMLElement.prototype.scrollBy = scrollBy
})

afterEach(() => {
  scrollBy.mockReset()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

async function renderCarousel(locale: Locale = 'de') {
  const view = render(CardCarousel, {
    props: {
      items,
      label: 'Bands',
      previousLabel: locale === 'de' ? 'Vorherige Bands' : 'Previous bands',
      nextLabel: locale === 'de' ? 'Nächste Bands' : 'Next bands',
    },
    slots: { slide: '<template #slide="{ item }"><p>{{ item }}</p></template>' },
    global: { plugins: [createAppI18n(locale)] },
  })
  // The buttons appear after mount.
  await screen.findAllByRole('button', { name: /bands/i })
  return view
}

/** Gives the track the metrics a browser would: a 600 px window onto 1800
 *  px of cards, scrolled to `left`. */
function layOut(track: HTMLElement, left: number): void {
  Object.defineProperty(track, 'clientWidth', { value: 600, configurable: true })
  Object.defineProperty(track, 'scrollWidth', { value: 1800, configurable: true })
  Object.defineProperty(track, 'scrollLeft', { value: left, configurable: true })
}

function trackOf(container: Element): HTMLElement {
  const track = container.querySelector<HTMLElement>('.carousel-track')
  if (!track) {
    throw new Error('no carousel track')
  }
  return track
}

describe('CardCarousel semantics', () => {
  it('is a named group with the role description carousel', async () => {
    await renderCarousel()

    const group = screen.getByRole('group', { name: 'Bands' })
    expect(group).toHaveAttribute('aria-roledescription', 'carousel')
  })

  it('renders every item once, in order, as a slide labelled "n von N"', async () => {
    const { container } = await renderCarousel()

    const slides = container.querySelectorAll('[aria-roledescription="slide"]')
    expect(slides).toHaveLength(items.length)
    slides.forEach((slide, index) => {
      expect(slide).toHaveAttribute('role', 'group')
      expect(slide).toHaveAttribute('aria-label', `${index + 1} von ${items.length}`)
      expect(slide).toHaveTextContent(items[index])
    })
  })

  it('labels the slides in English too', async () => {
    const { container } = await renderCarousel('en')

    expect(container.querySelector('[aria-roledescription="slide"]')).toHaveAttribute(
      'aria-label',
      '1 of 7',
    )
    expect(screen.getByRole('button', { name: 'Previous bands' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Next bands' })).toBeVisible()
  })

  it('names the arrows in German', async () => {
    await renderCarousel()

    expect(screen.getByRole('button', { name: 'Vorherige Bands' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Nächste Bands' })).toBeVisible()
  })

  it('makes the track focusable, as a named region, so it scrolls from the keyboard', async () => {
    const { container } = await renderCarousel()

    const track = trackOf(container)
    expect(track).toHaveAttribute('tabindex', '0')
    expect(track).toHaveAttribute('role', 'region')
    expect(screen.getByRole('region', { name: 'Bands, seitlich scrollbar' })).toBe(track)
  })

  it('names the track region in English too', async () => {
    await renderCarousel('en')

    expect(screen.getByRole('region', { name: 'Bands, scrolls sideways' })).toBeVisible()
  })

  it('stops observing its size when it is removed', async () => {
    const disconnect = vi.fn()
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe = vi.fn()
        disconnect = disconnect
      },
    )
    const view = await renderCarousel()

    view.unmount()

    expect(disconnect).toHaveBeenCalledOnce()
  })

  it('keeps a touch target of at least 44 px on the arrows', async () => {
    await renderCarousel()

    for (const name of ['Vorherige Bands', 'Nächste Bands']) {
      expect(screen.getByRole('button', { name }).className).toContain('min-h-11')
    }
  })
})

describe('CardCarousel arrows', () => {
  it('scrolls forward by one page, the visible width plus the gap', async () => {
    const { container } = await renderCarousel()
    const track = trackOf(container)
    layOut(track, 0)
    track.style.columnGap = '20px'
    await fireEvent.scroll(track)

    await userEvent.setup().click(screen.getByRole('button', { name: 'Nächste Bands' }))

    expect(scrollBy).toHaveBeenCalledTimes(1)
    expect(scrollBy).toHaveBeenCalledWith({ left: 620, behavior: 'smooth' })
  })

  it('scrolls back by one page', async () => {
    const { container } = await renderCarousel()
    const track = trackOf(container)
    layOut(track, 620)
    await fireEvent.scroll(track)

    await userEvent.setup().click(screen.getByRole('button', { name: 'Vorherige Bands' }))

    expect(scrollBy).toHaveBeenCalledWith({ left: -600, behavior: 'smooth' })
  })

  it('disables the previous arrow at the start and enables the next', async () => {
    const { container } = await renderCarousel()
    layOut(trackOf(container), 0)
    await fireEvent.scroll(trackOf(container))

    expect(screen.getByRole('button', { name: 'Vorherige Bands' })).toHaveAttribute(
      'aria-disabled',
      'true',
    )
    expect(screen.getByRole('button', { name: 'Nächste Bands' })).toHaveAttribute(
      'aria-disabled',
      'false',
    )
  })

  it('disables the next arrow at the end and enables the previous', async () => {
    const { container } = await renderCarousel()
    layOut(trackOf(container), 1200)
    await fireEvent.scroll(trackOf(container))

    expect(screen.getByRole('button', { name: 'Nächste Bands' })).toHaveAttribute(
      'aria-disabled',
      'true',
    )
    expect(screen.getByRole('button', { name: 'Vorherige Bands' })).toHaveAttribute(
      'aria-disabled',
      'false',
    )
  })

  it('does nothing when a disabled arrow is activated, and keeps it focusable', async () => {
    const { container } = await renderCarousel()
    layOut(trackOf(container), 0)
    await fireEvent.scroll(trackOf(container))
    const user = userEvent.setup()
    const previous = screen.getByRole('button', { name: 'Vorherige Bands' })

    await user.click(previous)

    expect(scrollBy).not.toHaveBeenCalled()
    expect(previous).not.toBeDisabled()
  })

  it('treats a fraction of a pixel at the end as the end', async () => {
    const { container } = await renderCarousel()
    layOut(trackOf(container), 1199.4)
    await fireEvent.scroll(trackOf(container))

    expect(screen.getByRole('button', { name: 'Nächste Bands' })).toHaveAttribute(
      'aria-disabled',
      'true',
    )
  })

  it('scrolls instantly when the visitor prefers reduced motion', async () => {
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query === '(prefers-reduced-motion: reduce)',
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    }))
    const { container } = await renderCarousel()
    layOut(trackOf(container), 0)
    await fireEvent.scroll(trackOf(container))

    await userEvent.setup().click(screen.getByRole('button', { name: 'Nächste Bands' }))

    expect(scrollBy).toHaveBeenCalledWith({ left: 600, behavior: 'auto' })
  })
})

describe('CardCarousel keyboard', () => {
  it('reaches the arrows with Tab and activates them with Enter and Space', async () => {
    const { container } = await renderCarousel()
    layOut(trackOf(container), 0)
    await fireEvent.scroll(trackOf(container))
    const user = userEvent.setup()

    await user.tab()
    expect(screen.getByRole('button', { name: 'Automatisches Scrollen pausieren' })).toHaveFocus()
    await user.tab()
    expect(screen.getByRole('button', { name: 'Vorherige Bands' })).toHaveFocus()
    await user.tab()
    const next = screen.getByRole('button', { name: 'Nächste Bands' })
    expect(next).toHaveFocus()

    await user.keyboard('{Enter}')
    expect(scrollBy).toHaveBeenCalledTimes(1)
    await user.keyboard(' ')
    expect(scrollBy).toHaveBeenCalledTimes(2)

    await user.tab()
    expect(trackOf(container)).toHaveFocus()
  })
})

describe('CardCarousel layout rules', () => {
  const css = readFileSync(`${process.cwd()}/src/components/CardCarousel.vue`, 'utf-8')

  it('shows one slide on a phone, two from 768 px and three from 1024 px', () => {
    expect(css).toMatch(/\.carousel-slide\s*\{[^}]*flex:\s*0 0 100%/)
    expect(css).toMatch(
      /@media \(min-width: 48rem\)\s*\{\s*\.carousel-slide\s*\{\s*flex-basis: calc\(\(100% - var\(--carousel-gap\)\) \/ 2\)/,
    )
    expect(css).toMatch(
      /@media \(min-width: 64rem\)\s*\{\s*\.carousel-slide\s*\{\s*flex-basis: calc\(\(100% - 2 \* var\(--carousel-gap\)\) \/ 3\)/,
    )
  })

  it('carries every band of the data file, in its order, on the home page', () => {
    const home = readFileSync(`${process.cwd()}/src/views/HomeView.vue`, 'utf-8')
    expect(home).toMatch(/<CardCarousel\s+:items="bands"/)
    expect(home).not.toMatch(/v-for="band in bands"/)
  })

  it('scrolls natively and snaps', () => {
    expect(css).toMatch(/overflow-x:\s*auto/)
    expect(css).toMatch(/scroll-snap-type:\s*x mandatory/)
    expect(css).toMatch(/scroll-snap-align:\s*start/)
  })

  it('turns a page every 6 seconds', () => {
    expect(css).toMatch(/AUTO_SCROLL_INTERVAL_MS = 6000/)
  })
})

describe('CardCarousel automatic scrolling', () => {
  const PAUSE = 'Automatisches Scrollen pausieren'
  const START = 'Automatisches Scrollen starten'

  beforeEach(() => {
    // Only the interval is faked: Testing Library's own waiting uses timeouts.
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  async function ready(left = 0) {
    const view = await renderCarousel()
    const track = trackOf(view.container)
    layOut(track, left)
    await fireEvent.scroll(track)
    const group = screen.getByRole('group', { name: 'Bands' })
    return { ...view, track, group }
  }

  function stubReducedMotion(): void {
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query === '(prefers-reduced-motion: reduce)',
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    }))
  }

  it('advances by one page after 6 seconds, not before', async () => {
    await ready()

    vi.advanceTimersByTime(5999)
    expect(scrollBy).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)

    expect(scrollBy).toHaveBeenCalledTimes(1)
    expect(scrollBy).toHaveBeenCalledWith({ left: 600, behavior: 'smooth' })
  })

  it('wraps back to the start at the end', async () => {
    await ready(1200)

    vi.advanceTimersByTime(6000)

    expect(scrollBy).toHaveBeenCalledTimes(1)
    expect(scrollBy).toHaveBeenCalledWith({ left: -1200, behavior: 'smooth' })
  })

  it('does nothing when there is nothing to scroll', async () => {
    const view = await renderCarousel()
    const track = trackOf(view.container)
    Object.defineProperty(track, 'clientWidth', { value: 600, configurable: true })
    Object.defineProperty(track, 'scrollWidth', { value: 600, configurable: true })
    Object.defineProperty(track, 'scrollLeft', { value: 0, configurable: true })
    await fireEvent.scroll(track)

    vi.advanceTimersByTime(12000)

    expect(scrollBy).not.toHaveBeenCalled()
  })

  it('pauses while the pointer is over it and resumes when it leaves', async () => {
    const { group } = await ready()

    await fireEvent(group, new Event('pointerenter'))
    vi.advanceTimersByTime(18000)
    expect(scrollBy).not.toHaveBeenCalled()

    await fireEvent(group, new Event('pointerleave'))
    vi.advanceTimersByTime(6000)
    expect(scrollBy).toHaveBeenCalledTimes(1)
  })

  it('pauses while a control inside has focus and resumes when focus leaves', async () => {
    const { group } = await ready()
    const next = screen.getByRole('button', { name: 'Nächste Bands' })

    await fireEvent.focusIn(next)
    vi.advanceTimersByTime(18000)
    expect(scrollBy).not.toHaveBeenCalled()

    await fireEvent.focusOut(next, { relatedTarget: group.querySelector('.carousel-track') })
    vi.advanceTimersByTime(18000)
    expect(scrollBy).not.toHaveBeenCalled()

    await fireEvent.focusOut(next, { relatedTarget: null })
    vi.advanceTimersByTime(6000)
    expect(scrollBy).toHaveBeenCalledTimes(1)
  })

  it('pauses while the page is hidden and resumes when it is shown', async () => {
    await ready()
    const hidden = vi.spyOn(document, 'hidden', 'get').mockReturnValue(true)

    document.dispatchEvent(new Event('visibilitychange'))
    await nextTick()
    vi.advanceTimersByTime(18000)
    expect(scrollBy).not.toHaveBeenCalled()

    hidden.mockReturnValue(false)
    document.dispatchEvent(new Event('visibilitychange'))
    await nextTick()
    vi.advanceTimersByTime(6000)
    expect(scrollBy).toHaveBeenCalledTimes(1)
  })

  it('never starts when the visitor prefers reduced motion, and shows no pause button', async () => {
    stubReducedMotion()
    await ready()

    vi.advanceTimersByTime(60000)

    expect(scrollBy).not.toHaveBeenCalled()
    expect(vi.getTimerCount()).toBe(0)
    expect(screen.queryByRole('button', { name: PAUSE })).toBeNull()
    expect(screen.queryByRole('button', { name: START })).toBeNull()
  })

  it('stops for good with the pause button and flips its label, with no pressed state', async () => {
    await ready()
    const pause = screen.getByRole('button', { name: PAUSE })
    expect(pause).toHaveAttribute('type', 'button')
    expect(pause).not.toHaveAttribute('aria-pressed')

    await fireEvent.click(pause)

    const start = screen.getByRole('button', { name: START })
    expect(start).not.toHaveAttribute('aria-pressed')
    vi.advanceTimersByTime(60000)
    expect(scrollBy).not.toHaveBeenCalled()

    await fireEvent.click(start)

    const again = screen.getByRole('button', { name: PAUSE })
    expect(again).not.toHaveAttribute('aria-pressed')
    vi.advanceTimersByTime(6000)
    expect(scrollBy).toHaveBeenCalledTimes(1)
  })

  it('keeps a pause stopped through a hover and a focus that come and go', async () => {
    const { group } = await ready()
    await fireEvent.click(screen.getByRole('button', { name: PAUSE }))

    await fireEvent(group, new Event('pointerenter'))
    await fireEvent(group, new Event('pointerleave'))
    vi.advanceTimersByTime(30000)

    expect(scrollBy).not.toHaveBeenCalled()
  })

  it('names the pause button in English and keeps it at least 44 px high and wide', async () => {
    const view = render(CardCarousel, {
      props: { items, label: 'Bands', previousLabel: 'Previous bands', nextLabel: 'Next bands' },
      slots: { slide: '<template #slide="{ item }"><p>{{ item }}</p></template>' },
      global: { plugins: [createAppI18n('en')] },
    })
    const pause = await view.findByRole('button', { name: 'Pause automatic scrolling' })

    expect(pause.className).toContain('min-h-11')
    expect(pause.className).toContain('carousel-arrow')
    await fireEvent.click(pause)
    expect(screen.getByRole('button', { name: 'Start automatic scrolling' })).toBeVisible()
  })

  it('restarts the countdown after a click on an arrow', async () => {
    await ready()

    vi.advanceTimersByTime(4000)
    await fireEvent.click(screen.getByRole('button', { name: 'Nächste Bands' }))
    expect(scrollBy).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(4000)
    expect(scrollBy).toHaveBeenCalledTimes(1)
    vi.advanceTimersByTime(2000)
    expect(scrollBy).toHaveBeenCalledTimes(2)
  })

  it('cleans up its timer and listeners when removed', async () => {
    const remove = vi.spyOn(document, 'removeEventListener')
    const view = await ready()
    expect(vi.getTimerCount()).toBe(1)

    view.unmount()

    expect(vi.getTimerCount()).toBe(0)
    expect(remove).toHaveBeenCalledWith('visibilitychange', expect.any(Function))
    vi.advanceTimersByTime(60000)
    expect(scrollBy).not.toHaveBeenCalled()
  })
})
