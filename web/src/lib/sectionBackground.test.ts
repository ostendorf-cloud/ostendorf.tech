import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  PROBE_RATIO,
  SECTION_BG_ATTRIBUTE,
  sectionToneAt,
  trackSectionBackground,
} from './sectionBackground'

describe('sectionToneAt', () => {
  const alt = [
    { top: 100, bottom: 300 },
    { top: 600, bottom: 800 },
  ]

  it('is base with no alt section', () => {
    expect(sectionToneAt([], 150)).toBe('base')
  })

  it('is alt while the line is inside an alt section', () => {
    expect(sectionToneAt(alt, 150)).toBe('alt')
    expect(sectionToneAt(alt, 700)).toBe('alt')
  })

  it('is base between sections and outside all of them', () => {
    expect(sectionToneAt(alt, 50)).toBe('base')
    expect(sectionToneAt(alt, 450)).toBe('base')
    expect(sectionToneAt(alt, 900)).toBe('base')
  })

  it('includes the top edge and excludes the bottom edge', () => {
    expect(sectionToneAt(alt, 100)).toBe('alt')
    expect(sectionToneAt(alt, 300)).toBe('base')
  })
})

describe('trackSectionBackground', () => {
  let top = 0
  let stop = (): void => {}
  let frames: FrameRequestCallback[] = []

  // Runs the animation frames the page asked for, as the browser would.
  const fire = (type: string): void => {
    window.dispatchEvent(new Event(type))
    const due = frames
    frames = []
    due.forEach((callback) => {
      callback(0)
    })
  }

  beforeEach(() => {
    top = 1000
    vi.stubGlobal('ResizeObserver', undefined)
    frames = []
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      frames.push(callback)
      return frames.length
    })
    vi.stubGlobal('cancelAnimationFrame', () => {
      frames = []
    })
    window.innerHeight = 800
    document.body.innerHTML = '<section data-section-tone="alt" id="alt"></section>'
    vi.spyOn(
      document.getElementById('alt') as HTMLElement,
      'getBoundingClientRect',
    ).mockImplementation(() => ({ top, bottom: top + 400 }) as DOMRect)
  })

  afterEach(() => {
    stop()
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  const value = (): string | null => document.documentElement.getAttribute(SECTION_BG_ATTRIBUTE)

  it('starts on base while the alt section is below the line', () => {
    stop = trackSectionBackground()
    expect(value()).toBe('base')
  })

  it('switches to alt once a scroll puts the section under the line, and back', () => {
    stop = trackSectionBackground()
    top = 100 // the line is at PROBE_RATIO of 800 px
    expect(800 * PROBE_RATIO).toBeGreaterThanOrEqual(100)
    fire('scroll')
    expect(value()).toBe('alt')
    top = -900
    fire('scroll')
    expect(value()).toBe('base')
  })

  it('reacts to a resize', () => {
    stop = trackSectionBackground()
    top = 100
    fire('resize')
    expect(value()).toBe('alt')
  })

  it('leaves the root as it found it when stopped', () => {
    stop = trackSectionBackground()
    stop()
    expect(value()).toBeNull()
    top = 100
    fire('scroll')
    expect(value()).toBeNull()
  })
})
