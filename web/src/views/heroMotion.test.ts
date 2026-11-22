/**
 * Proves the hero's motion and tap target rules from the source.
 *
 * There is no layout engine or animation clock here, so this reads what
 * the stylesheets declare: the numbers a browser would then apply. What it
 * cannot show is how the hero looks while it moves; that needs eyes.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

function source(path: string): string {
  return readFileSync(join(process.cwd(), path), 'utf-8')
}

function styleBlock(sfc: string): string {
  const match = /<style scoped>([\s\S]*)<\/style>/.exec(sfc)
  if (!match) {
    throw new Error('heroMotion: the component carries no <style scoped> block')
  }
  return match[1].replaceAll(/\/\*[\s\S]*?\*\//g, '')
}

/** rule returns the declaration block of the first rule whose selector
 *  list is exactly `selector`, outside any at-rule. */
function rule(css: string, selector: string): string {
  const escaped = selector.replaceAll(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)
  const match = new RegExp(String.raw`(?:^|\})\s*${escaped}\s*\{([^}]*)\}`).exec(css)
  if (!match) {
    throw new Error(`heroMotion: no rule for ${selector}`)
  }
  return match[1]
}

function declaration(block: string, property: string): string {
  const match = new RegExp(String.raw`(?:^|[;\s])${property}:\s*([^;]+);`).exec(block)
  if (!match) {
    throw new Error(`heroMotion: no ${property} declaration`)
  }
  return match[1].trim()
}

function pixels(value: string): number {
  const rem = /^(-?[\d.]+)rem$/.exec(value)
  if (rem) {
    return Number(rem[1]) * 16
  }
  const px = /^(-?[\d.]+)px$/.exec(value)
  if (px) {
    return Number(px[1])
  }
  throw new Error(`heroMotion: cannot read ${value} as a length`)
}

const home = styleBlock(source('src/views/HomeView.vue'))
const logo = styleBlock(source('src/components/LogoMark.vue'))

describe('hero scroll cue', () => {
  const cue = rule(home, '.hero-cue')

  it('is at least 44 by 44 px', () => {
    expect(pixels(declaration(cue, 'width'))).toBeGreaterThanOrEqual(44)
    expect(pixels(declaration(cue, 'height'))).toBeGreaterThanOrEqual(44)
  })

  it('keeps the mouse outline at its drawn size inside the larger target', () => {
    const outline = rule(home, '.hero-cue::after')
    expect(pixels(declaration(outline, 'width'))).toBe(26)
    expect(pixels(declaration(outline, 'height'))).toBe(42)
  })

  it('rolls its dot a limited number of times, within five seconds', () => {
    const dot = declaration(rule(home, '.hero-cue::before'), 'animation')
    const match = /hero-roll\s+([\d.]+)s\s+\S+\s+(\d+)$/.exec(dot)
    expect(match, dot).not.toBeNull()
    expect(Number(match?.[1]) * Number(match?.[2])).toBeLessThanOrEqual(5)
  })
})

describe('logo animation', () => {
  it('never loops', () => {
    expect(logo).not.toMatch(/infinite/)
    expect(home).not.toMatch(/infinite/)
  })

  it('ends within five seconds of starting, so it needs no pause control', () => {
    const animated = declaration(rule(logo, '.logo-mark-animated .logo-knob'), 'animation')
    const base = /logo-fader\s+([\d.]+)s\s+\S+\s+(\d+)$/.exec(animated)
    expect(base, animated).not.toBeNull()
    expect(Number(base?.[2])).toBe(1)
    expect(Number(base?.[1]) * Number(base?.[2])).toBeLessThanOrEqual(5)
  })

  // The shared rule sets the animation shorthand with higher specificity
  // than a .knob-N rule, so a duration or delay on a knob rule would never
  // apply and would only mislead. The animation that runs is the shared one.
  it('keeps no dead per knob timing that the shared rule would override', () => {
    for (const knob of ['.knob-1', '.knob-2', '.knob-3', '.knob-4']) {
      expect(rule(logo, knob)).not.toMatch(/animation/)
    }
  })

  it('stays still for reduced motion', () => {
    expect(logo).toMatch(
      /@media \(prefers-reduced-motion: reduce\)\s*\{\s*\.logo-mark-animated \.logo-knob\s*\{\s*animation: none;/,
    )
  })
})

function pixelsOfSeconds(value: string): number {
  const match = /^(-?[\d.]+)s$/.exec(value)
  if (!match) {
    throw new Error(`heroMotion: cannot read ${value} as seconds`)
  }
  return Number(match[1])
}

describe('hero light beams', () => {
  const stage = styleBlock(source('src/components/HeroStage.vue'))

  /** The duration each beam really runs with: the shared one, unless the
   *  beam's own rule overrides it. */
  function durations(): number[] {
    const shared = declaration(rule(stage, '.hero-beam'), 'animation')
    const base = /hero-sway\s+([\d.]+)s\s+ease-in-out\s+infinite\s+alternate\b/.exec(shared)
    expect(base, shared).not.toBeNull()
    return [1, 2, 3, 4].map((n) => {
      const own = /animation-duration:\s*([^;]+);/.exec(rule(stage, `.hero-beam:nth-child(${n})`))
      return own ? pixelsOfSeconds(own[1].trim()) : Number(base?.[1])
    })
  }

  it('sway slowly and endlessly, as on the old site: 12, 15, 13 and 17 seconds a leg', () => {
    expect(durations()).toEqual([12, 15, 13, 17])
  })

  it('are slower than the single 3.5 to 5 second sweep they replace', () => {
    expect(Math.min(...durations())).toBeGreaterThan(5)
  })

  // Endless motion is only allowed because the visitor can stop it.
  it('only run endlessly while a pause control exists and can stop them', () => {
    expect(stage).toMatch(
      /\.hero-stage\.is-paused \.hero-beam\s*\{\s*animation-play-state:\s*paused;/,
    )
    const home = source('src/views/HomeView.vue')
    expect(home).toContain('<HeroMotionToggle')
    expect(home).toMatch(/<HeroStage :paused="motionPaused"/)
  })

  it('stays still for reduced motion', () => {
    expect(stage).toMatch(
      /@media \(prefers-reduced-motion: reduce\)\s*\{\s*\.hero-beam\s*\{\s*animation: none;/,
    )
  })
})

describe('hero entrance', () => {
  it('does not fade the headline in from opacity 0', () => {
    const inner = rule(home, '.hero-inner')
    expect(inner).not.toMatch(/animation/)
    expect(home).not.toMatch(/\.hero-title[^{]*\{[^}]*animation/)
    expect(home).not.toMatch(/\.hero-eyebrow[^{]*\{[^}]*animation/)
  })

  it('fades only the decoration, and not at all for reduced motion', () => {
    const rise = /\.hero-logo,\s*\.hero-tagline,\s*\.hero-actions\s*\{([^}]*)\}/.exec(home)
    expect(rise?.[1]).toMatch(/animation: hero-rise/)

    const reduced = /@media \(prefers-reduced-motion: reduce\)\s*\{([\s\S]*?animation: none;)/.exec(
      home,
    )
    expect(reduced?.[1]).toContain('.hero-logo')
    expect(reduced?.[1]).toContain('.hero-tagline')
    expect(reduced?.[1]).toContain('.hero-actions')
    expect(reduced?.[1]).toContain('.hero-cue::before')
  })
})
