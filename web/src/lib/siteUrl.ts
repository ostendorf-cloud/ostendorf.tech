/**
 * The site's public origin, which the prerendered pages, sitemap.xml and
 * robots.txt carry. It is fixed at build time from VITE_SITE_URL, so one
 * frontend image serves one production origin.
 *
 * Pure TypeScript with no Vite import: vite.config.ts checks the value with
 * the same function before the build starts, and seo.ts resolves it for the
 * bundle and the prerender.
 */

/** The production origin, used when VITE_SITE_URL is not set at all. */
export const DEFAULT_SITE_URL = 'https://ostendorf.tech'

/**
 * resolveSiteUrl returns the origin for a VITE_SITE_URL value. Unset means
 * the production origin. Set but empty, or white space only, is a mistake
 * in the build environment and throws: an empty origin would write relative
 * canonical links, an empty sitemap origin and a robots.txt that names no
 * sitemap.
 */
export function resolveSiteUrl(raw: string | undefined): string {
  if (raw === undefined) {
    return DEFAULT_SITE_URL
  }
  if (raw.trim() === '') {
    throw new Error(
      'VITE_SITE_URL is set but empty: unset it for the production origin, or set the origin',
    )
  }
  return raw
}
