import { describe, expect, it } from 'vitest'

import { serviceIcons } from './services'
import de from '../i18n/locales/de.json'
import en from '../i18n/locales/en.json'

describe('the Leistungen icons', () => {
  it('has one icon for every entry the German catalogue lists', () => {
    expect(serviceIcons).toHaveLength(de.home.services.items.length)
  })

  it('has one icon for every entry the English catalogue lists', () => {
    expect(serviceIcons).toHaveLength(en.home.services.items.length)
  })

  it('names each icon only once, so no two entries share one', () => {
    expect(new Set(serviceIcons).size).toBe(serviceIcons.length)
  })
})
