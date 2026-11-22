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
    expect(styles).toMatch(/\.btn-gradient\s*\{[^}]*var\(--brand-gradient\) border-box/)
    expect(styles).toMatch(/\.btn-gradient\s*\{[^}]*var\(--color-surface\)/)
  })

  it('does not lift or animate for reduced motion', () => {
    expect(styles).toMatch(
      /prefers-reduced-motion: reduce\)\s*\{\s*\.btn-gradient\s*\{\s*transition: none;[\s\S]*?transform: none;/,
    )
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
