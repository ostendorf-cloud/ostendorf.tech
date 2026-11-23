/** Code points written as unicode escapes by jsLiteral: < > & and the
 *  two Unicode line and paragraph separators. */
const ENCODED = new Set([0x3c, 0x3e, 0x26, 0x2028, 0x2029])

/** jsLiteral writes value as a JSON string literal that is safe to place
 *  in markup: the characters of ENCODED become unicode escapes, which a
 *  JavaScript string literal reads back as the same characters.
 *
 *  JSON.parse(jsLiteral(x)) === x holds for every string x. */
export function jsLiteral(value: string): string {
  let out = ''
  for (const character of JSON.stringify(value)) {
    const code = character.codePointAt(0) ?? 0
    out += ENCODED.has(code) ? `\\u${code.toString(16).padStart(4, '0')}` : character
  }
  return out
}
