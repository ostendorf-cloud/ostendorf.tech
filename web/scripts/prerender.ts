/**
 * Renders every route of every locale to static HTML, so a crawler and a
 * browser's first paint both get real markup instead of an empty <div
 * id="app">. Run through "task prerender", after "task web:build" has
 * written the SPA bundle and its manifest into cmd/frontend/dist.
 *
 * The pages, the 404 page per language and sitemap.xml are written into
 * that same directory, the layout govue's frontend server reads: Vite's
 * dist with the prerender written into it. cmd/frontend embeds it. The SPA
 * bundle this same page loads then hydrates over the server-rendered markup
 * rather than replacing it, through createSSRApp on both sides.
 *
 * The head carries no favicon link, no theme-color tag and no theme boot
 * script: govue's frontend server injects all three at startup and refuses
 * a page that carries one of them already.
 */
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { createMemoryHistory } from 'vue-router'
import { createGovueI18n, sitemapXml, useTheme } from '@ostendorf-cloud/govue'

import App from '../src/App.vue'
import { createAppI18n, SUPPORTED_LOCALES, type Locale } from '../src/i18n'
import { HERO_SIZES } from '../src/lib/heroImage'
import { socialImageMeta } from '../src/lib/icons'
import { siteRouteDefs } from '../src/lib/legal'
import { routeTable } from '../src/lib/routeTable'
import {
  canonical,
  fullTitle,
  hreflangAlternates,
  isIndexable,
  NOT_FOUND,
  SITE_URL,
  ogLocale,
  pageMeta,
  robotsContent,
  type PageRoute,
} from '../src/lib/seo'
import { DEFAULT_MODE, STORAGE_KEY, THEME_NAME } from '../src/lib/theme'
import { attachLocaleSync, createAppRouter, NOT_FOUND_ROUTE_NAME } from '../src/router'

const here = dirname(fileURLToPath(import.meta.url))
// Vite's build and the prerender share one directory, govue's frontend
// build layout.
const distDir = join(here, '../../cmd/frontend/dist')
const outDir = distDir

// The same call main.ts makes, for the same two values, before anything
// is rendered. useTheme holds one shared state per process and the header
// reads it to draw the theme switch, so without this the switch would be
// prerendered on "system" and then jump to "dark" the moment the bundle
// hydrated. There is no document here,
// so the attribute writing half of useTheme does nothing;
// renderDocument below writes those attributes itself.
useTheme({ theme: THEME_NAME, mode: DEFAULT_MODE, storageKey: STORAGE_KEY })

/** loadPages turns the shared route table into this script's own page
 *  list, pairing each entry's name with the router route and the SEO
 *  route it names (the same string serves both now that PageRoute is a
 *  plain string), plus the one page the table itself does not carry:
 *  the 404 page, which answers no route and so is not part of the
 *  sitemap either. govue's frontend server serves it as <code>/404.html
 *  for every unknown path. */
function loadPages(): Array<{ routerName: string; seoRoute: PageRoute; file: string }> {
  const pages = routeTable.map((entry) => ({
    routerName: entry.name,
    seoRoute: entry.name,
    file: entry.file,
  }))
  pages.push({ routerName: NOT_FOUND_ROUTE_NAME, seoRoute: NOT_FOUND, file: '404.html' })
  return pages
}

interface ManifestEntry {
  file: string
  css?: string[]
}

interface Assets {
  script: string
  styles: string[]
  heroSrcset: string
}

/** The hero photograph's AVIF variants, keyed in the manifest by the
 *  source file each one was built from. Reading them back out of the
 *  manifest rather than importing them is what keeps the preload pointing
 *  at the same content hashed files the page itself asks for: this script
 *  is built separately from the bundle and would otherwise hash them a
 *  second time, under different names. */
const HERO_VARIANT = /^src\/assets\/hero-(\d+)\.avif$/

function heroSrcset(manifest: Record<string, ManifestEntry>): string {
  const variants = Object.entries(manifest)
    .map(([key, entry]) => {
      const match = HERO_VARIANT.exec(key)
      return match ? { width: Number(match[1]), file: entry.file } : undefined
    })
    .filter((variant) => variant !== undefined)
    .sort((a, b) => a.width - b.width)
  if (variants.length === 0) {
    throw new Error('prerender: the vite manifest carries no hero AVIF variant to preload')
  }
  return variants.map((variant) => `/${variant.file} ${variant.width}w`).join(', ')
}

/** stylesheets is every stylesheet the build wrote, whatever shape the
 *  manifest records it in: as a chunk's own "css" list, or as a
 *  standalone entry when the build does not split CSS at all.
 *
 *  All of them, not the entry's alone. A prerendered page is served as
 *  static HTML and the module graph that would pull a route chunk's own
 *  stylesheet in runs after first paint, or never at all for a visitor
 *  with no JavaScript. Linking only the entry's CSS meant a view with a
 *  <style> block of its own painted unstyled until then; linking all of
 *  them is correct for any view added later and for either setting of
 *  build.cssCodeSplit, rather than only for the one this repository
 *  happens to use. */
function stylesheets(manifest: Record<string, ManifestEntry>): string[] {
  const entry = manifest['index.html']
  const found = new Set<string>(entry?.css ?? [])
  for (const record of Object.values(manifest)) {
    for (const file of record.css ?? []) {
      found.add(file)
    }
    if (record.file.endsWith('.css')) {
      found.add(record.file)
    }
  }
  if (found.size === 0) {
    throw new Error('prerender: the vite manifest records no stylesheet, run task web:build first')
  }
  // The entry's own sheet first, so a split build still cascades from the
  // base styles outwards; another reason this repository does not split.
  return [...found]
}

