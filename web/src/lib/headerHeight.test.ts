import { afterEach, describe, expect, it, vi } from 'vitest'

import { HEADER_HEIGHT_PROPERTY, trackHeaderHeight } from './headerHeight'

function headerOfHeight(height: number): HTMLElement {
  const el = document.createElement('header')
  el.getBoundingClientRect = () => ({ height }) as DOMRect
  return el
}

const rootValue = () => document.documentElement.style.getPropertyValue(HEADER_HEIGHT_PROPERTY)

describe('trackHeaderHeight', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    document.documentElement.style.removeProperty(HEADER_HEIGHT_PROPERTY)
  })

  it('writes the measured height to the root element, rounded up', () => {
    vi.stubGlobal('ResizeObserver', undefined)

    trackHeaderHeight(headerOfHeight(107.2))

    expect(rootValue()).toBe('108px')
  })

  it('updates the property whenever the header is resized', () => {
    let notify: () => void = () => {}
    const observe = vi.fn()
    let height = 56
    const el = document.createElement('header')
    el.getBoundingClientRect = () => ({ height }) as DOMRect
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(callback: () => void) {
          notify = callback
        }
        observe = observe
        disconnect = vi.fn()
      },
    )

    trackHeaderHeight(el)
    expect(observe).toHaveBeenCalledWith(el)
    expect(rootValue()).toBe('56px')

    height = 140
    notify()
    expect(rootValue()).toBe('140px')
  })

  it('stops observing and clears the property when told to', () => {
    const disconnect = vi.fn()
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe = vi.fn()
        disconnect = disconnect
      },
    )

    const stop = trackHeaderHeight(headerOfHeight(56))
    stop()

    expect(disconnect).toHaveBeenCalledOnce()
    expect(rootValue()).toBe('')
  })

  it('leaves the stylesheet fallback alone when the header has no height yet', () => {
    vi.stubGlobal('ResizeObserver', undefined)

    trackHeaderHeight(headerOfHeight(0))

    expect(rootValue()).toBe('')
  })
})
