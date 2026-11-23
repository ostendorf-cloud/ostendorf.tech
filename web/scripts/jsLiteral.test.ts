import { describe, expect, it } from 'vitest'

import { jsLiteral } from './jsLiteral'

const LS = String.fromCharCode(0x2028)
const PS = String.fromCharCode(0x2029)

describe('jsLiteral', () => {
  it.each([
    ['<', '\\u003c'],
    ['>', '\\u003e'],
    ['&', '\\u0026'],
    [LS, '\\u2028'],
    [PS, '\\u2029'],
  ])('encodes the character %j as %s', (character, encoded) => {
    expect(jsLiteral(`a${character}b`)).toBe(`"a${encoded}b"`)
  })

  it('leaves a plain value as plain JSON', () => {
    expect(jsLiteral('ostendorf.tech:theme')).toBe('"ostendorf.tech:theme"')
    expect(jsLiteral('#f5f5f4')).toBe('"#f5f5f4"')
  })

  it('emits none of the encoded characters raw', () => {
    const out = jsLiteral(`</x><!-- & ${LS} ${PS}`)
    for (const character of ['<', '>', '&', LS, PS]) {
      expect(out).not.toContain(character)
    }
  })

  it.each([
    'plain',
    '</x>',
    '<!-- & -->',
    `line${LS}sep${PS}para`,
    'quote " and \' and backslash \\ and \\u003c',
    `tab\tnewline\n and <>&"\\ mixed ${LS}`,
    '',
  ])('parses back to the original string for %j', (value) => {
    expect(JSON.parse(jsLiteral(value))).toBe(value)
  })
})
