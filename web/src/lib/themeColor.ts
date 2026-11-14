/**
 * The colour a browser paints its own chrome with around the page, read
 * out of the theme rather than written down a second time.
 *
 * The prerendered <head> is the only place this is needed: the SPA never
 * changes it after load. It used to carry the dark value as a literal,
 * which meant --color-bg had a copy nothing held to it. Reading the token
 * here removes the copy instead of guarding it.
 *
 * Both ends are returned rather than the one the pages currently ship in,
 * because which end that is belongs to src/lib/theme.ts: this file's job
 * is only to say what the theme declares, and failing when the token has
 * been renamed or reshaped is the whole of the rest of it.
 */

/** The light and dark ends of one light-dark() colour token. */
export interface ThemeColors {
  light: string
  dark: string
}

const BACKGROUND = /--color-bg:\s*light-dark\(\s*(#[0-9a-fA-F]{6})\s*,\s*(#[0-9a-fA-F]{6})\s*\)/

/**
 * themeColors pulls --color-bg's two ends out of a theme stylesheet.
 *
 * It throws rather than falling back on a default, because a default
 * would be exactly the hard copy this exists to remove: a theme that has
 * renamed or reshaped the token should fail the build, not ship chrome
 * in a colour the page no longer uses.
 */
export function themeColors(css: string): ThemeColors {
  const match = BACKGROUND.exec(css)
  if (!match) {
    throw new Error('themeColors: the theme declares no --color-bg as light-dark(<hex>, <hex>)')
  }
  return { light: match[1], dark: match[2] }
}
