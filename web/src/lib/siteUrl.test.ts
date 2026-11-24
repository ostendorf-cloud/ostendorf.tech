import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

import { DEFAULT_SITE_URL, resolveSiteUrl } from './siteUrl'

describe('resolveSiteUrl', () => {
  it('takes the production origin when VITE_SITE_URL is not set', () => {
    expect(resolveSiteUrl(undefined)).toBe(DEFAULT_SITE_URL)
    expect(DEFAULT_SITE_URL).toBe('https://ostendorf.tech')
  })

  it('takes a set origin', () => {
    expect(resolveSiteUrl('https://staging.example.test')).toBe('https://staging.example.test')
  })

  it('fails for a set but empty value, which would write no origin', () => {
    expect(() => resolveSiteUrl('')).toThrow(/VITE_SITE_URL is set but empty/)
    expect(() => resolveSiteUrl('  ')).toThrow(/VITE_SITE_URL is set but empty/)
  })

  // The build, the bundle and the prerender all go through it: vite.config.ts
  // before anything is built, seo.ts for SITE_URL, which the prerender uses
  // for the pages, the sitemap and, through it, robots.txt.
  it('is what the build config and seo.ts resolve the origin with', () => {
    const config = readFileSync(join(process.cwd(), 'vite.config.ts'), 'utf-8')
    expect(config).toMatch(
      /resolveSiteUrl\(loadEnv\(mode, process\.cwd\(\), 'VITE_'\)\.VITE_SITE_URL\)/,
    )
    const seo = readFileSync(join(process.cwd(), 'src/lib/seo.ts'), 'utf-8')
    expect(seo).toMatch(/SITE_URL = resolveSiteUrl\(import\.meta\.env\.VITE_SITE_URL\)/)
  })
})
