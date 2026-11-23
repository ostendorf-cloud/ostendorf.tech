/**
 * Proves when the fader mark carries its animation class: only when asked
 * for, only once the script has run, and frozen while the visitor has
 * paused the hero. What the class then does is read from the stylesheet in
 * src/views/heroMotion.test.ts.
 */
import { render } from '@testing-library/vue'
import { createSSRApp, h, nextTick } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { describe, expect, it } from 'vitest'

import LogoMark from './LogoMark.vue'

describe('LogoMark animation state', () => {
  it('is at rest in the prerendered markup, even when animated', async () => {
    const html = await renderToString(
      createSSRApp({ render: () => h(LogoMark, { animated: true }) }),
    )
    expect(html).not.toContain('logo-mark-animated')
  })

  it('animates once mounted when asked to', async () => {
    const { container } = render(LogoMark, { props: { animated: true } })
    await nextTick()
    expect(container.querySelector('svg')).toHaveClass('logo-mark-animated')
    expect(container.querySelector('svg')).not.toHaveClass('is-paused')
  })

  it('stays calm when not asked to animate', async () => {
    const { container } = render(LogoMark)
    await nextTick()
    expect(container.querySelector('svg')).not.toHaveClass('logo-mark-animated')
  })

  it('is marked paused while the visitor has paused the hero', async () => {
    const { container, rerender } = render(LogoMark, { props: { animated: true, paused: true } })
    await nextTick()
    expect(container.querySelector('svg')).toHaveClass('is-paused')
    await rerender({ animated: true, paused: false })
    expect(container.querySelector('svg')).not.toHaveClass('is-paused')
  })
})
