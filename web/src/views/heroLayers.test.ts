/**
 * Proves the hero's own words are painted over the stage rather than
 * under it.
 *
 * src/theme/contrast.test.ts proves the arithmetic: what the scrim is
 * made of, and that every colour the hero paints on it clears WCAG 2.2 AA
 * over the worst pixel the photograph could put behind it. It knows
 * nothing about which element paints last, so it kept passing while the
 * stage, absolutely positioned and with no z-index of its own, painted
 * the photograph over every word in the hero. A positioned element with
 * z-index: auto paints after static in-flow content whatever the markup
 * order says; the only thing that hid it was the 800ms rise animation,
 * whose transform made a stacking context and then gave it back.
 *
 * This file is the other half: the order, not the colours.
 *
 * What it proves, and the limit of it. There is no layout engine here, so
 * this is not a picture of the rendered page; it is the painting rule
 * applied to what the source actually declares. Four facts together
 * decide the order for this structure, and all four are checked below:
 *
 *   - .hero is positioned and isolates, so its children's z-indexes are
 *     compared against each other and against nothing else on the page;
 *   - the two layer numbers are plain integers and the content one is the
 *     larger;
 *   - every direct child of .hero declares a non-static position and one
 *     of those two numbers, the stage the lower and everything else the
 *     higher;
 *   - the hero's words are inside the child that carries the higher one.
 *
 * What it cannot see is a rule from somewhere else landing on these
 * selectors, and anything that depends on real geometry. Whether the
 * beams and the grade look right is still a thing only eyes settle.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

// Resolved against the Vitest process's own working directory, for the
// same reason src/theme/contrast.test.ts gives: its module runner does
// not always give import.meta.url a real file: URL.
function source(path: string): string {
  return readFileSync(join(process.cwd(), path), 'utf-8')
}

const theme = source('src/theme/vtech.css')
const heroStage = source('src/components/HeroStage.vue')
const homeView = source('src/views/HomeView.vue')

function withoutComments(text: string): string {
  return text.replaceAll(/\/\*[\s\S]*?\*\//g, '').replaceAll(/<!--[\s\S]*?-->/g, '')
}

/** styleBlock is a single file component's scoped stylesheet. */
function styleBlock(sfc: string): string {
  const match = /<style scoped>([\s\S]*)<\/style>/.exec(sfc)
  if (!match) {
    throw new Error('heroLayers: the component carries no <style scoped> block')
  }
  return withoutComments(match[1])
}

/** declarations is the properties one top-level rule sets, by selector.
 *  Only the rules outside any at-rule are read: a rule nested in a media
 *  query describes one set of visitors rather than the layer order, which
 *  holds for all of them. */
function ruleFor(css: string, selector: string): Record<string, string> | undefined {
  let depth = 0
  let preludeStart = 0
  let bodyStart = 0
  for (let i = 0; i < css.length; i += 1) {
    if (css[i] === '{') {
      if (depth === 0) {
        bodyStart = i + 1
      }
      depth += 1
    } else if (css[i] === '}') {
      depth -= 1
      if (depth === 0) {
        if (css.slice(preludeStart, bodyStart - 1).trim() === selector) {
          return properties(css.slice(bodyStart, i))
        }
        preludeStart = i + 1
      }
    }
  }
  return undefined
}

function declarations(css: string, selector: string): Record<string, string> {
  const rule = ruleFor(css, selector)
  if (!rule) {
    throw new Error(`heroLayers: no rule for ${selector}`)
  }
  return rule
}

function properties(body: string): Record<string, string> {
  const result: Record<string, string> = {}
  for (const part of body.split(';')) {
    const colon = part.indexOf(':')
    if (colon === -1) {
      continue
    }
    result[part.slice(0, colon).trim()] = part.slice(colon + 1).trim()
  }
  return result
}

/** layer reads one of the hero's paint layers out of the theme. They are
 *  plain integers rather than var() references, so a value that stops
 *  being one is a failure here rather than a z-index the browser drops. */
function layer(name: string): number {
  const match = new RegExp(`--hero-layer-${name}:\\s*(-?\\d+)\\s*;`).exec(theme)
  if (!match) {
    throw new Error(`heroLayers: the theme declares no integer --hero-layer-${name}`)
  }
  return Number(match[1])
}

/** heroChildren is every element directly inside the hero section of
 *  HomeView's template, in order, with the markup each one encloses.
 *
 *  Reading them out of the template rather than listing them here is what
 *  makes a child added later a failure: it arrives with no layer, and the
 *  expectations below ask every child for one. */
