import { describe, expect, it } from 'vitest'

import de from './locales/de.json'
import en from './locales/en.json'

/** leafPaths lists the dotted path of every string leaf. An array counts
 *  as one leaf per index, so a list that is longer in one language shows. */
function leafPaths(value: unknown, prefix = ''): string[] {
  if (typeof value === 'string') {
    return [prefix]
  }
  if (Array.isArray(value)) {
    return value.flatMap((item, index) => leafPaths(item, `${prefix}[${index}]`))
  }
  if (value !== null && typeof value === 'object') {
    return Object.entries(value).flatMap(([key, child]) =>
      leafPaths(child, prefix === '' ? key : `${prefix}.${key}`),
    )
  }
  return [prefix]
}

describe('locale files', () => {
  // The legal subtree is left out on purpose: the English legal pages carry
  // two keys the German ones do not (the convenience translation notice and
  // the link to the German original), the German ones one key the English
  // ones do not (which version is binding), and that text is owned
  // separately.
  const withoutLegal = (paths: string[]) => paths.filter((path) => !path.startsWith('legal.'))

  it('carry the same keys in German and English', () => {
    const german = new Set(withoutLegal(leafPaths(de)))
    const english = new Set(withoutLegal(leafPaths(en)))

    expect([...german].filter((key) => !english.has(key))).toEqual([])
    expect([...english].filter((key) => !german.has(key))).toEqual([])
  })

  it('carry no empty string', () => {
    for (const locale of [de, en]) {
      const empty = leafPaths(locale).filter((path) => {
        const value = path
          .split(/[.[\]]+/)
          .filter(Boolean)
          .reduce<unknown>((node, key) => (node as Record<string, unknown>)[key], locale)
        return value === ''
      })
      expect(empty).toEqual([])
    }
  })
})
