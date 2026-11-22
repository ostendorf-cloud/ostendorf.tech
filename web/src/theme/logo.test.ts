/**
 * The logo's two colours are fixed: the favicon's cyan and violet, the same
 * in light and in dark mode, and read from their own tokens rather than from
 * the palette's accent roles.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

function source(path: string): string {
  return readFileSync(join(process.cwd(), path), 'utf-8')
}

const theme = source('src/theme/vtech.css')
const logo = source('src/components/LogoMark.vue')

describe('the logo colours', () => {
  it('are #0ff0ff to #803cff, as plain values with no mode in between', () => {
    expect(theme).toMatch(/--logo-from:\s*#0ff0ff;/)
    expect(theme).toMatch(/--logo-to:\s*#803cff;/)
    expect(theme.match(/--logo-from:/g)).toHaveLength(1)
    expect(theme.match(/--logo-to:/g)).toHaveLength(1)
    expect(theme).not.toMatch(/--logo-(from|to):\s*light-dark/)
  })

  it('are the favicon values', () => {
    const favicon = source('public/favicon.svg').toLowerCase()
    expect(favicon).toContain('#0ff0ff')
    expect(favicon).toContain('#803cff')
  })

  it('are what the mark takes its stops from, and nothing from the accent roles', () => {
    expect(logo).toMatch(/\.logo-mark-stop-start\s*\{\s*stop-color:\s*var\(--logo-from\);/)
    expect(logo).toMatch(/\.logo-mark-stop-end\s*\{\s*stop-color:\s*var\(--logo-to\);/)
    expect(logo).not.toMatch(/stop-color:\s*var\(--color-/)
  })
})
