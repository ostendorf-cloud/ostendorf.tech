/**
 * Renders every route of every locale to static HTML, so a crawler and a
 * browser's first paint both get real markup instead of an empty <div
 * id="app">. Run through "task prerender", after "task web:build" has
 * written the SPA bundle and its manifest into cmd/website/dist.
 *
 * The Go service embeds the files this script writes into
 * cmd/website/prerendered and serves them directly; the SPA bundle this
 * same page loads then hydrates over the server-rendered markup rather
 * than replacing it, through createSSRApp on both sides.
 */
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { createMemoryHistory } from 'vue-router'

import App from '../src/App.vue'
import { createAppI18n, SUPPORTED_LOCALES, type Locale } from '../src/i18n'
import { HERO_SIZES } from '../src/lib/heroImage'
import { routeTable } from '../src/lib/routeTable'
import {
  canonical,
  fullTitle,
  hreflangAlternates,
  NOT_FOUND,
  ogLocale,
  pageMeta,
  type PageRoute,
} from '../src/lib/seo'
import { attachLocaleSync, createAppRouter, NOT_FOUND_ROUTE_NAME } from '../src/router'

const here = dirname(fileURLToPath(import.meta.url))
const distDir = join(here, '../../cmd/website/dist')
const outDir = join(here, '../../cmd/website/prerendered')

/** loadPages turns the shared route table into this script's own page
 *  list, pairing each entry's name with the router route and the SEO
 *  route it names (the same string serves both now that PageRoute is a
 *  plain string), plus the one page the table itself does not carry:
 *  the 404 page, which answers no route and so is not part of the table
 *  internal/webapp's sitemap and mux registration read either. */
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
  return { script: entry.file, styles: entry.css ?? [], heroSrcset: heroSrcset(manifest) }
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

  const alternates = hreflangAlternates(route)
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
    <meta name="robots" content="index, follow">
    <link rel="canonical" href="${url}">
    ${alternates}
    <meta property="og:type" content="website">
    <meta property="og:locale" content="${ogLocale(locale)}">
    <meta property="og:site_name" content="${escapeHtml(siteName)}">
    <meta property="og:title" content="${escapeHtml(title)}">
    <meta property="og:description" content="${escapeHtml(meta.description)}">
    <meta property="og:url" content="${url}">
    <meta name="theme-color" content="#050a14">
    ${heroPreload}${styleLinks}
    <script type="module" src="/${assets.script}" defer></script>`
}

function renderDocument(locale: Locale, head: string, body: string): string {
  return `<!doctype html>
<html lang="${locale}" data-theme="vtech">
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
}

main().catch((error: unknown) => {
  console.error(error)
  process.exitCode = 1
})
