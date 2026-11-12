import type { Locale } from '../i18n'
import { SUPPORTED_LOCALES } from '../i18n'

/** The site's canonical origin. Overridable through VITE_SITE_URL for a
 *  staging build; the default matches the Go service's own default. */
export const SITE_URL = import.meta.env.VITE_SITE_URL ?? 'https://ostendorf.tech'

/** PageRoute is every route this milestone prerenders. */
export type PageRoute = 'home' | 'imprint' | 'privacy' | 'agb' | 'notFound'

/** slugs maps a route to its path segment, relative to one locale's own
 *  root. The home page has no segment of its own. */
const slugs: Record<PageRoute, string> = {
  home: '',
  imprint: 'imprint/',
  privacy: 'privacy/',
  agb: 'agb/',
  notFound: '',
}

/** path is the absolute path of route in locale, always starting and
 *  ending with a slash. */
export function path(locale: Locale, route: PageRoute): string {
  return `/${locale}/${slugs[route]}`
}

/** canonical is the absolute URL of route in locale. */
export function canonical(locale: Locale, route: PageRoute): string {
  return SITE_URL + path(locale, route)
}

/** ogLocale is the Open Graph og:locale value for locale. */
export function ogLocale(locale: Locale): string {
  return locale === 'de' ? 'de_DE' : 'en_US'
}

/**
 * PageMeta is translate, a vue-i18n t function bound to one locale, for
 * both a mounted component's own useI18n().t and the prerender script's
 * standalone instance.
 */
export interface Translate {
  (key: string): string
}

export interface PageMeta {
  title: string
  description: string
}

/** titleKeys and descriptionKeys name the i18n keys each route's title and
 *  meta description come from. */
const titleKeys: Record<PageRoute, string> = {
  home: 'home.meta.title',
  imprint: 'legal.imprint.title',
  privacy: 'legal.privacy.title',
  agb: 'legal.agb.title',
  notFound: 'notFound.title',
}

const descriptionKeys: Record<PageRoute, string> = {
  home: 'home.meta.description',
  imprint: 'legal.imprint.metaDescription',
  privacy: 'legal.privacy.metaDescription',
  agb: 'legal.agb.metaDescription',
  notFound: 'notFound.lead',
}

/** pageMeta resolves the title and meta description of route through t. */
export function pageMeta(route: PageRoute, t: Translate): PageMeta {
  return { title: t(titleKeys[route]), description: t(descriptionKeys[route]) }
}

/** fullTitle is the browser tab title: the page's own title for the home
 *  page, and "<title> – <site name>" for every other page, matching the
 *  old site's convention. */
export function fullTitle(route: PageRoute, meta: PageMeta, siteName: string): string {
  return route === 'home' ? `${siteName} – ${meta.title}` : `${meta.title} – ${siteName}`
}

/** hreflangAlternates lists every locale's canonical URL for route, plus
 *  the x-default entry pointing at German, this site's own default when a
 *  visitor's language is not one this site has a dedicated version for. */
export function hreflangAlternates(route: PageRoute): Array<{ hreflang: string; href: string }> {
  const alternates: Array<{ hreflang: string; href: string }> = SUPPORTED_LOCALES.map((locale) => ({
    hreflang: locale,
    href: canonical(locale, route),
  }))
  alternates.push({ hreflang: 'x-default', href: canonical('de', route) })
  return alternates
}
