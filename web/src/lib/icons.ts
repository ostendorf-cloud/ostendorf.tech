/**
 * The social preview image of every prerendered page.
 *
 * The favicon links and the theme-color tags are not written here: govue's
 * frontend server injects them into every page at startup, from the three
 * favicon files at the root of the build (govue's favicon contract:
 * web/public/favicon.svg, favicon-32x32.png and apple-touch-icon.png), and
 * refuses a page that carries them already. og-image.png is the 1200 by 630
 * preview, served as a public file. Every file is generated from the SVG
 * sources by "task favicon:generate"; see AGENTS.md.
 */

export const SOCIAL_IMAGE_PATH = '/og-image.png'

/** socialImageMeta is the og:image and twitter:image half of the head, with
 *  an absolute URL built from siteUrl, the configured site origin. alt is
 *  the page language's description of the image (common.socialImageAlt),
 *  already escaped for an attribute by the caller. */
export function socialImageMeta(siteUrl: string, alt: string): string[] {
  const url = siteUrl + SOCIAL_IMAGE_PATH
  return [
    `<meta property="og:image" content="${url}">`,
    '<meta property="og:image:width" content="1200">',
    '<meta property="og:image:height" content="630">',
    `<meta property="og:image:alt" content="${alt}">`,
    '<meta name="twitter:card" content="summary_large_image">',
    `<meta name="twitter:image" content="${url}">`,
    `<meta name="twitter:image:alt" content="${alt}">`,
  ]
}
