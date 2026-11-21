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

  it('shows a monitor and names the system setting as current for "system"', () => {
    renderToggle()

    const button = screen.getByRole('button', {
      name: 'Farbmodus: Systemeinstellung. Wechseln zu: Hell.',
    })
    expect(button).toBeVisible()
    expect(button).toHaveAttribute('data-mode', 'system')
    expect(button.querySelector('rect')).not.toBeNull()
  })

  // The site does not start on "system": src/lib/theme.ts starts it dark.
  it('shows a moon for dark, the mode this site starts in', () => {
    useTheme().setMode(DEFAULT_MODE)
    renderToggle()

    const button = screen.getByRole('button', {
      name: 'Farbmodus: Dunkel. Wechseln zu: Systemeinstellung.',
    })
    expect(button).toHaveAttribute('data-mode', 'dark')
    expect(button.querySelector('rect')).toBeNull()
    expect(button.querySelector('circle')).toBeNull()
  })

  it('shows a sun for light', () => {
    useTheme().setMode('light')
    renderToggle()

    const button = screen.getByRole('button', {
      name: 'Farbmodus: Hell. Wechseln zu: Dunkel.',
    })
    expect(button).toHaveAttribute('data-mode', 'light')
    expect(button.querySelector('circle')).not.toBeNull()
  })

  it('does not show light or dark for the system choice, whatever the system resolves to', () => {
    useTheme().setMode('system')
    renderToggle()

    expect(screen.getByRole('button').querySelector('circle')).toBeNull()
    expect(screen.getByRole('button').querySelector('rect')).not.toBeNull()
  })

  it('cycles through light, dark and back to system on repeated clicks', async () => {
    const user = userEvent.setup()
    renderToggle()

    const button = screen.getByRole('button')
    await user.click(button)
    expect(button).toHaveAccessibleName('Farbmodus: Hell. Wechseln zu: Dunkel.')

    await user.click(button)
    expect(button).toHaveAccessibleName('Farbmodus: Dunkel. Wechseln zu: Systemeinstellung.')

    await user.click(button)
    expect(button).toHaveAccessibleName('Farbmodus: Systemeinstellung. Wechseln zu: Hell.')
  })

  it('is operable from the keyboard with Enter and Space', async () => {
    const user = userEvent.setup()
    renderToggle()

    await user.tab()
    const button = screen.getByRole('button')
    expect(button).toHaveFocus()

    await user.keyboard('{Enter}')
    expect(button).toHaveAttribute('data-mode', 'light')
    await user.keyboard(' ')
    expect(button).toHaveAttribute('data-mode', 'dark')
  })

  it('persists the choice the way useTheme does and applies it to the page', async () => {
    const user = userEvent.setup()
    localStorage.removeItem('govue-theme')
    renderToggle()

    await user.click(screen.getByRole('button'))

    const stored = JSON.parse(localStorage.getItem('govue-theme') ?? '{}') as { mode?: string }
    expect(stored.mode).toBe('light')
    expect(document.documentElement.dataset.mode).toBe('light')
  })

  it('keeps a touch target of at least 44 px', () => {
    renderToggle()

    expect(screen.getByRole('button').className).toContain('min-h-11')
    expect(screen.getByRole('button').className).toContain('min-w-11')
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

    expect(
      screen.getByRole('button', { name: 'Farbmodus: Hell. Wechseln zu: Dunkel.' }),
    ).toBeVisible()
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

    expect(
      screen.getByRole('button', { name: 'Farbmodus: Hell. Wechseln zu: Dunkel.' }),
    ).toBeVisible()
    expect(startViewTransition).not.toHaveBeenCalled()
    expect(animate).not.toHaveBeenCalled()
  })
})
