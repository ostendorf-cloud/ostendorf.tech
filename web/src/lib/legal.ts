import { legalRouteDefs, type LegalPage, type RouteDef } from '@ostendorf-cloud/govue'

/** The legal pages this site has, in footer order. The site sells services
 *  and has terms (the AGB page), so all three of govue's pages are here.
 *  The Go side sets Terms in internal/webapp's legalConfig for the same
 *  reason. */
export const SITE_LEGAL_PAGES: readonly LegalPage[] = ['imprint', 'privacy', 'terms']

/** The public routes govue's sitemap and the legal check are built from:
 *  the home page and the legal pages. legal.test.ts proves this list and
 *  internal/routes/routes.json name the same pages. */
export const siteRouteDefs: RouteDef[] = [
  { path: '/', public: true },
  ...legalRouteDefs({ terms: true }),
]