function heroChildren(): Array<{ tag: string; classes: string[]; markup: string }> {
  const template = withoutComments(homeView)
  const heroStart = template.indexOf('<section aria-labelledby="hero-title" class="hero">')
  if (heroStart === -1) {
    throw new Error('heroLayers: HomeView has no hero section')
  }
  const children: Array<{ tag: string; classes: string[]; markup: string }> = []
  const tag = /<(\/?)([A-Za-z][\w-]*)((?:"[^"]*"|'[^']*'|[^>"'])*?)(\/?)>/g
  tag.lastIndex = heroStart
  let depth = 0
  let open: { tag: string; classes: string[]; start: number } | undefined
  let match: RegExpExecArray | null
  while ((match = tag.exec(template)) !== null) {
    const [whole, closing, name, attributes, selfClosing] = match
    if (closing === '/') {
      depth -= 1
      if (depth === 0) {
        break
      }
      if (depth === 1 && open) {
        children.push({ ...open, markup: template.slice(open.start, tag.lastIndex) })
        open = undefined
      }
      continue
    }
    const classes = /class="([^"]*)"/.exec(attributes)?.[1].split(/\s+/).filter(Boolean) ?? []
    if (selfClosing === '/') {
      if (depth === 1) {
        children.push({ tag: name, classes, markup: whole })
      }
      continue
    }
    if (depth === 1) {
      open = { tag: name, classes, start: match.index }
    }
    depth += 1
  }
  if (children.length === 0) {
    throw new Error('heroLayers: the hero section has no children')
  }
  return children
}

/** The component the stage ships as, and the class its own root carries,
 *  which is the selector its layer is declared on. */
const STAGE_COMPONENT = 'HeroStage'
const STAGE_SELECTOR = '.hero-stage'

const STATIC_POSITIONS = new Set(['static', 'initial', 'revert', 'unset'])

describe('the hero paint layers', () => {
  it('puts the content layer above the stage layer', () => {
    expect(layer('content')).toBeGreaterThan(layer('stage'))
  })

  it('compares those two against each other and nothing else on the page', () => {
    const hero = declarations(styleBlock(homeView), '.hero')

    expect(hero.position).toBe('relative')
    expect(hero.isolation).toBe('isolate')
  })

  it('names the stage as the element the stage layer is declared on', () => {
    expect(heroStage).toMatch(new RegExp(`<div\\s+class="${STAGE_SELECTOR.slice(1)}"`))
  })

  it('holds the stage on the stage layer', () => {
    const stage = declarations(styleBlock(heroStage), STAGE_SELECTOR)

    expect(stage.position).toBe('absolute')
    expect(stage['z-index']).toBe('var(--hero-layer-stage)')
  })

  it('holds every other child of the hero on the content layer', () => {
    const style = styleBlock(homeView)

    const failures = heroChildren()
      .filter((child) => child.tag !== STAGE_COMPONENT)
      .flatMap((child) => {
        if (child.classes.length === 0) {
          return [`<${child.tag}> in the hero carries no class to declare a layer on`]
        }
        return child.classes.flatMap((name) => {
          const rule = ruleFor(style, `.${name}`)
          if (!rule) {
            return [`.${name} has no rule in the hero's own stylesheet to declare a layer in`]
          }
          if (rule.position === undefined || STATIC_POSITIONS.has(rule.position)) {
            return [`.${name} is not positioned, so its z-index would be ignored`]
          }
          if (rule['z-index'] !== 'var(--hero-layer-content)') {
            return [`.${name} is on ${rule['z-index'] ?? 'no layer'}, want the content layer`]
          }
          return []
        })
      })

    expect(failures).toEqual([])
  })

  it('carries the hero wordmark, headline, tagline and buttons on that layer', () => {
    const inner = heroChildren().find((child) => child.classes.includes('hero-inner'))
    if (!inner) {
      throw new Error('heroLayers: the hero has no .hero-inner')
    }

    expect(inner.markup).toContain('<LogoMark')
    expect(inner.markup).toContain('id="hero-title"')
    expect(inner.markup).toContain('class="hero-eyebrow"')
    expect(inner.markup).toContain('class="hero-tagline"')
    expect(inner.markup).toContain('class="hero-actions"')
    expect(inner.markup).toContain("t('home.hero.ctaContact')")
    expect(inner.markup).toContain("t('home.hero.ctaServices')")
  })
})
