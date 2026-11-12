import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import { createAppI18n } from '../i18n'
import ContactChannels from './ContactChannels.vue'

function renderChannels() {
  return render(ContactChannels, { global: { plugins: [createAppI18n('de')] } })
}

describe('ContactChannels', () => {
  it('links every channel to its real destination', () => {
    renderChannels()

    expect(screen.getByRole('link', { name: /E-Mail/ })).toHaveAttribute(
      'href',
      'mailto:simon@ostendorf.tech',
    )
    expect(screen.getByRole('link', { name: /Telefon/ })).toHaveAttribute(
      'href',
      'tel:+4915730971784',
    )
    expect(screen.getByRole('link', { name: /WhatsApp/ })).toHaveAttribute(
      'href',
      'https://wa.me/4915730971784',
    )
    expect(screen.getByRole('link', { name: /Instagram/ })).toHaveAttribute(
      'href',
      'https://instagram.com/simonostendorf.vtech',
    )
    expect(screen.getByRole('link', { name: /Facebook/ })).toHaveAttribute(
      'href',
      'https://facebook.com/simonostendorf.vtech/',
    )
  })

  it('opens only the third party channels in a new tab', () => {
    renderChannels()

    expect(screen.getByRole('link', { name: /E-Mail/ })).not.toHaveAttribute('target')
    expect(screen.getByRole('link', { name: /Telefon/ })).not.toHaveAttribute('target')
    expect(screen.getByRole('link', { name: /WhatsApp/ })).toHaveAttribute('target', '_blank')
    expect(screen.getByRole('link', { name: /Instagram/ })).toHaveAttribute('target', '_blank')
    expect(screen.getByRole('link', { name: /Facebook/ })).toHaveAttribute('target', '_blank')
  })

  it('names the list for a screen reader', () => {
    renderChannels()

    expect(screen.getByRole('list', { name: 'Kontaktwege' })).toBeVisible()
  })
})
