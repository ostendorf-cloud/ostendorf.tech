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
import { createGovueI18n, sitemapXml, useTheme } from '@ostendorf-cloud/govue'

import App from '../src/App.vue'
import { createAppI18n, SUPPORTED_LOCALES, type Locale } from '../src/i18n'
import { HERO_SIZES } from '../src/lib/heroImage'
import { iconLinks, socialImageMeta } from '../src/lib/icons'
import { siteRouteDefs } from '../src/lib/legal'
import { routeTable } from '../src/lib/routeTable'
import {
  canonical,
  fullTitle,
  hreflangAlternates,
  NOT_FOUND,
  SITE_URL,
  ogLocale,
  pageMeta,
  type PageRoute,
} from '../src/lib/seo'
import { DEFAULT_MODE, STORAGE_KEY, THEME_NAME } from '../src/lib/theme'
import { themeColors } from '../src/lib/themeColor'
import { attachLocaleSync, createAppRouter, NOT_FOUND_ROUTE_NAME } from '../src/router'
// Inlined into this script's own bundle at build time, so the theme's
// colours reach the <head> without a second copy of them living here and
// without a path to the stylesheet that has to survive the SSR build's
// output directory.
import vtechCss from '../src/theme/vtech.css?raw'

const here = dirname(fileURLToPath(import.meta.url))
const distDir = join(here, '../../cmd/website/dist')
const outDir = join(here, '../../cmd/website/prerendered')

// The same call main.ts makes, for the same two values, before anything
// is rendered. useTheme holds one shared state per process and the header
// reads it to draw the theme toggle, so without this the toggle would be
// prerendered as the sun with "switch to light mode" on it and then flip
// to the moon the moment the bundle hydrated. There is no document here,
// so the attribute writing half of useTheme does nothing;
// renderDocument below writes those attributes itself.
useTheme({ theme: THEME_NAME, mode: DEFAULT_MODE, storageKey: STORAGE_KEY })

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

/** The browser chrome around the page, read out of the theme's own
 *  --color-bg at the mode the document itself ships in.
 *
 *  One meta, with no media query. There used to be two, one per
 *  prefers-color-scheme, because the page followed the visitor's own
 *  colour scheme and a single value would have framed a near-white page
 *  in near-black. The document now pins its mode (see renderDocument
 *  below), so the system setting no longer says anything about what this
 *  page is painted in, and a pair keyed on it would be wrong for exactly
 *  the visitor it used to be right for: someone on a light system reading
 *  a dark page.
 *
 *  That is no longer the whole story, though: a visitor who has stored a
 *  light or system preference is no longer left to keep this chrome
 *  colour as their one correction, the way the comment above used to
 *  read. themeBootScript below corrects data-mode before first paint, and
 *  the same stored preference means this static meta can be wrong too, so
 *  it corrects this tag's content as well, right after writing it. A
 *  resolved-light outcome needs the light colour; a resolved-dark one
 *  needs no correction, since this default already is the dark colour.
 *  themeColorValues keeps both ends in scope for that reason, rather than
 *  the single value DEFAULT_MODE picks for the attribute path below. */
const themeColorValues = themeColors(vtechCss)
const themeColorMeta = `<meta name="theme-color" content="${themeColorValues[DEFAULT_MODE]}">`

/** themeBootScript is a small, blocking inline script written into every
 *  prerendered <head>, before the stylesheet <link>, so a returning
 *  visitor who stored a light or system preference is not shown a dark
 *  flash: the prerendered markup above always pins data-mode="dark" (see
 *  renderDocument), and the only thing that used to correct that for such
 *  a visitor was main.ts's own useTheme call, which runs only once the
 *  deferred bundle has loaded, strictly after first paint. This script
 *  runs before the stylesheet applies instead, which is also why it has
 *  to sit before the <link rel="stylesheet"> in renderHead rather than
 *  after it or deferred.
 *
 *  It reads localStorage under STORAGE_KEY, the exact key useTheme reads
 *  and writes, rather than a second guess at it: see that constant's own
 *  comment for why three places have to agree on it on purpose. Reading
 *  it the same way useTheme's own restore() does (JSON.parse of
 *  {theme, mode}, mode one of "light"/"dark"/"system") means this script
 *  resolves the exact same choice useTheme would, before useTheme ever
 *  runs.
 *
 *  Every failure mode leaves the prerendered dark default untouched: no
 *  stored value, storage unavailable or throwing, malformed JSON, and an
 *  unrecognised mode value are all caught and ignored, and the whole
 *  script is wrapped in its own try/catch besides, so nothing it does can
 *  ever break the page it runs on.
 *
 *  Resolving "system" needs prefers-color-scheme itself, since there is
 *  no document.documentElement data-mode to read it back from reliably
 *  before this script has run; once resolved, a light outcome also
 *  corrects the theme-color meta above to themeColorValues.light, folding
 *  what would otherwise be a second, separate fix (the same visitor's
 *  stored light preference leaving the chrome colour wrong too) into this
 *  one mechanism.
 *
 *  Its text has to stay byte identical on every page, with no per-page
 *  interpolation beyond the two constants below (the same on every page):
 *  internal/webapp/themeboot.go computes one CSP script-src 'sha256-'
 *  source from whichever prerendered page it finds this script on first,
 *  since this site's CSP carries no 'unsafe-inline' for script-src and
 *  this static HTML has no per-response nonce to carry. The id="theme-boot"
 *  attribute is what that Go code locates the script by. */
const themeBootScript = `<script id="theme-boot">(function(){try{var raw=localStorage.getItem(${JSON.stringify(STORAGE_KEY)});if(!raw){return}var stored=JSON.parse(raw);var mode=stored&&stored.mode;if(mode!=='light'&&mode!=='dark'&&mode!=='system'){return}var root=document.documentElement;var resolved=mode;if(mode==='system'){root.removeAttribute('data-mode');resolved=window.matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light'}else{root.setAttribute('data-mode',mode)}if(resolved==='light'){var meta=document.querySelector('meta[name="theme-color"]');if(meta){meta.setAttribute('content',${JSON.stringify(themeColorValues.light)})}}}catch(e){}})();</script>`

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
  // and the legal page shell render in this page's language.
  app.use(createGovueI18n({ i18n, locale }))

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
    ${socialImageMeta(SITE_URL).join('\n    ')}
    ${iconLinks().join('\n    ')}
    ${themeColorMeta}
    ${themeBootScript}
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
 *  That pinned default is not the only word on the matter any more,
 *  though: a visitor who stored a light or system preference would
 *  otherwise see this dark default flash before main.ts's deferred
 *  useTheme call corrected it. themeBootScript, written into the <head>
 *  by renderHead, corrects data-mode (and the theme-color meta) before
 *  this markup is ever painted, so this function's own pinned default
 *  stays the fail-safe starting point rather than the whole story. */
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

  // The input of govue-check-legal, which reads sitemap.xml next to the
  // locale directories. It is govue's own sitemap of the same public routes.
  // The one the server answers at /sitemap.xml is built in Go from the
  // configured SITE_URL (internal/webapp/sitemap.go) and is not this file;
  // cmd/website's TestRuntimeSitemapMatchesPrerenderedSitemap proves the
  // two list the same paths.
  const sitemap = join(outDir, 'sitemap.xml')
  await writeFile(sitemap, sitemapXml(siteRouteDefs, SITE_URL), 'utf-8')
  console.log(`wrote ${sitemap}`)
}

main().catch((error: unknown) => {
  console.error(error)
  process.exitCode = 1
})
