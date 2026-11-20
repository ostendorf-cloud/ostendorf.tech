/**
 * The favicon links and the social preview image of every prerendered page.
 *
 * The three icon files follow govue's favicon contract (its Go package
 * "favicon" fixes the names): web/public/favicon.svg, favicon-32x32.png and
 * apple-touch-icon.png, served from the site root. The links are root
 * absolute so a page in either language resolves them the same way.
 * og-image.png is the 1200 by 630 preview. Every file is generated from
 * the SVG sources by "task favicon:generate"; see AGENTS.md.
 *
 * The favicon is a dark rounded square with a bright mark, so it reads on
 * a light and on a dark tab bar alike without a colour scheme switch.
 */

export const SOCIAL_IMAGE_PATH = '/og-image.png'

/** iconLinks is the icon half of the head, one tag per line. */
export function iconLinks(): string[] {
  return [
    '<link rel="icon" href="/favicon.svg" type="image/svg+xml">',
    '<link rel="icon" href="/favicon-32x32.png" type="image/png" sizes="32x32">',
    '<link rel="apple-touch-icon" href="/apple-touch-icon.png">',
  ]
}

/** socialImageMeta is the og:image and twitter:image half of the head, with
 *  an absolute URL built from siteUrl, the configured site origin. */
export function socialImageMeta(siteUrl: string): string[] {
  const url = siteUrl + SOCIAL_IMAGE_PATH
  return [
    `<meta property="og:image" content="${url}">`,
    '<meta property="og:image:width" content="1200">',
    '<meta property="og:image:height" content="630">',
    '<meta name="twitter:card" content="summary_large_image">',
    `<meta name="twitter:image" content="${url}">`,
  ]
}
