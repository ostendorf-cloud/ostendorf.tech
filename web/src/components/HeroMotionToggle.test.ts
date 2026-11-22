import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { defineComponent, h, ref } from 'vue'

import { createAppI18n } from '../i18n'
import type { Locale } from '../i18n'
import HeroMotionToggle from './HeroMotionToggle.vue'
import HeroStage from './HeroStage.vue'

const source = readFileSync(join(process.cwd(), 'src/components/HeroMotionToggle.vue'), 'utf-8')
const css = /<style scoped>([\s\S]*)<\/style>/.exec(source)?.[1] ?? ''

/** The control and the stage wired together the way HomeView wires them. */
function renderHero(locale: Locale) {
  const hero = defineComponent({
    setup() {
      const paused = ref(false)
      return () => [
        h(HeroStage, { paused: paused.value }),
        h(HeroMotionToggle, {
          paused: paused.value,
          onToggle: () => (paused.value = !paused.value),
        }),
      ]
    },
  })
  return render(hero, { global: { plugins: [createAppI18n(locale)] } })
}

describe('HeroMotionToggle', () => {
  it('is a real button, named for what the next press does, in German', async () => {
    renderHero('de')
    const button = screen.getByRole('button', { name: 'Animation pausieren' })
    expect(button.tagName).toBe('BUTTON')
    expect(button).toHaveAttribute('type', 'button')

    await userEvent.click(button)

    expect(screen.getByRole('button', { name: 'Animation starten' })).toBeInTheDocument()
  })

  it('flips between Pause animation and Start animation in English', async () => {
    renderHero('en')

    await userEvent.click(screen.getByRole('button', { name: 'Pause animation' }))
    expect(screen.getByRole('button', { name: 'Start animation' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Start animation' }))
    expect(screen.getByRole('button', { name: 'Pause animation' })).toBeInTheDocument()
  })

  it('carries no aria-pressed, the label does the work', async () => {
    renderHero('en')
    const button = screen.getByRole('button', { name: 'Pause animation' })
    expect(button).not.toHaveAttribute('aria-pressed')
    await userEvent.click(button)
    expect(screen.getByRole('button')).not.toHaveAttribute('aria-pressed')
  })

  it('freezes the beams when pressed and releases them when pressed again', async () => {
    const { container } = renderHero('en')
    const stage = container.querySelector('.hero-stage')
    expect(stage).not.toHaveClass('is-paused')

    await userEvent.click(screen.getByRole('button'))
    expect(stage).toHaveClass('is-paused')

    await userEvent.click(screen.getByRole('button'))
    expect(stage).not.toHaveClass('is-paused')
  })

  it('is reachable by keyboard', async () => {
    renderHero('en')
    await userEvent.tab()
    expect(screen.getByRole('button')).toHaveFocus()
    await userEvent.keyboard('{Enter}')
    expect(screen.getByRole('button', { name: 'Start animation' })).toBeInTheDocument()
  })

  it('is at least 44 by 44 px', () => {
    expect(css).toMatch(/min-width:\s*2\.75rem/)
    expect(css).toMatch(/min-height:\s*2\.75rem/)
  })

  it('remembers nothing', () => {
    expect(source).not.toMatch(/localStorage|sessionStorage|document\.cookie/)
  })

  it('is not shown, and so not focusable, when the visitor asks for reduced motion', () => {
    expect(css).toMatch(
      /@media \(prefers-reduced-motion: reduce\)\s*\{\s*\.hero-motion-toggle\s*\{\s*display:\s*none;/,
    )
  })
})
