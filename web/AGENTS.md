# web

`web/` is the Vue SPA, consuming govue's Vue package as a pnpm git dependency. `web/src`
holds the app.

## Contact section

- `web/src/views/HomeView.vue` renders govue's `GovueContactForm` with a cookie free
  client (`createClient({ credentials: 'omit' })`), the site's own consent text
  (`home.contact.consent`), the form heading and the privacy link of the page's language.
  Its fields, errors and success text come from govue's catalogue.
- The site says "Sie": `main.ts` and the prerender script both install `createGovueI18n`
  with `formality: 'formal'`, govue's formal German catalogue.
  `src/i18n/formality.test.ts` fails when either stops doing so.
- Its proof of work Worker is govue's, emitted by Vite as a same-origin file under
  `/assets/` that govue's default `worker-src 'self'` allows.
- `src/views/contactSection.test.ts` proves the section in both languages.

## Header

- `src/components/AppHeader.vue` is govue's `GovueHeader` with the navigation centred
  (`navPlacement="center"`) and the contact link as the `cta` nav item.
- `GovueLanguageSwitch` is in link mode (`linkFor`, `navigate` to the router).
  `GovueThemeSwitch` is in the `#theme` slot. Both are the same at every width: the
  `cycle` button and the `othersOnly` link to the other language, with `hreflang`,
  `lang` and `navigate`. There is no width variant and no wrapper.
- govue puts both after the navigation and the contact link, at the right edge of the
  bar, the theme button first and the language link last.
- Below 480 px govue's header has two rows, the brand row and the navigation row, which
  the owner accepted.
- `src/styles.css` holds the prerender fallback of `--header-height` per breakpoint,
  57 px from 480 px and 109 px below, read from govue's header classes.
  `AppHeader.test.ts` derives both from those classes and fails when the stylesheet or
  govue moves.
- Its translucent ground is govue's `--color-header-bg` and `--header-backdrop` tokens in
  `src/theme/vtech.css`.
- On the home page it is govue's `overlay` header, transparent over the hero until the
  page scrolls. That is an exception to govue's documented hero bound: the brightest hero
  ground is about 0.0638 in dark mode, above govue's 0.05 and above the bound with
  `text-muted`, which no header part paints. `src/theme/contrast.test.ts` guards the
  site's own bound over the roles the header paints.
- In light mode the hero is pale, so `src/styles.css` gives the transparent header the
  light roles and the navy focus ring. The hero itself is unchanged apart from filling
  the window under the header.

## Routes

- `scripts/prerender.ts` is the static site generation step `task prerender` runs.
- `src/lib/routeTable.ts` is the one place the frontend reads `internal/routes/routes.json`
  from. `scripts/prerender.ts`, `src/router/index.ts` (the Vue Router table) and
  `src/lib/seo.ts` (the `PageRoute` list and its slugs) all import it instead of
  restating the page list.

## Generated files

These are generated and never hand edited. Sources, commands and sizes are in
`docs/generated-assets.md`.

- `src/assets/logo-wordmark-light.svg` and `logo-wordmark-dark.svg`. The light variant
  carries the value of `--color-text` (`#0f1a2e`), not a reference: regenerate it when
  that token moves.
- `src/assets/hero-*` and `portrait-*` (`avif`, `webp`, `jpg`). Never re-cropped. The
  build resizes nothing, on purpose. `src/lib/heroImage.ts` lists the variants and
  `src/lib/portraitImage.test.ts` fails when a declared width differs from the pixels.
- `public/favicon-32x32.png`, `apple-touch-icon.png` and `og-image.png`, from
  `public/favicon.svg` through `task favicon:generate`. `public/favicon.svg` is the one
  hand kept source.
- `src/assets/bands/*.webp`. Sources are not committed, `.gitignore` keeps them out.
- `LogoMark.vue` is not generated: inline markup whose gradient reads `--logo-from` and
  `--logo-to`, so the theme never forces a regeneration.
