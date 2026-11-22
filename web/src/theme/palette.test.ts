/**
 * Proves the old site's gradient reaches the page through tokens only: one
 * class in styles.css, and no colour literal in any component. What it
 * cannot show is how it looks; that needs eyes.
 */
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

function source(path: string): string {
  return readFileSync(join(process.cwd(), path), 'utf-8')
}

function sfcs(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) {
      return sfcs(path)
    }
    return name.endsWith('.vue') ? [path] : []
  })
}

describe('the gradient button', () => {
  const styles = source('src/styles.css')

  it('is defined once, from the brand gradient and the surface tokens', () => {
    expect(styles.match(/^\.btn-gradient\s*\{/gm)).toHaveLength(1)
    expect(styles).toMatch(/^\.btn-gradient\s*\{[^}]*var\(--brand-gradient\) border-box/m)
    expect(styles).toMatch(/^\.btn-gradient\s*\{[^}]*var\(--color-surface\)/m)
  })

  it('does not lift or animate for reduced motion', () => {
    expect(styles).toMatch(
      /prefers-reduced-motion: reduce\)\s*\{\s*\.btn-gradient\s*\{\s*transition: none;[\s\S]*?transform: none;/,
    )
  })

  // The purple end of the border is 3.2:1 on the surface and 2.95:1 on the
  // hover surface (see contrast.test.ts), so hover must not repaint the fill.
  it('keeps the surface fill on hover, only lifting', () => {
    expect(styles).not.toMatch(/\.btn-gradient[^{]*hover[^{]*\{[^}]*surface-hover/)
    expect(styles).toMatch(/\.btn-gradient:is\(:hover, :focus-visible\)\s*\{\s*transform:/)
  })

  it('restyles no govue link through its href: govue offers no hook for it', () => {
    expect(styles).not.toMatch(/a\[href/)
  })

  it('makes the header translucent from one token and one selector, with the govue gap named', () => {
    expect(styles).toMatch(
      /@supports \(backdrop-filter: blur\(1px\)\)[^{]*\{\s*\.app-header\s*\{[^}]*background-color: var\(--header-bg\)[^}]*backdrop-filter: blur\(14px\)/,
    )
    expect(styles.match(/\.app-header\b/g)).toHaveLength(1)
    expect(styles).toMatch(/govue has no\s+\*?\s*header background token/)
  })

  it('is what the primary buttons of the home page use', () => {
    const home = source('src/views/HomeView.vue')
    expect(home.match(/class="btn-gradient /g)).toHaveLength(2)
    expect(home).not.toMatch(/bg-accent-strong/)
  })
})

describe('components', () => {
  it('carry no hex colour of their own, except the logo mark', () => {
    const offenders = sfcs(join(process.cwd(), 'src'))
      .filter((path) => !path.endsWith('LogoMark.vue'))
      .filter((path) =>
        /#[0-9a-fA-F]{3,8}\b(?![\w-])/.test(
          readFileSync(path, 'utf-8')
            .replaceAll(/&#\d+;/g, '')
            .replaceAll(/href="#[\w-]+"/g, '')
            .replaceAll(/url\(#[\w-]+\)/g, ''),
        ),
      )
    expect(offenders).toEqual([])
  })
})
