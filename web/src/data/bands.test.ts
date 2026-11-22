import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

import de from '../i18n/locales/de.json'
import en from '../i18n/locales/en.json'
import { bands } from './bands'

const home = readFileSync(`${process.cwd()}/src/views/HomeView.vue`, 'utf-8')

describe('bands data', () => {
  it('has a description in both locales for every band that claims one', () => {
    for (const band of bands) {
      for (const catalogue of [de, en]) {
        const entry = (catalogue.bands as Record<string, { name: string; paragraphs?: string[] }>)[
          band.slug
        ]
        expect(entry.name).not.toBe('')
        if (band.hasDescription) {
          expect(entry.paragraphs?.length ?? 0).toBeGreaterThan(0)
        }
      }
    }
  })

  it('gives neighbouring bands different accents', () => {
    bands.slice(1).forEach((band, index) => {
      expect(band.accent).not.toBe(bands[index].accent)
    })
  })

  it('maps every accent in use to a class that reads theme tokens only', () => {
    for (const accent of new Set(bands.map((band) => band.accent))) {
      const rule = new RegExp(`\\.band-accent-${accent}\\s*\\{([^}]*)\\}`).exec(home)
      expect(rule, `class for ${accent}`).not.toBeNull()
      const body = rule?.[1] ?? ''
      expect(body).toContain(`--band-text: var(--color-${accent}-text)`)
      expect(body).toContain(`--band-subtle: var(--color-${accent}-subtle)`)
      expect(body).not.toMatch(/#[0-9a-f]{3,8}\b/i)
    }
  })

  it('ships no logo until the owner adds one', () => {
    expect(bands.every((band) => band.logo === undefined)).toBe(true)
  })
})
