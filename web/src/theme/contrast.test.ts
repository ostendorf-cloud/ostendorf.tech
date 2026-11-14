/**
 * Proves every pair vtech.css fills has to carry passes WCAG 2.2 AA, in
 * both modes.
 *
 * govue ships the exact same check as web/src/styles/contrast.ts, but its
 * package.json "exports" map only publishes ".", "./styles.css" and
 * "./themes/*.css": contrast.ts itself is not on the public surface a
 * consumer can import. This file is therefore the same algorithm and the
 * same pairs, read directly out of govue's source at the pinned version
 * this package depends on, so a change to either side is caught here
 * rather than two implementations quietly drifting apart.
 *
 * The second half of the file is this app's own, and has no counterpart
 * upstream: the home page's hero paints text over a photograph, which is
 * a pair no token list can describe. See "the hero stage" there for what
 * it proves and why the photograph itself never enters the arithmetic.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const TEXT_MINIMUM = 4.5
const NON_TEXT_MINIMUM = 3

const TEXT_PAIRS: Array<[fg: string, bg: string]> = [
  ['text', 'bg'],
  ['text', 'bg-subtle'],
  ['text', 'surface'],
  ['text', 'surface-hover'],
  ['text-muted', 'bg'],
  ['text-muted', 'surface'],
  ['text-faint', 'bg'],
  ['text-faint', 'surface'],
  ['accent-text', 'bg'],
  ['accent-text', 'surface'],
  ['accent-text', 'accent-subtle'],
  ['accent-on', 'accent-strong'],
  ['accent-on', 'accent-strong-hover'],
  ['danger-on', 'danger-strong'],
  ['danger-on', 'danger-strong-hover'],
  ['success-text', 'surface'],
  ['success-text', 'success-subtle'],
  ['warning-text', 'surface'],
  ['warning-text', 'warning-subtle'],
  ['danger-text', 'surface'],
  ['danger-text', 'danger-subtle'],
  ['info-text', 'surface'],
  ['info-text', 'info-subtle'],
]

const NON_TEXT_PAIRS: Array<[fg: string, bg: string]> = [
  ['border-strong', 'bg'],
  ['border-strong', 'surface'],
  ['accent', 'bg'],
  ['accent', 'surface'],
  ['focus', 'bg'],
  ['focus', 'surface'],
  ['focus', 'surface-hover'],
]

type Palette = Record<string, string>

function tokens(css: string): { light: Palette; dark: Palette } {
  const light: Palette = {}
  const dark: Palette = {}
  const pattern =
    /--color-([\w-]+):\s*light-dark\(\s*(#[0-9a-fA-F]{6})\s*,\s*(#[0-9a-fA-F]{6})\s*\)/g
  for (const match of css.matchAll(pattern)) {
    const [, name, lightValue, darkValue] = match
    light[name] = lightValue
    dark[name] = darkValue
  }
  return { light, dark }
}

function channel(value: number): number {
  const srgb = value / 255
  return srgb <= 0.03928 ? srgb / 12.92 : ((srgb + 0.055) / 1.055) ** 2.4
}

function luminance(hex: string): number {
  const value = Number.parseInt(hex.slice(1), 16)
  const r = channel((value >> 16) & 0xff)
  const g = channel((value >> 8) & 0xff)
  const b = channel(value & 0xff)
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

function ratio(foreground: string, background: string): number {
  const a = luminance(foreground)
  const b = luminance(background)
  const [lighter, darker] = a > b ? [a, b] : [b, a]
  return (lighter + 0.05) / (darker + 0.05)
}

function round(value: number): number {
  return Math.round(value * 100) / 100
}

function failures(
  palette: Palette,
  pairs: Array<[fg: string, bg: string]>,
  minimum: number,
): string[] {
  return pairs.flatMap(([fg, bg]) => {
    const missing = [fg, bg].filter((token) => palette[token] === undefined)
    if (missing.length > 0) {
      return [`${fg} on ${bg} cannot be checked, ${missing.join(' and ')} is not defined`]
    }
    const value = ratio(palette[fg], palette[bg])
    return value < minimum ? [`${fg} on ${bg} is ${round(value)}:1, want ${minimum}:1`] : []
  })
}

// Resolved against the Vitest process's own working directory, which
// "task web:test" and "pnpm run test" both run from this package's root,
// rather than import.meta.url: Vitest's module runner does not always
// give that a real file: URL.
const themePath = join(process.cwd(), 'src/theme/vtech.css')
const css = readFileSync(themePath, 'utf-8')
const { light, dark } = tokens(css)

describe('the vtech theme', () => {
  it('declares every text pair at 4.5:1 in light mode', () => {
    expect(failures(light, TEXT_PAIRS, TEXT_MINIMUM)).toEqual([])
  })

  it('declares every non-text pair at 3:1 in light mode', () => {
    expect(failures(light, NON_TEXT_PAIRS, NON_TEXT_MINIMUM)).toEqual([])
  })

  it('declares every text pair at 4.5:1 in dark mode', () => {
    expect(failures(dark, TEXT_PAIRS, TEXT_MINIMUM)).toEqual([])
  })

  it('declares every non-text pair at 3:1 in dark mode', () => {
    expect(failures(dark, NON_TEXT_PAIRS, NON_TEXT_MINIMUM)).toEqual([])
  })
})

/*
 * The hero stage.
 *
 * HeroStage.vue layers a photograph, the haze and the light beams under
 * one scrim painted in --hero-scrim-color, and the hero's text sits on
 * top of that scrim with nothing decorative between. So the darkest the
 * hero's ground can ever be read as follows from two declared numbers and
 * a worst case pixel rather than from what the photograph happens to
 * contain: a pure white pixel is the worst a light text can sit on.
 *
 * Swapping the photograph for another one therefore cannot regress any
 * of this, and neither can a photograph that is lighter than the one
 * shipped today, which is the whole point of putting the scrim last.
 *
 * There is one scrim to prove rather than two, because the hero is dark
 * whichever mode the rest of the page is in. That is not an assumption
 * this file is free to make: HomeView.vue has to pin the scheme on the
 * hero for the dark palette below to be the one the hero actually paints
 * with, so the first test here reads that declaration back out of it. Were
 * it to go, this file would be proving the contrast of a hero that no
 * longer exists.
 */