async function loadManifest(): Promise<Assets> {
  const raw = await readFile(join(distDir, '.vite/manifest.json'), 'utf-8')
  const manifest = JSON.parse(raw) as Record<string, ManifestEntry>
  // The manifest keys its one HTML entry by the input file vite built,
  // index.html, not by the script tag inside it: that script's own
  // bundle is this entry's "file".
  const entry = manifest['index.html']
  if (!entry) {
    throw new Error('prerender: no index.html entry in the vite manifest, run task web:build first')
  }
  return { script: entry.file, styles: stylesheets(manifest), heroSrcset: heroSrcset(manifest) }
}

/** renderBody server-renders the app at routerName, under locale, with a
 *  fresh router and i18n instance per page: nothing from one page's
 *  render may leak into the next. */
async function renderBody(locale: Locale, routerName: string): Promise<string> {
  const i18n = createAppI18n(locale)
  const router = createAppRouter(createMemoryHistory())
  attachLocaleSync(router, i18n.global.locale)

  const app = createSSRApp(App)
  app.use(router)
  app.use(i18n)
  // The same language layer main.ts installs, so the footer's legal links
  // and the legal page shell render in this page's language, with govue's
  // formal German, the same as in the browser.
  app.use(createGovueI18n({ i18n, locale, formality: 'formal' }))

  await router.push({ name: routerName, params: { locale } })
  await router.isReady()

  return renderToString(app)
}

function renderHead(locale: Locale, route: PageRoute, assets: Assets): string {
  const i18n = createAppI18n(locale)
  const t = i18n.global.t
  const meta = pageMeta(route, t)
  const siteName = t('common.siteName')
  const title = fullTitle(route, meta, siteName)
  const url = canonical(locale, route)

  const indexable = isIndexable(route)
  // The 404 page gets no canonical, no hreflang pair and no og:url: see isIndexable.
  const canonicalLink = indexable ? `<link rel="canonical" href="${url}">\n    ` : ''
  const ogUrl = indexable ? `<meta property="og:url" content="${url}">\n    ` : ''
  const alternates = (indexable ? hreflangAlternates(route) : [])
    .map((a) => `<link rel="alternate" hreflang="${a.hreflang}" href="${a.href}">`)
    .join('\n    ')
  const styleLinks = assets.styles
    .map((href) => `<link rel="stylesheet" href="/${href}">`)
    .join('\n    ')

  // The home page's hero photograph is its Largest Contentful Paint
  // element, and nothing above it in the document mentions the picture, so
  // without this the browser only learns about it once the stylesheet and
  // the markup are both in. No other page carries it.
  const heroPreload =
    route === 'home'
      ? `<link rel="preload" as="image" type="image/avif" fetchpriority="high" imagesrcset="${assets.heroSrcset}" imagesizes="${HERO_SIZES}">\n    `
      : ''

  return `<meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>${escapeHtml(title)}</title>
    <meta name="description" content="${escapeHtml(meta.description)}">
    <meta name="robots" content="${robotsContent(route)}">
    ${canonicalLink}${alternates}
    <meta property="og:type" content="website">
    <meta property="og:locale" content="${ogLocale(locale)}">
    <meta property="og:site_name" content="${escapeHtml(siteName)}">
    <meta property="og:title" content="${escapeHtml(title)}">
    <meta property="og:description" content="${escapeHtml(meta.description)}">
    ${ogUrl}${socialImageMeta(SITE_URL, escapeHtml(t('common.socialImageAlt'))).join('\n    ')}
    ${heroPreload}${styleLinks}
    <script type="module" src="/${assets.script}" defer></script>`
}

/** renderDocument wraps one rendered page.
 *
 *  data-theme and data-mode are the two attributes govue's stylesheets
 *  switch on, and they are written here rather than left to useTheme so
 *  the very first paint of a prerendered page is already the right one: an
 *  absent data-mode is useTheme's "follow the operating system" state, so
 *  without it a visitor on a light system would see a light page until the
 *  bundle loaded and then watch it turn dark. Both values come from
 *  src/lib/theme.ts, the same module main.ts starts useTheme from.
 *
 *  govue's theme boot script, which its frontend server injects into the
 *  head, then applies a stored choice before the first paint. */
function renderDocument(locale: Locale, head: string, body: string): string {
  return `<!doctype html>
<html lang="${locale}" data-theme="${THEME_NAME}" data-mode="${DEFAULT_MODE}">
  <head>
    ${head}
  </head>
  <body>
    <div id="app">${body}</div>
  </body>
</html>
`
}

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

async function main(): Promise<void> {
  const assets = await loadManifest()
  const pages = loadPages()

  for (const locale of SUPPORTED_LOCALES) {
    for (const page of pages) {
      const body = await renderBody(locale, page.routerName)
      const head = renderHead(locale, page.seoRoute, assets)
      const html = renderDocument(locale, head, body)

      const target = join(outDir, locale, page.file)
      await mkdir(dirname(target), { recursive: true })
      await writeFile(target, html, 'utf-8')
      console.log(`wrote ${target}`)
    }
  }

  // govue's sitemap of the public routes, next to the locale directories.
  // govue-check-legal reads it, and govue's frontend server serves it as
  // built, registers one page per loc and writes robots.txt from its origin.
  // The origin is SITE_URL at build time (VITE_SITE_URL), so the image
  // carries one production origin.
  const sitemap = join(outDir, 'sitemap.xml')
  await writeFile(sitemap, sitemapXml(siteRouteDefs, SITE_URL), 'utf-8')
  console.log(`wrote ${sitemap}`)
}

main().catch((error: unknown) => {
  console.error(error)
  process.exitCode = 1
})
