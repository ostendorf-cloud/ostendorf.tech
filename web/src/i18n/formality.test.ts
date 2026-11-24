/**
 * The site says "Sie". govue's language layer loads its formal German
 * catalogue with formality 'formal', and the browser and the prerender must
 * both install it, so the first paint and the running app say the same.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { createGovueI18n } from '@ostendorf-cloud/govue'
import { describe, expect, it } from 'vitest'

import { createAppI18n } from './index'

function source(path: string): string {
  return readFileSync(join(process.cwd(), path), 'utf-8')
}

describe('formal German', () => {
  for (const path of ['src/main.ts', 'scripts/prerender.ts']) {
    it(`is installed by ${path}`, () => {
      const text = source(path)
      expect(text).toMatch(/createGovueI18n\(\{[^}]*formality: 'formal'[^}]*\}\)/)
      expect(text).not.toMatch(/createSiteGovueI18n/)
    })
  }

  it('addresses the visitor with "Sie" in govue texts', () => {
    const i18n = createAppI18n('de')
    createGovueI18n({ i18n, locale: 'de', formality: 'formal' })
    const t = i18n.global.t
    expect(t('govue.contact.error.required')).toBe('Bitte füllen Sie dieses Feld aus.')
    expect(t('govue.contact.privacyIntro')).toBe('So gehen wir mit Ihren Angaben um:')
  })
})
