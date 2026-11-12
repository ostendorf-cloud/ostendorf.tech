import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { useTheme } from '@ostendorf-cloud/govue'
import { beforeEach, describe, expect, it } from 'vitest'

import { createAppI18n } from '../i18n'
import ThemeToggle from './ThemeToggle.vue'

function renderToggle() {
  return render(ThemeToggle, {
    global: { plugins: [createAppI18n('de')] },
  })
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
