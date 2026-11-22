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

  // Three roles for seven bands: the last card cannot differ from the
  // first, but any three bands in a row, a full row on a wide screen, can.
  it('gives any three consecutive bands three different accents', () => {
    for (let start = 0; start + 3 <= bands.length; start++) {
      const window = bands.slice(start, start + 3).map((band) => band.accent)
      expect(new Set(window).size, `bands ${start} to ${start + 2}`).toBe(3)
    }
  })

  it('uses only the accent, info and success roles, never warning or danger', () => {
    for (const band of bands) {
      expect(['accent', 'info', 'success']).toContain(band.accent)
    }
    expect(home).not.toMatch(/\.band-accent-(warning|danger)/)
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

  it('states the real pixel size of every logo file', () => {
    for (const band of bands.filter((entry) => entry.logo)) {
      const file = `${process.cwd()}/src/assets/bands/${band.slug}-logo.webp`
      const bytes = readFileSync(file)
      expect(bytes.toString('ascii', 0, 4)).toBe('RIFF')
      expect(bytes.toString('ascii', 8, 12)).toBe('WEBP')
      // The extended header (VP8X) holds width and height minus one as 24 bit
      // little endian numbers at bytes 24 and 27. The command in AGENTS.md
      // writes that header because the logo has an alpha channel.
      expect(bytes.toString('ascii', 12, 16), 'VP8X header').toBe('VP8X')
      const width = bytes.readUIntLE(24, 3) + 1
      const height = bytes.readUIntLE(27, 3) + 1
      expect({ width: band.logo?.width, height: band.logo?.height }).toEqual({ width, height })
    }
  })

  it('names every logo in both locales', () => {
    for (const band of bands.filter((entry) => entry.logo)) {
      for (const catalogue of [de, en]) {
        const entry = (catalogue.bands as Record<string, { logoAlt?: string }>)[band.slug]
        expect(entry.logoAlt ?? '').not.toBe('')
      }
    }
  })
})