const WORST_CASE_BEHIND_SCRIM = 0xff

/** Every role the hero paints on the scrim, with the ratio WCAG 2.2 AA
 *  asks of it. The headline is clamped to at least 2.3rem at 850 weight,
 *  which is large text, so its gradient's two ends carry the 3:1 large
 *  text minimum rather than 4.5:1; the eyebrow, the tagline and the
 *  buttons' own labels are body text and carry 4.5:1. */
const HERO_ROLES: Array<[token: string, minimum: number]> = [
  ['text', TEXT_MINIMUM],
  ['brand-gradient-from', NON_TEXT_MINIMUM],
  ['brand-gradient-to', NON_TEXT_MINIMUM],
  ['accent', NON_TEXT_MINIMUM],
  ['focus', NON_TEXT_MINIMUM],
]

/** heroTokens resolves the hero's own custom properties, which unlike the
 *  colour tokens above are written as a plain number or as a var()
 *  reference to a colour token, so that the gradient provably cannot
 *  carry a value the palette does not already have. */
function heroTokens(source: string): { alpha: number; refs: Record<string, string> } {
  const alphaMatch = /--hero-scrim-alpha:\s*([0-9.]+)/.exec(source)
  if (!alphaMatch) {
    throw new Error('contrast: vtech.css declares no --hero-scrim-alpha')
  }
  const refs: Record<string, string> = {}
  const pattern = /--((?:hero|brand)-[\w-]+):\s*var\(\s*--color-([\w-]+)\s*\)/g
  for (const [, name, target] of source.matchAll(pattern)) {
    refs[name] = target
  }
  return { alpha: Number(alphaMatch[1]), refs }
}

/** heroGround is the colour the hero's text actually sits on in the worst
 *  case: the scrim at its weakest declared stop, composited over a pixel
 *  of `behind` in every channel. */
function heroGround(scrim: string, alpha: number, behind: number): string {
  const value = Number.parseInt(scrim.slice(1), 16)
  const channels = [(value >> 16) & 0xff, (value >> 8) & 0xff, value & 0xff].map((c) =>
    Math.round(alpha * c + (1 - alpha) * behind),
  )
  return `#${channels.map((c) => c.toString(16).padStart(2, '0')).join('')}`
}

/** heroFailures reads every hero role out of the palette, following a
 *  var() reference where the hero declares one, and reports the ones that
 *  do not reach their minimum against the ground. */
function heroFailures(palette: Palette, refs: Record<string, string>, ground: string): string[] {
  return HERO_ROLES.flatMap(([token, minimum]) => {
    const colour = palette[refs[token] ?? token]
    if (colour === undefined) {
      return [`${token} cannot be checked, it resolves to no colour token`]
    }
    const value = ratio(colour, ground)
    return value < minimum
      ? [`${token} on the hero scrim is ${round(value)}:1, want ${minimum}:1`]
      : []
  })
}

const hero = heroTokens(css)

/** The hero's own stylesheet, where the scheme it paints in is pinned.
 *  Resolved the same way the theme above is, and for the same reason. */
const homeView = readFileSync(join(process.cwd(), 'src/views/HomeView.vue'), 'utf-8')

describe('the hero scrim', () => {
  it('is painted in the mode this file measures, whatever the page is in', () => {
    expect(homeView).toMatch(/\.hero\s*\{[^}]*\bcolor-scheme:\s*dark\s*;/)
  })

  it('is built from the theme, not from a value of its own', () => {
    expect(hero.refs['hero-scrim-color']).toBe('bg')
    expect(hero.refs['brand-gradient-from']).toBe('accent')
    expect(hero.refs['brand-gradient-to']).toBe('accent-text')
  })

  it('carries every hero role over the brightest possible pixel', () => {
    const ground = heroGround(dark.bg, hero.alpha, WORST_CASE_BEHIND_SCRIM)
    expect(heroFailures(dark, hero.refs, ground)).toEqual([])
  })

  // The scrim is the only thing standing between the photograph and the
  // hero's text, so every hundredth of alpha above the floor is paid for
  // out of the picture. Holding the alpha *at* the floor rather than
  // merely above it is what stops a floor that has since dropped, because
  // a token moved or because a mode stopped being proved here, from
  // quietly keeping the photograph dimmer than it has to be.
  it('is no thicker than the roles it carries need', () => {
    const thinner = heroGround(dark.bg, hero.alpha - 0.01, WORST_CASE_BEHIND_SCRIM)
    expect(heroFailures(dark, hero.refs, thinner)).not.toEqual([])
  })
})
