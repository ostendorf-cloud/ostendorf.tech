import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { catalogues, type MessageTree } from '@ostendorf-cloud/govue'
import { describe, expect, it } from 'vitest'

import { GOVUE_CONTACT_DE, NEUTRAL_CONTACT_KEYS } from './govue'

type Tree = MessageTree | string

/** leaves lists every string leaf of a tree as [dotted key, text]. */
function leaves(tree: Tree, prefix = ''): Array<[string, string]> {
  if (typeof tree === 'string') {
    return [[prefix, tree]]
  }
  return Object.entries(tree).flatMap(([key, child]) =>
    leaves(child, prefix === '' ? key : `${prefix}.${key}`),
  )
}

const govueContactDe = new Map(
  leaves((catalogues.de.messages as unknown as { govue: { contact: MessageTree } }).govue.contact),
)
const overrides = new Map(leaves(GOVUE_CONTACT_DE))

/** placeholders are the {name} parts of a text, sorted. */
function placeholders(text: string): string[] {
  return [...text.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort()
}

describe("the site's German texts for govue's contact form", () => {
  it("overrides only keys govue's German catalogue has, with the same placeholders", () => {
    for (const [key, text] of overrides) {
      expect(govueContactDe.has(key), `govue.contact.${key} in govue's catalogue`).toBe(true)
      expect(placeholders(text), key).toEqual(placeholders(govueContactDe.get(key) ?? ''))
    }
  })

  it('sorts every govue contact key into overridden or neutral, never both', () => {
    const overridden = [...overrides.keys()]
    expect(overridden.filter((key) => NEUTRAL_CONTACT_KEYS.includes(key))).toEqual([])
    expect([...overridden, ...NEUTRAL_CONTACT_KEYS].sort()).toEqual(
      [...govueContactDe.keys()].sort(),
    )
  })

  it('leaves no German contact text that says du, dein, dir or dich', () => {
    const merged = new Map([...govueContactDe, ...overrides])
    for (const [key, text] of merged) {
      expect(text, key).not.toMatch(/\b(du|dein\w*|dir|dich)\b/i)
    }
  })

  it('is installed by the browser entry and the prerender script', () => {
    for (const path of ['src/main.ts', 'scripts/prerender.ts']) {
      const source = readFileSync(join(process.cwd(), path), 'utf-8')
      expect(source, path).toMatch(/app\.use\(createSiteGovueI18n\(/)
      expect(source, path).not.toMatch(/createGovueI18n\(/)
    }
  })
})
