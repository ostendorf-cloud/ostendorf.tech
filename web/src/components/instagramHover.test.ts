/**
 * Proves the Instagram tile's hover from the source: the numbers a browser
 * would apply. How it looks while it moves needs eyes.
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
    throw new Error('instagramHover: the component carries no <style scoped> block')
  }
  return match[1].replaceAll(/\/\*[\s\S]*?\*\//g, '')
}

function rule(css: string, selector: string): string {
  const escaped = selector.replaceAll(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)
  const match = new RegExp(String.raw`(?:^|\})\s*${escaped}\s*\{([^}]*)\}`).exec(css)
  if (!match) {
    throw new Error(`instagramHover: no rule for ${selector}`)
  }
  return match[1]
}

const css = styleBlock(source('src/components/InstagramFeed.vue'))
const theme = source('src/theme/vtech.css')

describe('the tile hover', () => {
  it('rounds the tile at 14 px with a 1 px border', () => {
    const tile = rule(css, '.insta-tile')
    expect(tile).toMatch(/border:\s*1px solid var\(--color-border\)/)
    expect(tile).toMatch(/border-radius:\s*14px/)
  })

  it('scales the picture to 1.06 over 600 ms with the theme easing', () => {
    expect(rule(css, '.insta-tile img')).toMatch(/transition:\s*transform 600ms var\(--ease-out\)/)
    expect(css).toMatch(
      /\.insta-tile:hover img,\s*\.insta-tile:focus-visible img\s*\{[^}]*transform:\s*scale\(1\.06\)/,
    )
  })

  it('fades the overlay in over 250 ms, on hover and on keyboard focus', () => {
    expect(rule(css, '.insta-overlay')).toMatch(/transition:\s*opacity 250ms/)
    expect(rule(css, '.insta-overlay')).toMatch(/opacity:\s*0;/)
    expect(css).toMatch(
      /\.insta-tile:hover \.insta-overlay,\s*\.insta-tile:focus-visible \.insta-overlay\s*\{[^}]*opacity:\s*1/,
    )
  })

  it('sizes the glyph at 34 px', () => {
    const glyph = rule(css, '.insta-overlay svg')
    expect(glyph).toMatch(/width:\s*34px/)
    expect(glyph).toMatch(/height:\s*34px/)
  })

  it('paints the overlay at 135 degrees from the two theme variables', () => {
    expect(rule(css, '.insta-overlay')).toMatch(
      /linear-gradient\(135deg,\s*var\(--ig-overlay-from\),\s*var\(--ig-overlay-to\)\)/,
    )
  })

  it('defines the two overlay colours once, from the palette tokens', () => {
    expect(theme.match(/--ig-overlay-from:/g)).toHaveLength(1)
    expect(theme.match(/--ig-overlay-to:/g)).toHaveLength(1)
    expect(theme).toMatch(
      /--ig-overlay-from:\s*color-mix\(in srgb, var\(--color-accent\) 35%, transparent\)/,
    )
    expect(theme).toMatch(
      /--ig-overlay-to:\s*color-mix\(in srgb, var\(--color-accent-strong\) 55%, transparent\)/,
    )
  })

  it('takes away the scale and the fade for reduced motion', () => {
    expect(css).toMatch(
      /@media \(prefers-reduced-motion: reduce\)\s*\{\s*\.insta-tile img,\s*\.insta-overlay\s*\{\s*transition: none;\s*\}\s*\.insta-tile:hover img,\s*\.insta-tile:focus-visible img\s*\{\s*transform: none;/,
    )
  })
})
