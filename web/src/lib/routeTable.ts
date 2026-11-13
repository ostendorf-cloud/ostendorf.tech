import rawTable from '../../../internal/routes/routes.json'

/** RouteEntry is one row of internal/routes/routes.json, the site's one
 *  route table: see internal/routes/routes.go for the Go side's own copy
 *  of this shape and who else reads the same file. */
export interface RouteEntry {
  name: string
  slug: string
  file: string
}

/** routeTable is the checked-in route table, read at build time so the
 *  browser bundle, the prerender script, the Vue router and the SEO
 *  metadata all carry the exact same list without restating it. */
export const routeTable: RouteEntry[] = rawTable
