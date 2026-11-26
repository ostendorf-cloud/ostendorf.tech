# cmd/frontend

`cmd/frontend` is govue's `frontend` package and nothing else of its own. It embeds
`cmd/frontend/dist`, Vite's build with the prerendered pages, the 404 page per language
and `sitemap.xml` written into it, and calls `frontend.New` and `Run`.

`embed.go` sits under `cmd/` rather than `internal/` because `go:embed` resolves relative
to the file holding the directive.

## What govue does for it

- It serves every `loc` of the sitemap at its exact path, the slash forms with 301, the
  assets, the three favicon files and `og-image.png` (`PublicFiles`), robots.txt from the
  sitemap's origin and the language redirect at `/`.
- `language.HandleLegal` adds the aliases `/imprint`, `/privacy`, `/terms`, `/impressum`,
  `/datenschutz`, `/datenschutzerklaerung` and `/agb`.
- `/de/agb` and `/en/agb`, with and without the slash, answer 301 to the terms page
  through `Config.Redirects`.
- It injects the favicon links, one `theme-color` tag per mode (`ThemeColor` `#f6f5fd`,
  `ThemeColorDark` `#07060f`, the two ends of `--color-bg`) and its theme boot script
  with `themeboot.WithDefaultMode(themeboot.ModeDark)`. It adds the script's hash to
  `script-src` and refuses a page that carries any of them already.
- The page rate limit keeps an HTML answer in the page's language.

## Rules

- The frontend holds no secret (`internal/frontendconfig` reads none) and serves no API.
  `/api/` answers 404, unless `FRONTEND_DEV_API_PROXY` is set.
- `FRONTEND_DEV_API_PROXY` is the development switch that turns on `Config.Development`
  and forwards `/api/` to the API container. Only `docker-compose.yaml` sets it, the
  chart never.
- The sitemap and robots.txt carry the origin the image was built with (`VITE_SITE_URL`,
  default `https://ostendorf.tech`), so an image serves one production origin.
- `internal/headertest` holds the security headers the single image sent before the
  split. The frontend's tests compare against it, with only the nonce and the boot
  script hash dropped from the policy.

## Tests

- `build_test.go` runs the real prerender output: every route of `internal/routes` has a
  page and every page answers a route (`TestPrerenderedOutputMatchesRouteTable`,
  `task test:routes`, its own CI job).
- It also checks that every page pins `data-mode="dark"`, the indexing tags, no head
  markup govue writes, and govue's `favicon.CheckPages` over the served pages.
- `site_test.go` feeds the real build to `frontend.New` and checks the headers, the
  redirects, the counting and the dev proxy.

## Umami

- Page views are reported to Umami from the frontend server by govue's `analytics`
  package, through `frontend.Config.Analytics`. The site has no analytics code of its own.
- `internal/frontendconfig` reads it through `analytics.LoadConfig` (`UMAMI_URL`,
  `UMAMI_WEBSITE_ID`, `UMAMI_TIMEOUT`, `UMAMI_QUEUE_SIZE`).
- `site.go` builds the client with the trusted proxy resolver (`TRUSTED_PROXIES`, chart
  value `config.trustedProxies`), `FullIP` off and the host of `SITE_URL`.
- govue places the counter around its mux, registers `analytics_views_dropped_total` and
  closes the client after the listeners stopped.
- Only a page answered with 200, or revalidated with 304, counts, by its pattern and
  without the query. Redirects, the legal aliases, 404s, assets, public files,
  robots.txt and the sitemap count nothing. The API counts nothing.
- `site_test.go` proves it through the real build.
- What is counted is stated in `web/src/views/PrivacyView.vue`. Change that text
  whenever the payload changes, which means a govue upgrade that touches the analytics
  privacy checklist.
