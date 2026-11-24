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

  it('glows from the theme tokens, stronger on hover and weaker when pressed', () => {
    expect(styles).toMatch(/^\.btn-gradient\s*\{[^}]*box-shadow:[^}]*var\(--glow-purple\)/m)
    expect(styles).toMatch(/\.btn-gradient:active\s*\{[^}]*box-shadow:/)
    expect(source('src/theme/vtech.css')).toMatch(/--glow-purple:\s*light-dark\(/)
    expect(source('src/theme/vtech.css')).toMatch(/--glow-cyan:\s*light-dark\(/)
  })

  it('has a visible focus ring and a ghost sibling that stills for reduced motion', () => {
    expect(styles).toMatch(
      /:is\(\.btn-gradient, \.btn-ghost\):focus-visible\s*\{\s*outline: 2px solid var\(--color-focus\)/,
    )
    expect(styles.match(/^\.btn-ghost\s*\{/gm)).toHaveLength(1)
    expect(styles).toMatch(
      /prefers-reduced-motion: reduce\)[\s\S]*\.btn-ghost\s*\{\s*transition: none/,
    )
  })

  it('restyles no govue link through its href: govue offers no hook for it', () => {
    expect(styles).not.toMatch(/a\[href/)
  })

  it('makes the header translucent through govue header tokens, with no selector of its own', () => {
    const theme = source('src/theme/vtech.css')
    expect(theme).toMatch(/--color-header-bg:\s*color-mix\(in srgb, var\(--color-bg\) 86%/)
    expect(theme).toMatch(/--header-backdrop:\s*blur\(14px\) saturate\(1\.2\);/)
    expect(styles).not.toMatch(/\.app-header\b/)
    expect(styles).not.toMatch(/backdrop-filter/)
  })

  it('is what the primary buttons of the home page use', () => {
    const home = source('src/views/HomeView.vue')
    expect(home.match(/class="btn-gradient /g)).toHaveLength(2)
    expect(home.match(/class="btn-ghost /g)).toHaveLength(1)
    expect(home).not.toMatch(/bg-accent-strong/)
  })
})

describe('the section ground', () => {
  const styles = source('src/styles.css')

  it('turns the body to the raised ground, smoothly, and not at all for reduced motion', () => {
    expect(styles).toMatch(
      /:root\[data-section-bg='alt'\] body\s*\{\s*background-color: var\(--color-bg-subtle\)/,
    )
    expect(styles).toMatch(/^body\s*\{\s*transition: background-color/m)
    expect(styles).toMatch(/prefers-reduced-motion: reduce\)\s*\{\s*body\s*\{\s*transition: none/)
  })

  it('is asked for by Über mich and Impressionen only', () => {
    expect(source('src/views/HomeView.vue').match(/data-section-tone="alt"/g)).toHaveLength(1)
    expect(
      source('src/components/InstagramFeed.vue').match(/data-section-tone="alt"/g),
    ).toHaveLength(1)
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
