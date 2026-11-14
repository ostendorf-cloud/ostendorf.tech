import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

import { themeColors } from './themeColor'

// Resolved against the Vitest process's own working directory, for the
// same reason src/theme/contrast.test.ts gives.
const theme = readFileSync(join(process.cwd(), 'src/theme/vtech.css'), 'utf-8')

describe('themeColors', () => {
  it('reads both ends of the background token', () => {
    const css = ":root[data-theme='x'] { --color-bg: light-dark(#f4f7fd, #050a14); }"

    expect(themeColors(css)).toEqual({ light: '#f4f7fd', dark: '#050a14' })
  })

  it('refuses a theme whose background is one colour for both modes', () => {
    const css = ":root[data-theme='x'] { --color-bg: #050a14; }"

    expect(() => themeColors(css)).toThrow(/--color-bg/)
  })

  it('refuses a theme that declares no background at all', () => {
    expect(() => themeColors(':root { --color-text: light-dark(#000000, #ffffff); }')).toThrow(
      /--color-bg/,
    )
  })

  it('finds two different colours in the theme this site ships', () => {
    const { light, dark } = themeColors(theme)

    expect(light).toMatch(/^#[0-9a-f]{6}$/)
    expect(dark).toMatch(/^#[0-9a-f]{6}$/)
    expect(light).not.toBe(dark)
  })
})
