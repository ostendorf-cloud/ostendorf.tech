import { fireEvent, render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { useTheme } from '@ostendorf-cloud/govue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { createAppI18n } from '../i18n'
import { DEFAULT_MODE } from '../lib/theme'
import ThemeToggle from './ThemeToggle.vue'

function renderToggle() {
  return render(ThemeToggle, {
    global: { plugins: [createAppI18n('de')] },
  })
}

/** jsdom has neither the View Transitions API nor the Web Animations API,
 *  so a test that exercises the reveal has to supply both by hand, and
 *  remove them again afterwards to leave the feature-detection in
 *  ThemeToggle.vue seeing the same absence every other test sees. */
function stubViewTransitions(): { animate: ReturnType<typeof vi.fn> } {
  const animate = vi.fn()
  document.documentElement.animate = animate

  const fakeTransition = {
    ready: Promise.resolve(),
    finished: Promise.resolve(),
    updateCallbackDone: Promise.resolve(),
    skipTransition: vi.fn(),
    types: new Set<string>(),
  }
  document.startViewTransition = (callback?: () => void | Promise<void>) => {
    void callback?.()
    return fakeTransition
  }

  return { animate }
}

function removeViewTransitionStubs(): void {
  delete (document as unknown as { startViewTransition?: unknown }).startViewTransition
  delete (document.documentElement as unknown as { animate?: unknown }).animate
}

describe('ThemeToggle', () => {
  // useTheme's state is a module level singleton, by design: one page has
  // one theme. A test that leaves mode changed would leak into the next
  // one, so every test starts by forcing it back to "system" through the
  // same public setMode the component itself calls.
  beforeEach(() => {
    useTheme().setMode('system')
  })

  it('names light, the first step of the cycle, as the action at the start', () => {
    renderToggle()

    expect(screen.getByRole('button', { name: 'Zu hellem Modus wechseln' })).toBeVisible()
  })

  // The site does not start on "system": src/lib/theme.ts starts it dark,
  // and main.ts and the prerender script both hand that to useTheme. So
  // the state a visitor who has never touched the toggle actually finds
  // it in is dark, with the system setting as the next step, and the moon
  // rather than the sun on the button.
  it('shows the mode this site starts in, with the system setting as the next step', () => {
    useTheme().setMode(DEFAULT_MODE)
    renderToggle()

    expect(screen.getByRole('button', { name: 'Systemeinstellung übernehmen' })).toBeVisible()
  })

  it('cycles through light, dark and back to system on repeated clicks', async () => {
    const user = userEvent.setup()
    renderToggle()

    const button = screen.getByRole('button')
    // Starting from system, a click switches to light; the label then
    // names the next step, dark.
    await user.click(button)
    expect(screen.getByRole('button', { name: 'Zu dunklem Modus wechseln' })).toBeVisible()

    await user.click(button)
    expect(screen.getByRole('button', { name: 'Systemeinstellung übernehmen' })).toBeVisible()

    await user.click(button)
    expect(screen.getByRole('button', { name: 'Zu hellem Modus wechseln' })).toBeVisible()
  })
})

describe('ThemeToggle with View Transitions support', () => {
  beforeEach(() => {
    useTheme().setMode('system')
  })

  afterEach(() => {
    removeViewTransitionStubs()
    vi.unstubAllGlobals()
  })

  it('still switches the mode, through the same cycleMode, when the browser supports the reveal', async () => {
    stubViewTransitions()
    const user = userEvent.setup()
    renderToggle()

    await user.click(screen.getByRole('button'))

    expect(screen.getByRole('button', { name: 'Zu dunklem Modus wechseln' })).toBeVisible()
  })

  it('clips the new view to a circle growing from the click point', async () => {
    const { animate } = stubViewTransitions()
    const user = userEvent.setup()
    renderToggle()

    await user.click(screen.getByRole('button'))

    expect(animate).toHaveBeenCalledTimes(1)
    const [keyframes, options] = animate.mock.calls[0] as [
      { clipPath: string[] },
      { duration: number; pseudoElement: string },
    ]
    expect(keyframes.clipPath).toHaveLength(2)
    expect(keyframes.clipPath[0]).toMatch(/^circle\(0px at/)
    expect(keyframes.clipPath[1]).toMatch(/^circle\(/)
    expect(options.duration).toBe(300)
    expect(options.pseudoElement).toBe('::view-transition-new(root)')
  })

  it('centres the reveal on the button when activated from the keyboard', async () => {
    const { animate } = stubViewTransitions()
    renderToggle()
    const button = screen.getByRole('button')
    button.getBoundingClientRect = () => ({
      left: 100,
      top: 50,
      width: 40,
      height: 20,
      right: 140,
      bottom: 70,
      x: 100,
      y: 50,
      toJSON: () => ({}),
    })

    // A keyboard click carries no pointer coordinates: MouseEvent.detail
    // is 0 for one synthesized by Enter or Space, which is what tells
    // ThemeToggle.vue to fall back to the button's own centre, here
    // (120, 60).
    await fireEvent.click(button, { detail: 0 })

    const [keyframes] = animate.mock.calls[0] as [{ clipPath: string[] }]
    expect(keyframes.clipPath[0]).toBe('circle(0px at 120px 60px)')
  })

  it('switches instantly, with no reveal, when the visitor prefers reduced motion', async () => {
    const { animate } = stubViewTransitions()
    const startViewTransition = vi.spyOn(document, 'startViewTransition')
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query === '(prefers-reduced-motion: reduce)',
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    }))
    const user = userEvent.setup()
    renderToggle()

    await user.click(screen.getByRole('button'))

    expect(screen.getByRole('button', { name: 'Zu dunklem Modus wechseln' })).toBeVisible()
    expect(startViewTransition).not.toHaveBeenCalled()
    expect(animate).not.toHaveBeenCalled()
  })
})
